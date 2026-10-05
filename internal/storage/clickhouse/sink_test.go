package clickhouse

import (
	"strings"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func TestSplitStatementsSkipsComments(t *testing.T) {
	stmts := splitStatements(SchemaDDL)
	if len(stmts) < 5 {
		t.Fatalf("expected the table/view statements from SchemaDDL, got %d", len(stmts))
	}
	for i, stmt := range stmts {
		if strings.HasPrefix(strings.TrimSpace(stmt), "--") {
			t.Fatalf("statement %d is comment-only: %q", i, stmt)
		}
		if !strings.Contains(stmt, "CREATE") {
			t.Fatalf("statement %d is not a CREATE: %q", i, stmt)
		}
	}
}

func TestBuildEventRowMapsDomainEvent(t *testing.T) {
	cost := 1.25
	review := 94.0
	event := domain.TokenEvent{
		EventID:          "evt-1",
		Timestamp:        time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		TenantID:         "tenant-a",
		WorkerID:         "worker-1",
		WorkerName:       "Worker One",
		ModelID:          "model-x",
		Provider:         "demo",
		PromptTokens:     900,
		CompletionTokens: 180,
		CostEstimateUSD:  &cost,
		TaskCategory:     "classification",
		OutputStatus:     domain.OutputAccepted,
		ReviewScore:      &review,
		Fingerprint:      "fp-1",
		Tags:             map[string]string{"profile": "efficient"},
	}

	row := buildEventRow(event)
	if len(row) != len(tokenEventColumns) {
		t.Fatalf("row has %d values for %d columns", len(row), len(tokenEventColumns))
	}
	if row[0] != "evt-1" || row[1] != "tenant-a" || row[2] != "worker-1" {
		t.Fatalf("identity columns wrong: %#v", row[:3])
	}
	if row[3] != "model-x" || row[4] != "classification" {
		t.Fatalf("model/feature columns wrong: %#v", row[3:5])
	}
	if row[7] != uint64(1080) {
		t.Fatalf("total tokens should fall back to prompt+completion, got %#v", row[7])
	}
	if row[8] != 1.25 {
		t.Fatalf("cost column wrong: %#v", row[8])
	}
	if row[10] != "fp-1" {
		t.Fatalf("fingerprint column wrong: %#v", row[10])
	}
	meta, ok := row[11].(map[string]string)
	if !ok {
		t.Fatalf("metadata column should be map[string]string, got %T", row[11])
	}
	for k, want := range map[string]string{
		"profile":       "efficient",
		"provider":      "demo",
		"worker_name":   "Worker One",
		"output_status": "accepted",
		"review_score":  "94",
	} {
		if meta[k] != want {
			t.Fatalf("metadata[%q] = %q, want %q", k, meta[k], want)
		}
	}
}

func TestBuildMetadataOmitsEmptyAndSurvivesDegradedCost(t *testing.T) {
	event := domain.TokenEvent{
		EventID:          "evt-2",
		CostIsDegraded:   true,
		CostDegradedCode: "unknown_pricing",
	}
	meta := buildMetadata(event)
	if _, ok := meta["provider"]; ok {
		t.Fatal("empty provider must be omitted from metadata")
	}
	if meta["cost_degraded_code"] != "unknown_pricing" {
		t.Fatalf("degraded cost code missing: %#v", meta)
	}
}

func TestValidIdentifier(t *testing.T) {
	for name, want := range map[string]bool{
		"tokengoblin":     true,
		"tg_analytics_2":  true,
		"":                false,
		"tokengoblin;drop": false,
		"tokengoblin.test": false,
		"db name":         false,
	} {
		if got := validIdentifier(name); got != want {
			t.Fatalf("validIdentifier(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestZombieRecommendation(t *testing.T) {
	cases := []struct {
		name      string
		rate      float64
		cost      float64
		requests  int64
		wantRec   string
		wantZombi bool
	}{
		{"zero acceptance with spend", 0, 4.2, 6, "quarantine", true},
		{"zero acceptance with UNPRICED tokens", 0, 0, 6, "investigate", true},
		{"low acceptance with spend", 0.15, 2.0, 10, "investigate", true},
		{"nothing at all", 0, 0, 0, "", false},
	}
	for _, tc := range cases {
		rec, ok := zombieRecommendation(tc.rate, tc.cost, tc.requests)
		if ok != tc.wantZombi || rec != tc.wantRec {
			t.Fatalf("%s: got (%q, %v), want (%q, %v)", tc.name, rec, ok, tc.wantRec, tc.wantZombi)
		}
	}
}

func TestEventSinkConfigFromEnvDisabledWhenUnset(t *testing.T) {
	t.Setenv("TG_CLICKHOUSE_ADDR", "")
	if _, ok := EventSinkConfigFromEnv(); ok {
		t.Fatal("telemetry must be disabled when TG_CLICKHOUSE_ADDR is empty")
	}
	t.Setenv("TG_CLICKHOUSE_ADDR", "ch-a:9000,ch-b:9000")
	cfg, ok := EventSinkConfigFromEnv()
	if !ok {
		t.Fatal("telemetry must be enabled when TG_CLICKHOUSE_ADDR is set")
	}
	if len(cfg.Addresses) != 2 || cfg.Addresses[0] != "ch-a:9000" {
		t.Fatalf("address parsing wrong: %#v", cfg.Addresses)
	}
	if cfg.Database != "tokengoblin" {
		t.Fatalf("default database wrong: %q", cfg.Database)
	}
}
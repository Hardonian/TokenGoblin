1|package storage
2|
3|import (
4|	"context"
5|	"encoding/json"
6|	"errors"
7|	"fmt"
8|	"strings"
9|	"time"
10|
11|	"github.com/Hardonian/TokenGoblin/internal/domain"
12|	"github.com/golang-migrate/migrate/v4"
13|	_ "github.com/golang-migrate/migrate/v4/database/postgres"
14|	_ "github.com/golang-migrate/migrate/v4/source/file"
15|	"github.com/jackc/pgx/v5"
16|	"github.com/jackc/pgx/v5/pgxpool"
17|)
18|
19|type PostgresRepository struct {
20|	pool *pgxpool.Pool
21|}
22|
23|func OpenPostgres(ctx context.Context, dsn string) (*PostgresRepository, error) {
24|	if dsn == "" {
25|		return nil, fmt.Errorf("%w: missing database DSN", ErrUnavailable)
26|	}
27|
28|	m, err := migrate.New(
29|		"file://data/migrations",
30|		dsn,
31|	)
32|	if err != nil {
33|		return nil, fmt.Errorf("%w: create migrator: %v", ErrUnavailable, err)
34|	}
35|	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
36|		return nil, fmt.Errorf("%w: migrate up: %v", ErrUnavailable, err)
37|	}
38|
39|	config, err := pgxpool.ParseConfig(dsn)
40|	if err != nil {
41|		return nil, fmt.Errorf("%w: parse dsn: %v", ErrUnavailable, err)
42|	}
43|
44|	config.MaxConns = 25
45|	config.MinConns = 5
46|	config.MaxConnIdleTime = 30 * time.Minute
47|	config.MaxConnLifetime = time.Hour
48|
49|	pool, err := pgxpool.NewWithConfig(ctx, config)
50|	if err != nil {
51|		return nil, fmt.Errorf("%w: connect to postgres: %v", ErrUnavailable, err)
52|	}
53|
54|	if err := pool.Ping(ctx); err != nil {
55|		return nil, fmt.Errorf("%w: ping postgres: %v", ErrUnavailable, err)
56|	}
57|	if err := verifyPostgresRLS(ctx, pool); err != nil {
58|		pool.Close()
59|		return nil, err
60|	}
61|
62|	return &PostgresRepository{pool: pool}, nil
63|}
64|
65|func verifyPostgresRLS(ctx context.Context, pool *pgxpool.Pool) error {
66|	tables := []string{
67|		"tenants",
68|		"workers",
69|		"jobs",
70|		"token_usage_events",
71|		"cost_snapshots",
72|		"anomaly_signals",
73|		"productivity_summaries",
74|		"tenant_pricing_overrides",
75|		"output_analyses",
76|		"tenant_members",
77|		"audit_events",
78|		"recommendation_states",
79|		"api_keys",
80|	}
81|	var missing []string
82|	for _, table := range tables {
83|		var enabled bool
84|		if err := pool.QueryRow(ctx, `SELECT COALESCE((SELECT relrowsecurity FROM pg_class WHERE oid = to_regclass($1)), false)`, table).Scan(&enabled); err != nil {
85|			return fmt.Errorf("%w: verify postgres rls for %s: %v", ErrUnavailable, table, err)
86|		}
87|		if !enabled {
88|			missing = append(missing, table)
89|		}
90|	}
91|	if len(missing) > 0 {
92|		return fmt.Errorf("%w: postgres row level security is not enabled for: %s", ErrUnavailable, strings.Join(missing, ", "))
93|	}
94|	return nil
95|}
96|
97|func (r *PostgresRepository) Close() error {
98|	r.pool.Close()
99|	return nil
100|}
101|
102|func (r *PostgresRepository) Ping(ctx context.Context) error {
103|	return r.pool.Ping(ctx)
104|}
105|
106|func (r *PostgresRepository) UpsertTenant(ctx context.Context, tenant domain.Tenant) error {
107|	_, err := r.pool.Exec(ctx, `
108|		INSERT INTO tenants (tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at)
109|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
110|		ON CONFLICT(tenant_id) DO UPDATE SET
111|			name = EXCLUDED.name,
112|			tier = EXCLUDED.tier,
113|			usage_limit_usd = EXCLUDED.usage_limit_usd,
114|			stripe_customer_id = EXCLUDED.stripe_customer_id,
115|			stripe_subscription_id = EXCLUDED.stripe_subscription_id,
116|			updated_at = EXCLUDED.updated_at
117|	`, tenant.TenantID, tenant.Name, tenant.Tier, tenant.UsageLimitUSD, nullString(tenant.StripeCustomerID), nullString(tenant.StripeSubscriptionID), tenant.CreatedAt, tenant.UpdatedAt)
118|	return wrapDBErr(err)
119|}
120|
121|func (r *PostgresRepository) GetTenant(ctx context.Context, tenantID string) (*domain.Tenant, error) {
122|	var t domain.Tenant
123|	var stripeCust, stripeSub *string
124|	err := r.pool.QueryRow(ctx, `
125|		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
126|		FROM tenants
127|		WHERE tenant_id = $1
128|	`, tenantID).Scan(&t.TenantID, &t.Name, &t.Tier, &t.UsageLimitUSD, &stripeCust, &stripeSub, &t.CreatedAt, &t.UpdatedAt)
129|
130|	if err != nil {
131|		if errors.Is(err, pgx.ErrNoRows) {
132|			return nil, nil
133|		}
134|		return nil, wrapDBErr(err)
135|	}
136|	if stripeCust != nil {
137|		t.StripeCustomerID = *stripeCust
138|	}
139|	if stripeSub != nil {
140|		t.StripeSubscriptionID = *stripeSub
141|	}
142|	return &t, nil
143|}
144|
145|func (r *PostgresRepository) GetTenantCurrentMonthCost(ctx context.Context, tenantID string) (float64, error) {
146|	now := time.Now().UTC()
147|	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
148|
149|	var total *float64
150|	err := r.pool.QueryRow(ctx, `
151|		SELECT SUM(cost_estimate_usd)
152|		FROM token_usage_events
153|		WHERE tenant_id = $1 AND occurred_at >= $2
154|	`, tenantID, startOfMonth).Scan(&total)
155|
156|	if err != nil {
157|		return 0, wrapDBErr(err)
158|	}
159|	if total == nil {
160|		return 0, nil
161|	}
162|	return *total, nil
163|}
164|
165|func (r *PostgresRepository) GetPricingOverride(ctx context.Context, tenantID, provider, modelID string) (*domain.PricePoint, error) {
166|	var point domain.PricePoint
167|	var created time.Time
168|	err := r.pool.QueryRow(ctx, `
169|		SELECT provider, model_id, prompt_price_per_million, completion_price_per_million, created_at
170|		FROM tenant_pricing_overrides
171|		WHERE tenant_id = $1 AND provider = $2 AND model_id = $3
172|	`, tenantID, provider, modelID).Scan(&point.Provider, &point.ModelID, &point.InputCostPerMillion, &point.OutputCostPerMillion, &created)
173|
174|	if err != nil {
175|		if errors.Is(err, pgx.ErrNoRows) {
176|			return nil, nil
177|		}
178|		return nil, wrapDBErr(err)
179|	}
180|	point.Currency = "USD"
181|	point.Source = "override"
182|	point.EffectiveFrom = created
183|	point.CachedInputCostPerMillion = point.InputCostPerMillion / 2.0 // Simple default logic for overrides
184|	return &point, nil
185|}
186|
187|func (r *PostgresRepository) SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error {
188|	overrideID := tenantID + ":" + point.Provider + ":" + point.ModelID
189|	_, err := r.pool.Exec(ctx, `
190|		INSERT INTO tenant_pricing_overrides (override_id, tenant_id, provider, model_id, prompt_price_per_million, completion_price_per_million, created_at)
191|		VALUES ($1, $2, $3, $4, $5, $6, $7)
192|		ON CONFLICT(tenant_id, provider, model_id) DO UPDATE SET
193|			prompt_price_per_million = EXCLUDED.prompt_price_per_million,
194|			completion_price_per_million = EXCLUDED.completion_price_per_million,
195|			created_at = EXCLUDED.created_at
196|	`, overrideID, tenantID, point.Provider, point.ModelID, point.InputCostPerMillion, point.OutputCostPerMillion, time.Now().UTC())
197|	return wrapDBErr(err)
198|}
199|
200|func (r *PostgresRepository) ListPricingOverrides(ctx context.Context, tenantID string) ([]domain.PricePoint, error) {
201|	rows, err := r.pool.Query(ctx, `
202|		SELECT provider, model_id, prompt_price_per_million, completion_price_per_million, created_at
203|		FROM tenant_pricing_overrides
204|		WHERE tenant_id = $1
205|		ORDER BY provider, model_id
206|		ORDER BY created_at DESC
207|	`, tenantID)
208|	if err != nil {
209|		return nil, wrapDBErr(err)
210|	}
211|	defer rows.Close()
212|	var points []domain.PricePoint
213|	for rows.Next() {
214|		var point domain.PricePoint
215|		var created time.Time
216|		if err := rows.Scan(&point.Provider, &point.ModelID, &point.InputCostPerMillion, &point.OutputCostPerMillion, &created); err != nil {
217|			return nil, wrapDBErr(err)
218|		}
219|		point.Currency = "USD"
220|		point.Source = "override"
221|		point.EffectiveFrom = created
222|		point.CachedInputCostPerMillion = point.InputCostPerMillion / 2.0
223|		points = append(points, point)
224|	}
225|	return points, wrapDBErr(rows.Err())
226|}
227|
228|func (r *PostgresRepository) DeleteOldEvents(ctx context.Context, retentionDays int) (int64, error) {
229|	cutoff := time.Now().AddDate(0, 0, -retentionDays).UTC()
230|	res, err := r.pool.Exec(ctx, `DELETE FROM token_usage_events WHERE created_at < $1`, cutoff)
231|	if err != nil {
232|		return 0, wrapDBErr(err)
233|	}
234|	return res.RowsAffected(), nil
235|}
236|
237|func (r *PostgresRepository) DeleteTenantData(ctx context.Context, tenantID string) error {
238|	_, err := r.pool.Exec(ctx, `DELETE FROM tenants WHERE tenant_id = $1`, tenantID)
239|	return wrapDBErr(err)
240|}
241|
242|func (r *PostgresRepository) SaveAPIKey(ctx context.Context, key domain.APIKey) error {
243|	_, err := r.pool.Exec(ctx, `
244|		INSERT INTO api_keys (key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked)
245|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
246|	`, key.KeyID, key.TenantID, key.Name, key.KeyHash, normalizeRole(key.Role), key.CreatedAt, key.LastUsedAt, key.IsRevoked)
247|	return wrapDBErr(err)
248|}
249|
250|func (r *PostgresRepository) GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error) {
251|	var key domain.APIKey
252|	err := r.pool.QueryRow(ctx, `
253|		SELECT key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked
254|		FROM api_keys
255|		WHERE key_id = $1
256|	`, keyID).Scan(&key.KeyID, &key.TenantID, &key.Name, &key.KeyHash, &key.Role, &key.CreatedAt, &key.LastUsedAt, &key.IsRevoked)
257|	if err != nil {
258|		if errors.Is(err, pgx.ErrNoRows) {
259|			return nil, nil
260|		}
261|		return nil, wrapDBErr(err)
262|	}
263|	key.Role = normalizeRole(key.Role)
264|	return &key, nil
265|}
266|
267|func (r *PostgresRepository) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
268|	_, err := r.pool.Exec(ctx, `
269|		UPDATE api_keys
270|		SET last_used_at = $1
271|		WHERE key_id = $2
272|	`, time.Now().UTC(), keyID)
273|	return wrapDBErr(err)
274|}
275|
276|func (r *PostgresRepository) ListAPIKeys(ctx context.Context, tenantID string) ([]domain.APIKey, error) {
277|	rows, err := r.pool.Query(ctx, `
278|		SELECT key_id, tenant_id, name, key_hash, role, created_at, last_used_at, is_revoked
279|		FROM api_keys
280|		WHERE tenant_id = $1 AND is_revoked = false
281|		ORDER BY created_at DESC
282|	`, tenantID)
283|	if err != nil {
284|		return nil, wrapDBErr(err)
285|	}
286|	defer rows.Close()
287|
288|	var keys []domain.APIKey
289|	for rows.Next() {
290|		var key domain.APIKey
291|		if err := rows.Scan(&key.KeyID, &key.TenantID, &key.Name, &key.KeyHash, &key.Role, &key.CreatedAt, &key.LastUsedAt, &key.IsRevoked); err != nil {
292|			return nil, wrapDBErr(err)
293|		}
294|		keys = append(keys, key)
295|	}
296|	return keys, wrapDBErr(rows.Err())
297|}
298|
299|func (r *PostgresRepository) RevokeAPIKey(ctx context.Context, tenantID, keyID string) error {
300|	_, err := r.pool.Exec(ctx, `
301|		UPDATE api_keys
302|		SET is_revoked = true
303|		WHERE tenant_id = $1 AND key_id = $2
304|	`, tenantID, keyID)
305|	return wrapDBErr(err)
306|}
307|
308|func (r *PostgresRepository) UpsertTenantMember(ctx context.Context, member domain.TenantMember) error {
309|	now := member.UpdatedAt
310|	if now.IsZero() {
311|		now = time.Now().UTC()
312|	}
313|	createdAt := member.CreatedAt
314|	if createdAt.IsZero() {
315|		createdAt = now
316|	}
317|	_, err := r.pool.Exec(ctx, `
318|		INSERT INTO tenant_members (tenant_id, subject_id, email, role, created_at, updated_at)
319|		VALUES ($1, $2, $3, $4, $5, $6)
320|		ON CONFLICT(tenant_id, subject_id) DO UPDATE SET
321|			email = EXCLUDED.email,
322|			role = EXCLUDED.role,
323|			updated_at = EXCLUDED.updated_at
324|	`, member.TenantID, member.SubjectID, nullString(member.Email), normalizeRole(member.Role), createdAt, now)
325|	return wrapDBErr(err)
326|}
327|
328|func (r *PostgresRepository) ListTenantMembers(ctx context.Context, tenantID string) ([]domain.TenantMember, error) {
329|	rows, err := r.pool.Query(ctx, `
330|		SELECT tenant_id, subject_id, email, role, created_at, updated_at
331|		FROM tenant_members
332|		WHERE tenant_id = $1
333|		ORDER BY role, subject_id
334|	`, tenantID)
335|	if err != nil {
336|		return nil, wrapDBErr(err)
337|	}
338|	defer rows.Close()
339|	var members []domain.TenantMember
340|	for rows.Next() {
341|		var member domain.TenantMember
342|		var email *string
343|		if err := rows.Scan(&member.TenantID, &member.SubjectID, &email, &member.Role, &member.CreatedAt, &member.UpdatedAt); err != nil {
344|			return nil, wrapDBErr(err)
345|		}
346|		if email != nil {
347|			member.Email = *email
348|		}
349|		member.Role = normalizeRole(member.Role)
350|		members = append(members, member)
351|	}
352|	return members, wrapDBErr(rows.Err())
353|}
354|
355|func (r *PostgresRepository) SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error {
356|	metadataJSON, err := marshalNullable(event.Metadata)
357|	if err != nil {
358|		return err
359|	}
360|	if event.Timestamp.IsZero() {
361|		event.Timestamp = time.Now().UTC()
362|	}
363|	_, err = r.pool.Exec(ctx, `
364|		INSERT INTO audit_events (tenant_id, event_id, type, actor, resource, metadata_json, created_at)
365|		VALUES ($1, $2, $3, $4, $5, $6, $7)
366|	`, event.TenantID, event.EventID, event.Type, event.Actor, nullString(event.Resource), metadataJSON, event.Timestamp)
367|	return wrapDBErr(err)
368|}
369|
370|func (r *PostgresRepository) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error) {
371|	if limit <= 0 || limit > 500 {
372|		limit = 100
373|	}
374|	rows, err := r.pool.Query(ctx, `
375|		SELECT tenant_id, event_id, type, actor, resource, metadata_json, created_at
376|		FROM audit_events
377|		WHERE tenant_id = $1
378|		ORDER BY created_at DESC, event_id DESC
379|		LIMIT $2
380|	`, tenantID, limit)
381|	if err != nil {
382|		return nil, wrapDBErr(err)
383|	}
384|	defer rows.Close()
385|	var events []domain.AuditEvent
386|	for rows.Next() {
387|		var event domain.AuditEvent
388|		var resource *string
389|		var metadataJSON []byte
390|		if err := rows.Scan(&event.TenantID, &event.EventID, &event.Type, &event.Actor, &resource, &metadataJSON, &event.Timestamp); err != nil {
391|			return nil, wrapDBErr(err)
392|		}
393|		if resource != nil {
394|			event.Resource = *resource
395|		}
396|		if len(metadataJSON) > 0 {
397|			_ = json.Unmarshal(metadataJSON, &event.Metadata)
398|		}
399|		events = append(events, event)
400|	}
401|	return events, wrapDBErr(rows.Err())
402|}
403|
404|func (r *PostgresRepository) SetRecommendationState(ctx context.Context, state domain.RecommendationState) error {
405|	now := state.UpdatedAt
406|	if now.IsZero() {
407|		now = time.Now().UTC()
408|	}
409|	createdAt := state.CreatedAt
410|	if createdAt.IsZero() {
411|		createdAt = now
412|	}
413|	_, err := r.pool.Exec(ctx, `
414|		INSERT INTO recommendation_states (tenant_id, recommendation_id, status, actor, note, created_at, updated_at)
415|		VALUES ($1, $2, $3, $4, $5, $6, $7)
416|		ON CONFLICT(tenant_id, recommendation_id) DO UPDATE SET
417|			status = EXCLUDED.status,
418|			actor = EXCLUDED.actor,
419|			note = EXCLUDED.note,
420|			updated_at = EXCLUDED.updated_at
421|	`, state.TenantID, state.RecommendationID, state.Status, nullString(state.Actor), nullString(state.Note), createdAt, now)
422|	return wrapDBErr(err)
423|}
424|
425|func (r *PostgresRepository) ListRecommendationStates(ctx context.Context, tenantID string) ([]domain.RecommendationState, error) {
426|	rows, err := r.pool.Query(ctx, `
427|		SELECT tenant_id, recommendation_id, status, actor, note, created_at, updated_at
428|		FROM recommendation_states
429|		WHERE tenant_id = $1
430|	`, tenantID)
431|	if err != nil {
432|		return nil, wrapDBErr(err)
433|	}
434|	defer rows.Close()
435|	var states []domain.RecommendationState
436|	for rows.Next() {
437|		var state domain.RecommendationState
438|		var actor, note *string
439|		if err := rows.Scan(&state.TenantID, &state.RecommendationID, &state.Status, &actor, &note, &state.CreatedAt, &state.UpdatedAt); err != nil {
440|			return nil, wrapDBErr(err)
441|		}
442|		if actor != nil {
443|			state.Actor = *actor
444|		}
445|		if note != nil {
446|			state.Note = *note
447|		}
448|		states = append(states, state)
449|	}
450|	return states, wrapDBErr(rows.Err())
451|}
452|
453|func (r *PostgresRepository) upsertTenant(ctx context.Context, tx pgx.Tx, tenantID string, now time.Time) error {
454|	_, err := tx.Exec(ctx, `
455|		INSERT INTO tenants (tenant_id, name, created_at, updated_at)
456|		VALUES ($1, $2, $3, $4)
457|		ON CONFLICT(tenant_id) DO UPDATE SET updated_at = EXCLUDED.updated_at
458|	`, tenantID, tenantID, now, now)
459|	return err
460|}
461|
462|func (r *PostgresRepository) upsertWorker(ctx context.Context, tx pgx.Tx, tenantID, workerID, workerName string, now time.Time) error {
463|	_, err := tx.Exec(ctx, `
464|		INSERT INTO workers (tenant_id, worker_id, name, created_at, updated_at)
465|		VALUES ($1, $2, $3, $4, $5)
466|		ON CONFLICT(tenant_id, worker_id) DO UPDATE SET
467|			name = EXCLUDED.name,
468|			updated_at = EXCLUDED.updated_at
469|	`, tenantID, workerID, workerName, now, now)
470|	return err
471|}
472|
473|func (r *PostgresRepository) upsertJob(ctx context.Context, tx pgx.Tx, tenantID, jobID, workerID, taskCategory string, now time.Time) error {
474|	_, err := tx.Exec(ctx, `
475|		INSERT INTO jobs (tenant_id, job_id, worker_id, name, task_category, status, started_at, ended_at, created_at, updated_at)
476|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
477|		ON CONFLICT(tenant_id, job_id) DO UPDATE SET
478|			worker_id = EXCLUDED.worker_id,
479|			task_category = EXCLUDED.task_category,
480|			updated_at = EXCLUDED.updated_at
481|	`, tenantID, jobID, workerID, jobID, taskCategory, "active", nil, nil, now, now)
482|	return err
483|}
484|
485|func (r *PostgresRepository) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
486|	tx, err := r.pool.Begin(ctx)
487|	if err != nil {
488|		return wrapDBErr(err)
489|	}
490|	defer func() { _ = tx.Rollback(ctx) }()
491|
492|	now := event.CreatedAt
493|	if now.IsZero() {
494|		now = time.Now().UTC()
495|	}
496|
497|	if err := r.upsertTenant(ctx, tx, event.TenantID, now); err != nil {
498|		return wrapDBErr(err)
499|	}
500|
501|	workerName := event.WorkerName
502|	if workerName == "" {
503|		workerName = event.WorkerID
504|	}
505|	if err := r.upsertWorker(ctx, tx, event.TenantID, event.WorkerID, workerName, now); err != nil {
506|		return wrapDBErr(err)
507|	}
508|
509|	taskCategory := event.TaskCategory
510|	if taskCategory == "" {
511|		taskCategory = event.TaskType
512|	}
513|	if taskCategory == "" {
514|		taskCategory = "uncategorized"
515|	}
516|	if event.JobID != "" {
517|		if err := r.upsertJob(ctx, tx, event.TenantID, event.JobID, event.WorkerID, taskCategory, now); err != nil {
518|			return wrapDBErr(err)
519|		}
520|	}
521|
522|	var tagsJSON interface{}
523|	if event.TagsJSON != nil {
524|		tagsJSON = string(event.TagsJSON)
525|	} else {
526|		tagsJSON, err = marshalNullable(event.GetTags())
527|	}
528|	if err != nil {
529|		return err
530|	}
531|
532|	var externalCost interface{}
533|	var externalCurrency interface{}
534|	if event.ExternalEstimate != nil {
535|		externalCost = event.ExternalEstimate.CostUSD
536|		externalCurrency = event.ExternalEstimate.Currency
537|	}
538|
539|	_, err = tx.Exec(ctx, `
540|		INSERT INTO token_usage_events (
541|			tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
542|			provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
543|			input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
544|			cost_is_degraded, cost_degraded_code, external_estimate_usd,
545|			external_estimate_currency, latency_ms, task_category, output_status,
546|			review_score, occurred_at, created_at, prompt_excerpt, output_excerpt,
547|			prompt_reference, output_reference, tags_json, idempotency_key, fingerprint
548|		)
549|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34)
550|		ON CONFLICT(tenant_id, event_id) DO UPDATE SET
551|			worker_id = EXCLUDED.worker_id,
552|			worker_name = EXCLUDED.worker_name,
553|			job_id = EXCLUDED.job_id,
554|			session_id = EXCLUDED.session_id,
555|			run_id = EXCLUDED.run_id,
556|			provider = EXCLUDED.provider,
557|			model_id = EXCLUDED.model_id,
558|			prompt_tokens = EXCLUDED.prompt_tokens,
559|			completion_tokens = EXCLUDED.completion_tokens,
560|			cached_tokens = EXCLUDED.cached_tokens,
561|			input_tokens = EXCLUDED.input_tokens,
562|			output_tokens = EXCLUDED.output_tokens,
563|			total_tokens = EXCLUDED.total_tokens,
564|			cost_estimate_usd = EXCLUDED.cost_estimate_usd,
565|			cost_currency = EXCLUDED.cost_currency,
566|			cost_is_degraded = EXCLUDED.cost_is_degraded,
567|			cost_degraded_code = EXCLUDED.cost_degraded_code,
568|			external_estimate_usd = EXCLUDED.external_estimate_usd,
569|			external_estimate_currency = EXCLUDED.external_estimate_currency,
570|			latency_ms = EXCLUDED.latency_ms,
571|			task_category = EXCLUDED.task_category,
572|			output_status = EXCLUDED.output_status,
573|			review_score = EXCLUDED.review_score,
574|			occurred_at = EXCLUDED.occurred_at,
575|			prompt_excerpt = EXCLUDED.prompt_excerpt,
576|			output_excerpt = EXCLUDED.output_excerpt,
577|			prompt_reference = EXCLUDED.prompt_reference,
578|			output_reference = EXCLUDED.output_reference,
579|			tags_json = EXCLUDED.tags_json,
580|			idempotency_key = EXCLUDED.idempotency_key,
581|			fingerprint = EXCLUDED.fingerprint
582|	`, event.TenantID, event.EventID, event.WorkerID, workerName, nullString(event.JobID),
583|		nullString(event.SessionID), nullString(event.RunID), event.Provider, event.ModelID,
584|		event.PromptTokens, event.CompletionTokens, event.CachedTokens, event.InputTokens,
585|		event.OutputTokens, event.TotalTokens, event.CostEstimateUSD, event.CostCurrency,
586|		event.CostIsDegraded, nullString(event.CostDegradedCode), externalCost,
587|		externalCurrency, event.LatencyMs, taskCategory, string(event.OutputStatus),
588|		event.ReviewScore, event.Timestamp, now, nullString(event.PromptExcerpt),
589|		nullString(event.OutputExcerpt), nullString(event.PromptReference), nullString(event.OutputReference),
590|		tagsJSON, nullString(event.IdempotencyKey), nullString(event.Fingerprint))
591|	if err != nil {
592|		return wrapDBErr(err)
593|	}
594|
595|	return wrapDBErr(tx.Commit(ctx))
596|}
597|
598|func (r *PostgresRepository) SaveCostSnapshot(ctx context.Context, snapshot domain.CostSnapshot) error {
599|	_, err := r.pool.Exec(ctx, `
600|		INSERT INTO cost_snapshots (
601|			tenant_id, snapshot_id, event_id, provider, model_id, input_tokens,
602|			output_tokens, cached_tokens, cost_estimate_usd, currency, is_degraded,
603|			degraded_code, created_at
604|		)
605|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
606|		ON CONFLICT(tenant_id, snapshot_id) DO UPDATE SET
607|			cost_estimate_usd = EXCLUDED.cost_estimate_usd,
608|			is_degraded = EXCLUDED.is_degraded,
609|			degraded_code = EXCLUDED.degraded_code,
610|			created_at = EXCLUDED.created_at
611|	`, snapshot.TenantID, snapshot.SnapshotID, snapshot.EventID, snapshot.Provider,
612|		snapshot.ModelID, snapshot.InputTokens, snapshot.OutputTokens, snapshot.CachedTokens,
613|		snapshot.CostEstimateUSD, snapshot.Currency, snapshot.IsDegraded,
614|		nullString(snapshot.DegradedCode), snapshot.CreatedAt)
615|	return wrapDBErr(err)
616|}
617|
618|func (r *PostgresRepository) SaveAnomalySignal(ctx context.Context, signal domain.AnomalySignal) error {
619|	detailsJSON, err := marshalNullable(signal.Details)
620|	if err != nil {
621|		return err
622|	}
623|
624|	_, err = r.pool.Exec(ctx, `
625|		INSERT INTO anomaly_signals (
626|			tenant_id, anomaly_id, event_id, worker_id, detected_at, severity,
627|			type, description, observed_value, threshold_value, details_json, created_at
628|		)
629|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
630|		ON CONFLICT(tenant_id, anomaly_id) DO UPDATE SET
631|			detected_at = EXCLUDED.detected_at,
632|			severity = EXCLUDED.severity,
633|			description = EXCLUDED.description,
634|			observed_value = EXCLUDED.observed_value,
635|			threshold_value = EXCLUDED.threshold_value,
636|			details_json = EXCLUDED.details_json,
637|			created_at = EXCLUDED.created_at
638|	`, signal.TenantID, signal.AnomalyID, nullString(signal.EventID), nullString(signal.WorkerID),
639|		signal.DetectedAt, string(signal.Severity), string(signal.Type),
640|		signal.Description, signal.ObservedValue, signal.ThresholdValue, detailsJSON,
641|		signal.DetectedAt)
642|	return wrapDBErr(err)
643|}
644|
645|func (r *PostgresRepository) SaveOutputAnalysis(ctx context.Context, analysis domain.OutputAnalysis) error {
646|	issuesJSON, err := json.Marshal(analysis.Issues)
647|	if err != nil {
648|		return err
649|	}
650|	recsJSON, err := json.Marshal(analysis.Recommendations)
651|	if err != nil {
652|		return err
653|	}
654|	evidenceJSON, err := json.Marshal(analysis.Evidence)
655|	if err != nil {
656|		return err
657|	}
658|	degradedJSON, err := marshalNullable(analysis.Degraded)
659|	if err != nil {
660|		return err
661|	}
662|	_, err = r.pool.Exec(ctx, `
663|		INSERT INTO output_analyses (
664|			tenant_id, analysis_id, event_id, worker_id, analyzed_at,
665|			efficiency_score, goblin_score, issues_json, recommendations_json,
666|			evidence_json, degraded_json, created_at
667|		)
668|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
669|		ON CONFLICT(tenant_id, analysis_id) DO UPDATE SET
670|			analyzed_at = EXCLUDED.analyzed_at,
671|			efficiency_score = EXCLUDED.efficiency_score,
672|			goblin_score = EXCLUDED.goblin_score,
673|			issues_json = EXCLUDED.issues_json,
674|			recommendations_json = EXCLUDED.recommendations_json,
675|			evidence_json = EXCLUDED.evidence_json,
676|			degraded_json = EXCLUDED.degraded_json
677|	`, analysis.TenantID, analysis.AnalysisID, analysis.EventID, analysis.WorkerID,
678|		analysis.AnalyzedAt, analysis.EfficiencyScore, analysis.GoblinScore,
679|		string(issuesJSON), string(recsJSON), string(evidenceJSON), degradedJSON, time.Now().UTC())
680|	return wrapDBErr(err)
681|}
682|
683|func (r *PostgresRepository) SaveProductivitySummary(ctx context.Context, summary domain.ProductivitySummary) error {
684|	body, err := json.Marshal(summary)
685|	if err != nil {
686|		return err
687|	}
688|
689|	_, err = r.pool.Exec(ctx, `
690|		INSERT INTO productivity_summaries (
691|			tenant_id, summary_id, period_start, period_end, generated_at,
692|			total_cost_usd, total_events, output_count, avg_latency_ms,
693|			anomaly_count, cost_per_accepted_output_with_review, summary_json
694|		)
695|		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
696|		ON CONFLICT(tenant_id, summary_id) DO UPDATE SET
697|			generated_at = EXCLUDED.generated_at,
698|			total_cost_usd = EXCLUDED.total_cost_usd,
699|			total_events = EXCLUDED.total_events,
700|			output_count = EXCLUDED.output_count,
701|			avg_latency_ms = EXCLUDED.avg_latency_ms,
702|			anomaly_count = EXCLUDED.anomaly_count,
703|			cost_per_accepted_output_with_review = EXCLUDED.cost_per_accepted_output_with_review,
704|			summary_json = EXCLUDED.summary_json
705|	`, summary.TenantID, summary.SummaryID, summary.PeriodStart,
706|		summary.PeriodEnd, summary.GeneratedAt, summary.TotalCostUSD,
707|		summary.TotalEvents, summary.OutputCount, summary.AvgLatencyMs, summary.AnomalyCount,
708|		summary.CostPerAcceptedOutputWithReview, string(body))
709|	return wrapDBErr(err)
710|}
711|
712|func (r *PostgresRepository) ListOutputAnalyses(ctx context.Context, tenantID string, limit int) ([]domain.OutputAnalysis, error) {
713|	if limit <= 0 || limit > 500 {
714|		limit = 100
715|	}
716|	rows, err := r.pool.Query(ctx, outputAnalysisSelectPostgres+`
717|		WHERE tenant_id = $1
718|		ORDER BY analyzed_at DESC, analysis_id DESC
719|		LIMIT $2
720|	`, tenantID, limit)
721|	if err != nil {
722|		return nil, wrapDBErr(err)
723|	}
724|	defer rows.Close()
725|	return scanOutputAnalysesPostgres(rows)
726|}
727|
728|func (r *PostgresRepository) ListOutputAnalysesByWorker(ctx context.Context, tenantID, workerID string, limit int) ([]domain.OutputAnalysis, error) {
729|	if limit <= 0 || limit > 500 {
730|		limit = 100
731|	}
732|	rows, err := r.pool.Query(ctx, outputAnalysisSelectPostgres+`
733|		WHERE tenant_id = $1 AND worker_id = $2
734|		ORDER BY analyzed_at DESC, analysis_id DESC
735|		LIMIT $3
736|	`, tenantID, workerID, limit)
737|	if err != nil {
738|		return nil, wrapDBErr(err)
739|	}
740|	defer rows.Close()
741|	return scanOutputAnalysesPostgres(rows)
742|}
743|
744|func (r *PostgresRepository) ListTokenEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error) {
745|	if limit <= 0 || limit > 500 {
746|		limit = 100
747|	}
748|	rows, err := r.pool.Query(ctx, tokenEventSelectPostgres+`
749|		WHERE tenant_id = $1
750|		ORDER BY occurred_at DESC, event_id DESC
751|		LIMIT $2
752|	`, tenantID, limit)
753|	if err != nil {
754|		return nil, wrapDBErr(err)
755|	}
756|	defer rows.Close()
757|	return scanTokenEventsPostgres(rows)
758|}
759|
760|func (r *PostgresRepository) ListTokenEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error) {
761|	if limit <= 0 || limit > 500 {
762|		limit = 100
763|	}
764|	rows, err := r.pool.Query(ctx, tokenEventSelectPostgres+`
765|		WHERE tenant_id = $1 AND occurred_at < $2
766|		ORDER BY occurred_at DESC, event_id DESC
767|		LIMIT $3
768|	`, tenantID, before, limit)
769|	if err != nil {
770|		return nil, wrapDBErr(err)
771|	}
772|	defer rows.Close()
773|	return scanTokenEventsPostgres(rows)
774|}
775|
776|func (r *PostgresRepository) ListAnomalySignals(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error) {
777|	if limit <= 0 || limit > 500 {
778|		limit = 100
779|	}
780|	rows, err := r.pool.Query(ctx, `
781|		SELECT tenant_id, anomaly_id, event_id, worker_id, detected_at, severity,
782|			type, description, observed_value, threshold_value, details_json
783|		FROM anomaly_signals
784|		WHERE tenant_id = $1
785|		ORDER BY detected_at DESC, anomaly_id DESC
786|		LIMIT $2
787|	`, tenantID, limit)
788|	if err != nil {
789|		return nil, wrapDBErr(err)
790|	}
791|	defer rows.Close()
792|
793|	var signals []domain.AnomalySignal
794|	for rows.Next() {
795|		var signal domain.AnomalySignal
796|		var eventID, workerID, details *string
797|		if err := rows.Scan(&signal.TenantID, &signal.AnomalyID, &eventID, &workerID,
798|			&signal.DetectedAt, &signal.Severity, &signal.Type, &signal.Description, &signal.ObservedValue,
799|			&signal.ThresholdValue, &details); err != nil {
800|			return nil, wrapDBErr(err)
801|		}
802|		if eventID != nil {
803|			signal.EventID = *eventID
804|		}
805|		if workerID != nil {
806|			signal.WorkerID = *workerID
807|		}
808|		if details != nil && *details != "" && *details != "{}" {
809|			_ = json.Unmarshal([]byte(*details), &signal.Details)
810|		}
811|		signals = append(signals, signal)
812|	}
813|	return signals, wrapDBErr(rows.Err())
814|}
815|
816|const outputAnalysisSelectPostgres = `
817|	SELECT tenant_id, analysis_id, event_id, worker_id, analyzed_at,
818|		efficiency_score, goblin_score, issues_json, recommendations_json,
819|		evidence_json, degraded_json
820|	FROM output_analyses
821|`
822|
823|func scanOutputAnalysesPostgres(rows pgx.Rows) ([]domain.OutputAnalysis, error) {
824|	var analyses []domain.OutputAnalysis
825|	for rows.Next() {
826|		var analysis domain.OutputAnalysis
827|		var issuesJSON, recommendationsJSON, evidenceJSON string
828|		var degradedJSON *string
829|		if err := rows.Scan(
830|			&analysis.TenantID,
831|			&analysis.AnalysisID,
832|			&analysis.EventID,
833|			&analysis.WorkerID,
834|			&analysis.AnalyzedAt,
835|			&analysis.EfficiencyScore,
836|			&analysis.GoblinScore,
837|			&issuesJSON,
838|			&recommendationsJSON,
839|			&evidenceJSON,
840|			&degradedJSON,
841|		); err != nil {
842|			return nil, wrapDBErr(err)
843|		}
844|		if issuesJSON != "" {
845|			if err := json.Unmarshal([]byte(issuesJSON), &analysis.Issues); err != nil {
846|				return nil, wrapDBErr(err)
847|			}
848|		}
849|		if recommendationsJSON != "" {
850|			if err := json.Unmarshal([]byte(recommendationsJSON), &analysis.Recommendations); err != nil {
851|				return nil, wrapDBErr(err)
852|			}
853|		}
854|		if evidenceJSON != "" {
855|			if err := json.Unmarshal([]byte(evidenceJSON), &analysis.Evidence); err != nil {
856|				return nil, wrapDBErr(err)
857|			}
858|		}
859|		if degradedJSON != nil && *degradedJSON != "" && *degradedJSON != "[]" && *degradedJSON != "null" {
860|			if err := json.Unmarshal([]byte(*degradedJSON), &analysis.Degraded); err != nil {
861|				return nil, wrapDBErr(err)
862|			}
863|		}
864|		analyses = append(analyses, analysis)
865|	}
866|	return analyses, wrapDBErr(rows.Err())
867|}
868|
869|const tokenEventSelectPostgres = `
870|	SELECT tenant_id, event_id, worker_id, worker_name, job_id, session_id, run_id,
871|		provider, model_id, prompt_tokens, completion_tokens, cached_tokens,
872|		input_tokens, output_tokens, total_tokens, cost_estimate_usd, cost_currency,
873|		cost_is_degraded, cost_degraded_code, external_estimate_usd,
874|		external_estimate_currency, latency_ms, task_category, output_status,
875|		review_score, occurred_at, created_at, prompt_excerpt, output_excerpt,
876|		prompt_reference, output_reference, tags_json, idempotency_key
877|	FROM token_usage_events
878|`
879|
880|func scanTokenEventsPostgres(rows pgx.Rows) ([]domain.TokenEvent, error) {
881|	var events []domain.TokenEvent
882|	for rows.Next() {
883|		var event domain.TokenEvent
884|		var jobID, sessionID, runID, costCode, externalCurrency, promptExcerpt, outputExcerpt, promptReference, outputReference, tags, idempotencyKey *string
885|		var externalCost *float64
886|		if err := rows.Scan(&event.TenantID, &event.EventID, &event.WorkerID, &event.WorkerName,
887|			&jobID, &sessionID, &runID, &event.Provider, &event.ModelID, &event.PromptTokens,
888|			&event.CompletionTokens, &event.CachedTokens, &event.InputTokens, &event.OutputTokens,
889|			&event.TotalTokens, &event.CostEstimateUSD, &event.CostCurrency, &event.CostIsDegraded, &costCode,
890|			&externalCost, &externalCurrency, &event.LatencyMs, &event.TaskCategory,
891|			&event.OutputStatus, &event.ReviewScore, &event.Timestamp, &event.CreatedAt, &promptExcerpt,
892|			&outputExcerpt, &promptReference, &outputReference, &tags, &idempotencyKey); err != nil {
893|			return nil, wrapDBErr(err)
894|		}
895|		if jobID != nil {
896|			event.JobID = *jobID
897|		}
898|		if sessionID != nil {
899|			event.SessionID = *sessionID
900|		}
901|		if runID != nil {
902|			event.RunID = *runID
903|		}
904|		if costCode != nil {
905|			event.CostDegradedCode = *costCode
906|		}
907|		if promptExcerpt != nil {
908|			event.PromptExcerpt = *promptExcerpt
909|		}
910|		if outputExcerpt != nil {
911|			event.OutputExcerpt = *outputExcerpt
912|		}
913|		if promptReference != nil {
914|			event.PromptReference = *promptReference
915|		}
916|		if outputReference != nil {
917|			event.OutputReference = *outputReference
918|		}
919|		event.TaskType = event.TaskCategory
920|		if externalCost != nil {
921|			event.ExternalEstimate = &domain.ExternalEstimate{
922|				CostUSD: *externalCost,
923|			}
924|			if externalCurrency != nil {
925|				event.ExternalEstimate.Currency = *externalCurrency
926|			}
927|		}
928|		if tags != nil && *tags != "" && *tags != "{}" {
929|			_ = json.Unmarshal([]byte(*tags), &event.Tags)
930|		}
931|		if idempotencyKey != nil {
932|			event.IdempotencyKey = *idempotencyKey
933|		}
934|		events = append(events, event)
935|	}
936|	return events, wrapDBErr(rows.Err())
937|}
938|
939|func (r *PostgresRepository) SaveRecommendationDecision(ctx context.Context, tenantID, recID, status string) error {
940|	now := time.Now().UTC()
941|	_, err := r.pool.Exec(ctx, `
942|		INSERT INTO recommendation_states (tenant_id, recommendation_id, status, actor, note, created_at, updated_at)
943|		VALUES ($1, $2, $3, $4, $5, $6, $7)
944|		ON CONFLICT(tenant_id, recommendation_id) DO UPDATE SET
945|			status = EXCLUDED.status,
946|			updated_at = EXCLUDED.updated_at
947|	`, tenantID, recID, status, nil, nil, now, now)
948|	return wrapDBErr(err)
949|}
950|
951|func (r *PostgresRepository) GetRecommendationDecisions(ctx context.Context, tenantID string) (map[string]string, error) {
952|	rows, err := r.pool.Query(ctx, `
953|		SELECT recommendation_id, status
954|		FROM recommendation_states
955|		WHERE tenant_id = $1
956|	`, tenantID)
957|	if err != nil {
958|		return nil, wrapDBErr(err)
959|	}
960|	defer rows.Close()
961|	decisions := make(map[string]string)
962|	for rows.Next() {
963|		var recID, status string
964|		if err := rows.Scan(&recID, &status); err != nil {
965|			return nil, wrapDBErr(err)
966|		}
967|		decisions[recID] = status
968|	}
969|	return decisions, wrapDBErr(rows.Err())
970|}
971|
972|func (r *PostgresRepository) GetTenantByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.Tenant, error) {
973|	var tenant domain.Tenant
974|	var stripeCustID, stripeSubID *string
975|	err := r.pool.QueryRow(ctx, `
976|		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
977|		FROM tenants
978|		WHERE stripe_customer_id = $1
979|	`, stripeCustomerID).Scan(&tenant.TenantID, &tenant.Name, &tenant.Tier, &tenant.UsageLimitUSD, &stripeCustID, &stripeSubID, &tenant.CreatedAt, &tenant.UpdatedAt)
980|	if err != nil {
981|		if errors.Is(err, pgx.ErrNoRows) {
982|			return nil, nil
983|		}
984|		return nil, wrapDBErr(err)
985|	}
986|	if stripeCustID != nil {
987|		tenant.StripeCustomerID = *stripeCustID
988|	}
989|	if stripeSubID != nil {
990|		tenant.StripeSubscriptionID = *stripeSubID
991|	}
992|	return &tenant, nil
993|}
994|
995|func (r *PostgresRepository) GetTenantByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (*domain.Tenant, error) {
996|	var tenant domain.Tenant
997|	var stripeCustID, stripeSubID *string
998|	err := r.pool.QueryRow(ctx, `
999|		SELECT tenant_id, name, tier, usage_limit_usd, stripe_customer_id, stripe_subscription_id, created_at, updated_at
1000|		FROM tenants
1001|		WHERE stripe_subscription_id = $1
1002|	`, stripeSubscriptionID).Scan(&tenant.TenantID, &tenant.Name, &tenant.Tier, &tenant.UsageLimitUSD, &stripeCustID, &stripeSubID, &tenant.CreatedAt, &tenant.UpdatedAt)
1003|	if err != nil {
1004|		if errors.Is(err, pgx.ErrNoRows) {
1005|			return nil, nil
1006|		}
1007|		return nil, wrapDBErr(err)
1008|	}
1009|	if stripeCustID != nil {
1010|		tenant.StripeCustomerID = *stripeCustID
1011|	}
1012|	if stripeSubID != nil {
1013|		tenant.StripeSubscriptionID = *stripeSubID
1014|	}
1015|	return &tenant, nil
1016|}
1017|
1018|func (r *PostgresRepository) UpsertAgent(ctx context.Context, agent domain.Agent) error {
1019|	return errors.New("not implemented")
1020|}
1021|func (r *PostgresRepository) ListAgents(ctx context.Context, tenantID string) ([]domain.Agent, error) {
1022|	return nil, errors.New("not implemented")
1023|}
1024|func (r *PostgresRepository) UpsertGovernancePolicy(ctx context.Context, policy domain.GovernancePolicy) error {
1025|	return errors.New("not implemented")
1026|}
1027|func (r *PostgresRepository) ListGovernancePolicies(ctx context.Context, tenantID string) ([]domain.GovernancePolicy, error) {
1028|	return nil, errors.New("not implemented")
1029|}
1030|func (r *PostgresRepository) UpsertBudget(ctx context.Context, budget domain.Budget) error {
1031|	return errors.New("not implemented")
1032|}
1033|func (r *PostgresRepository) ListBudgets(ctx context.Context, tenantID string) ([]domain.Budget, error) {
1034|	return nil, errors.New("not implemented")
1035|}
1036|
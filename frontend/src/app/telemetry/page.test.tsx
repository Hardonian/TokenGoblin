/**
 * Telemetry page render tests: the widgets must show REAL mirrored rows, and
 * product states (tier gate, mirror offline) must render as honest blocks —
 * never crash, never fabricate.
 */
import { render, screen, waitFor } from "@testing-library/react";
import { SWRConfig } from "swr";
import TelemetryPage from "./page";

jest.mock("next/navigation", () => ({
  useRouter: () => ({ push: jest.fn() }),
}));
jest.mock("next/link", () => ({
  __esModule: true,
  default: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));
jest.mock("@/lib/auth", () => ({
  useAuth: () => ({ tenantId: "demo-tenant", isLoading: false }),
}));

type FetchRoute = { body: unknown; status: number };

function mockFetch(routes: Record<string, FetchRoute>) {
  global.fetch = jest.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    const hit = Object.entries(routes).find(([frag]) => url.includes(frag));
    const route = hit?.[1] ?? { body: { ok: false, error: { code: "http_404", message: "no route" } }, status: 404 };
    return {
      ok: route.status >= 200 && route.status < 300,
      status: route.status,
      json: async () => route.body,
    } as Response;
  }) as jest.Mock;
}

function renderPage() {
  return render(
    <SWRConfig value={{ provider: () => new Map(), shouldRetryOnError: false, dedupingInterval: 0 }}>
      <TelemetryPage />
    </SWRConfig>
  );
}

const ok = (data: unknown) => ({ body: { ok: true, status: "success", data }, status: 200 });

describe("TelemetryPage", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("renders mirrored rows from the analytics endpoints", async () => {
    mockFetch({
      "/v1/analytics/cost/by-model": ok([
        { Model: "expensive-model", TotalCostUSD: 8.658, TotalTokens: 466500, RequestCount: 8, AvgCostPerReq: 1.08 },
      ]),
      "/v1/analytics/cost/by-feature": ok([
        { Feature: "research", TotalCostUSD: 8.658, TotalTokens: 466500, RequestCount: 8, ROI: 1.24 },
      ]),
      "/v1/analytics/cost": ok({
        TenantID: "demo-tenant",
        PeriodStart: "2026-09-05T00:00:00Z",
        PeriodEnd: "2026-10-05T00:00:00Z",
        TotalCostUSD: 8.66,
        TotalTokens: 497670,
        RequestCount: 23,
      }),
      "/v1/analytics/zombie-agents": ok([
        {
          AgentID: "worker-unknown",
          AcceptanceRate: 0,
          TotalCost: 0,
          TotalRequests: 6,
          LastActivity: "2026-10-05T22:07:00Z",
          Recommendation: "investigate",
        },
      ]),
      "/v1/analytics/anomalies": ok([
        {
          ID: "demo-unknown-06:unknown_model_pricing",
          Type: "unknown_model_pricing",
          Severity: "medium",
          Description: "Pricing was unavailable for this provider/model; cost is degraded.",
          Timestamp: "2026-10-05T22:07:00Z",
          MetricValue: 0,
          Threshold: 0,
          Metadata: { worker_id: "worker-unknown", event_id: "demo-unknown-06" },
        },
      ]),
    });

    renderPage();

    expect(await screen.findByText("expensive-model")).toBeInTheDocument();
    expect(screen.getByText("research")).toBeInTheDocument();
    expect(screen.getByText("worker-unknown")).toBeInTheDocument();
    expect(screen.getByText(/Pricing was unavailable/)).toBeInTheDocument();
    // ROI column shows the real derived metric.
    expect(screen.getByText("$1.24")).toBeInTheDocument();
  });

  it("renders the paid-tier gate as an honest upgrade block", async () => {
    mockFetch({
      "/v1/analytics/cost/by-model": ok([]),
      "/v1/analytics/cost/by-feature": ok([]),
      "/v1/analytics/cost": ok({
        TenantID: "demo-tenant",
        PeriodStart: "2026-09-05T00:00:00Z",
        PeriodEnd: "2026-10-05T00:00:00Z",
        TotalCostUSD: 0,
        TotalTokens: 0,
        RequestCount: 0,
      }),
      "/v1/analytics/anomalies": ok([]),
      "/v1/analytics/zombie-agents": {
        body: { ok: false, status: "error", error: { code: "upgrade_required", message: "This feature requires a paid plan." } },
        status: 403,
      },
    });

    renderPage();

    expect(await screen.findByText(/Paid plan required/)).toBeInTheDocument();
    // The gate must NOT crash the rest of the page.
    await waitFor(() => expect(screen.getByText("Cost_By_Model")).toBeInTheDocument());
  });

  it("renders the mirror-offline state without fabricating numbers", async () => {
    mockFetch({
      "/v1/analytics/cost/by-model": {
        body: { ok: false, status: "error", error: { code: "analytics_unavailable", message: "mirror not configured" } },
        status: 503,
      },
      "/v1/analytics/cost/by-feature": {
        body: { ok: false, status: "error", error: { code: "analytics_unavailable", message: "mirror not configured" } },
        status: 503,
      },
      "/v1/analytics/cost": {
        body: { ok: false, status: "error", error: { code: "analytics_unavailable", message: "mirror not configured" } },
        status: 503,
      },
      "/v1/analytics/zombie-agents": {
        body: { ok: false, status: "error", error: { code: "analytics_unavailable", message: "mirror not configured" } },
        status: 503,
      },
      "/v1/analytics/anomalies": {
        body: { ok: false, status: "error", error: { code: "analytics_unavailable", message: "mirror not configured" } },
        status: 503,
      },
    });

    renderPage();

    expect(await screen.findAllByText(/mirror offline/)).not.toHaveLength(0);
    // No fabricated zeros masquerading as data.
    expect(screen.queryByText("LIVE")).toBeNull();
  });
});
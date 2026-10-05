"use client";

import useSWR from "swr";
import Link from "next/link";
import { useAuth } from "@/lib/auth";

type CostSummary = {
  TenantID: string;
  PeriodStart: string;
  PeriodEnd: string;
  TotalCostUSD: number;
  TotalTokens: number;
  RequestCount: number;
};

type ModelCost = {
  Model: string;
  TotalCostUSD: number;
  TotalTokens: number;
  RequestCount: number;
  AvgCostPerReq: number;
};

type FeatureCost = {
  Feature: string;
  TotalCostUSD: number;
  TotalTokens: number;
  RequestCount: number;
  ROI: number;
};

type ZombieRow = {
  AgentID: string;
  AcceptanceRate: number;
  TotalCost: number;
  TotalRequests: number;
  LastActivity: string;
  Recommendation: string;
};

type AnomalyRow = {
  ID: string;
  Type: string;
  Severity: string;
  Description: string;
  Timestamp: string;
  MetricValue: number;
  Threshold: number;
  Metadata?: Record<string, string>;
};

type Query<T> = {
  status: number;
  data: T | null;
  errorCode: string | null;
  errorMessage: string | null;
};

// Envelope-preserving fetcher: 503 (mirror off) and 403 (tier gate) are
// PRODUCT STATES, not crashes — the UI must render them honestly.
async function envelopeFetcher(url: string): Promise<Query<unknown>> {
  const res = await fetch(url, {
    headers: {
      Accept: "application/json",
      // Demo-mode fallback identity — real credentials (session cookie / API
      // key) always win server-side; production rejects this header outright.
      "x-tenant-id": "demo-tenant",
    },
    cache: "no-store",
  });
  const json = await res.json().catch(() => null);
  return {
    status: res.status,
    data: json?.ok ? (json.data ?? null) : null,
    errorCode: json?.error?.code ?? (res.ok ? null : `http_${res.status}`),
    errorMessage: json?.error?.message ?? (res.ok ? null : `Request failed (${res.status})`),
  };
}

const money = (n: number) =>
  n >= 100 ? `$${n.toFixed(0)}` : n >= 0.01 ? `$${n.toFixed(2)}` : `$${n.toFixed(4)}`;
const num = (n: number) => n.toLocaleString("en-US");
const pct = (n: number) => `${(n * 100).toFixed(0)}%`;

function Card({ title, badge, children }: { title: string; badge?: string; children: React.ReactNode }) {
  return (
    <div className="border border-[#333] bg-black">
      <div className="border-b border-[#333] px-4 py-3 flex justify-between items-center bg-[#0a0a0a]">
        <h2 className="text-zinc-300 font-bold tracking-widest text-sm uppercase">{title}</h2>
        {badge ? <span className="text-xs text-zinc-600 uppercase">{badge}</span> : null}
      </div>
      <div className="p-4">{children}</div>
    </div>
  );
}

function StateBlock({ code, message }: { code: string; message: string }) {
  if (code === "upgrade_required") {
    return (
      <div className="bg-[#0a0a0a] border border-amber-900 p-4">
        <p className="text-[#ffb000] text-xs uppercase tracking-widest font-bold mb-2">
          [ locked ] Paid plan required
        </p>
        <p className="text-zinc-500 text-xs mb-3">{message}</p>
        <Link
          href="/pricing"
          className="bg-black hover:bg-[#111] border border-[#333] hover:border-zinc-500 text-[#ffb000] text-xs px-4 py-1.5 transition-all uppercase tracking-widest"
        >
          [ View Plans ]
        </Link>
      </div>
    );
  }
  if (code === "analytics_unavailable") {
    return (
      <div className="bg-[#0a0a0a] border border-[#333] p-4">
        <p className="text-zinc-400 text-xs uppercase tracking-widest font-bold mb-2">
          [ mirror offline ] Telemetry not configured
        </p>
        <p className="text-zinc-600 text-xs">{message}</p>
      </div>
    );
  }
  return (
    <div className="bg-[#0a0a0a] border border-red-950 p-4">
      <p className="text-red-500 text-xs uppercase tracking-widest font-bold mb-2">[ error ] {code}</p>
      <p className="text-zinc-600 text-xs">{message}</p>
    </div>
  );
}

function Empty({ label }: { label: string }) {
  return <p className="text-zinc-600 text-xs uppercase tracking-widest">{">>"} {label}</p>;
}

export default function TelemetryPage() {
  const { tenantId, isLoading: authLoading } = useAuth();
  // Fetches run for signed-in AND anonymous visitors: the fetcher carries a
  // demo-tenant fallback identity that the API honors only in demo mode.
  const active = true;

  const cost = useSWR(active ? "/v1/analytics/cost" : null, envelopeFetcher);
  const byModel = useSWR(active ? "/v1/analytics/cost/by-model" : null, envelopeFetcher);
  const byFeature = useSWR(active ? "/v1/analytics/cost/by-feature" : null, envelopeFetcher);
  const zombies = useSWR(
    active ? "/v1/analytics/zombie-agents?threshold=0.2" : null,
    envelopeFetcher
  );
  const anomalies = useSWR(active ? "/v1/analytics/anomalies" : null, envelopeFetcher);

  const summary = (cost.data?.data ?? null) as CostSummary | null;
  const modelRows = ((byModel.data?.data ?? []) as ModelCost[]);
  const featureRows = ((byFeature.data?.data ?? []) as FeatureCost[]);
  const zombieRows = ((zombies.data?.data ?? []) as ZombieRow[]);
  const anomalyRows = ((anomalies.data?.data ?? []) as AnomalyRow[]);

  return (
    <div className="min-h-screen bg-[#050505] text-white">
      <div className="max-w-[1400px] mx-auto px-6 py-6">
        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
          <div>
            <div className="text-[#10b981] font-black text-xl">[TELEMETRY_MIRROR]</div>
            <h1 className="text-lg font-bold text-white tracking-widest uppercase">
              Enterprise Telemetry — ClickHouse Analytics
            </h1>
            <p className="text-xs text-[#ffb000] uppercase tracking-[0.2em]">
              High-volume cost intelligence, mirrored from the source of truth
            </p>
          </div>
          <Link
            href="/"
            className="bg-black hover:bg-[#111] border border-[#333] hover:border-zinc-500 text-zinc-400 text-xs px-4 py-1.5 transition-all uppercase tracking-widest"
          >
            [ {"<"} War Room ]
          </Link>
        </div>

        {authLoading ? (
          <Empty label="Authenticating..." />
        ) : !tenantId ? (
          <div className="bg-[#0a0a0a] border border-[#333] p-3 mb-6 flex flex-wrap items-center gap-3">
            <span className="text-[#ffb000] text-xs uppercase tracking-widest font-bold">
              [ demo mode ] Viewing the demo tenant
            </span>
            <span className="text-zinc-600 text-xs">
              Sign in to see telemetry scoped to your own tenant.
            </span>
            <Link
              href="/login"
              className="ml-auto bg-black hover:bg-[#111] border border-[#333] hover:border-zinc-500 text-[#ffb000] text-xs px-4 py-1.5 transition-all uppercase tracking-widest"
            >
              [ Login ]
            </Link>
          </div>
        ) : null}

        {!authLoading && (
          <div className="grid gap-6">
            {/* Cost summary strip */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              {cost.data?.errorCode ? (
                <div className="col-span-2 lg:col-span-4">
                  <StateBlock code={cost.data.errorCode} message={cost.data.errorMessage ?? ""} />
                </div>
              ) : (
                <>
                  <Card title="Window_Cost" badge="30d">
                    <div className="text-2xl font-black text-[#10b981]">
                      {summary ? money(summary.TotalCostUSD) : "..."}
                    </div>
                  </Card>
                  <Card title="Requests">
                    <div className="text-2xl font-black text-white">
                      {summary ? num(summary.RequestCount) : "..."}
                    </div>
                  </Card>
                  <Card title="Tokens">
                    <div className="text-2xl font-black text-white">
                      {summary ? num(summary.TotalTokens) : "..."}
                    </div>
                  </Card>
                  <Card title="Mirror" badge="ClickHouse">
                    <div className="text-2xl font-black text-[#ffb000]">LIVE</div>
                  </Card>
                </>
              )}
            </div>

            {/* Zombie agents — the tier-gated signature feature */}
            <Card title="Zombie_Agents" badge="pro+ · threshold 20%">
              {zombies.data?.errorCode ? (
                <StateBlock code={zombies.data.errorCode} message={zombies.data.errorMessage ?? ""} />
              ) : zombieRows.length === 0 ? (
                <Empty label="No zombies. Every agent is earning its keep." />
              ) : (
                <div className="grid gap-3">
                  {zombieRows.map((z) => (
                    <div
                      key={z.AgentID}
                      className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 bg-[#0a0a0a] border border-[#222] p-3 hover:border-red-900 transition-colors"
                    >
                      <div>
                        <div className="text-red-500 font-bold text-xs">
                          [{z.Recommendation.toUpperCase()}]
                        </div>
                        <div className="text-white text-sm font-bold">{z.AgentID}</div>
                        <div className="text-zinc-600 text-xs">
                          {num(z.TotalRequests)} requests · acceptance {pct(z.AcceptanceRate)} ·{" "}
                          {money(z.TotalCost)} visible spend
                        </div>
                      </div>
                      <div className="text-[#ffb000] text-xs uppercase tracking-widest">
                        spend with no output
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </Card>

            {/* Cost by model + by feature */}
            <div className="grid lg:grid-cols-2 gap-6">
              <Card title="Cost_By_Model" badge="window 30d">
                {byModel.data?.errorCode ? (
                  <StateBlock code={byModel.data.errorCode} message={byModel.data.errorMessage ?? ""} />
                ) : modelRows.length === 0 ? (
                  <Empty label="No mirrored events in window." />
                ) : (
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="text-zinc-600 uppercase tracking-widest text-left">
                        <th className="py-2">Model</th>
                        <th className="py-2 text-right">Cost</th>
                        <th className="py-2 text-right">Reqs</th>
                        <th className="py-2 text-right">Avg/req</th>
                      </tr>
                    </thead>
                    <tbody>
                      {modelRows.map((m) => (
                        <tr key={m.Model} className="border-t border-[#222]">
                          <td className="py-2 text-white">{m.Model}</td>
                          <td className="py-2 text-right text-[#10b981]">{money(m.TotalCostUSD)}</td>
                          <td className="py-2 text-right text-zinc-400">{num(m.RequestCount)}</td>
                          <td className="py-2 text-right text-zinc-400">{money(m.AvgCostPerReq)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </Card>

              <Card title="Cost_By_Feature" badge="ROI = cost / accepted">
                {byFeature.data?.errorCode ? (
                  <StateBlock code={byFeature.data.errorCode} message={byFeature.data.errorMessage ?? ""} />
                ) : featureRows.length === 0 ? (
                  <Empty label="No mirrored events in window." />
                ) : (
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="text-zinc-600 uppercase tracking-widest text-left">
                        <th className="py-2">Feature</th>
                        <th className="py-2 text-right">Cost</th>
                        <th className="py-2 text-right">Reqs</th>
                        <th className="py-2 text-right">ROI</th>
                      </tr>
                    </thead>
                    <tbody>
                      {featureRows.map((f) => (
                        <tr key={f.Feature} className="border-t border-[#222]">
                          <td className="py-2 text-white">{f.Feature}</td>
                          <td className="py-2 text-right text-[#10b981]">{money(f.TotalCostUSD)}</td>
                          <td className="py-2 text-right text-zinc-400">{num(f.RequestCount)}</td>
                          <td className="py-2 text-right text-[#ffb000]">
                            {f.ROI > 0 ? money(f.ROI) : "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </Card>
            </div>

            {/* Mirrored anomalies */}
            <Card title="Anomaly_Signals" badge="mirrored">
              {anomalies.data?.errorCode ? (
                <StateBlock code={anomalies.data.errorCode} message={anomalies.data.errorMessage ?? ""} />
              ) : anomalyRows.length === 0 ? (
                <Empty label="No anomalies in window." />
              ) : (
                <div className="grid gap-2">
                  {anomalyRows.slice(0, 20).map((a) => (
                    <div key={a.ID} className="bg-[#0a0a0a] border border-[#222] p-3">
                      <div className="flex flex-wrap items-center gap-3">
                        <span className="text-[#ffb000] font-bold text-xs">
                          [{a.Severity.toUpperCase()}]
                        </span>
                        <span className="text-zinc-400 text-xs uppercase tracking-widest">{a.Type}</span>
                        <span className="text-zinc-600 text-xs ml-auto">
                          {new Date(a.Timestamp).toLocaleString()}
                        </span>
                      </div>
                      <p className="text-zinc-300 text-xs mt-1">{a.Description}</p>
                      {a.Metadata?.worker_id ? (
                        <p className="text-zinc-600 text-xs mt-1">
                          worker: {a.Metadata.worker_id}
                          {a.Metadata.event_id ? ` · event: ${a.Metadata.event_id}` : ""}
                        </p>
                      ) : null}
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </div>
        )}
      </div>
    </div>
  );
}
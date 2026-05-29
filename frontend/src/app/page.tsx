"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";

type Envelope<T> = {
  ok: boolean;
  status: string;
  data?: T;
  degraded?: Issue[];
  error?: Issue;
};

type Issue = {
  code: string;
  message: string;
};

type Summary = {
  total_cost_usd: number;
  known_cost_event_count: number;
  unknown_cost_event_count: number;
  total_events: number;
  total_output_tokens: number;
  output_count: number;
  avg_latency_ms?: number;
  anomaly_count: number;
  cost_by_worker: Worker[];
  top_cost_drivers: CostDriver[];
  degraded?: Issue[];
};

type Worker = {
  worker_id: string;
  worker_name: string;
  event_count: number;
  output_count: number;
  failed_output_count: number;
  total_tokens: number;
  total_cost_usd: number;
  unknown_cost_event_count: number;
  efficiency_rating?: string;
  trend?: string;
};

type CostDriver = {
  type: string;
  key: string;
  label: string;
  total_cost_usd: number;
  event_count: number;
};

type TokenEvent = {
  event_id: string;
  timestamp: string;
  worker_id: string;
  worker_name: string;
  provider: string;
  model_id: string;
  total_tokens: number;
  cost_estimate_usd?: number;
  cost_is_degraded: boolean;
  task_category?: string;
  output_status: string;
};

type Recommendation = {
  recommendation_id: string;
  task_category: string;
  current_model: string;
  recommended_model: string;
  estimated_savings_usd: number;
  evidence_count: number;
  confidence: string;
  status: string;
  status_note?: string;
  reason: string;
};

type OutputAnalysis = {
  analysis_id: string;
  event_id: string;
  worker_id: string;
  efficiency_score: number;
  goblin_score: number;
  issues: { code: string; severity: string; message: string; evidence?: string }[];
  recommendations: string[];
  degraded?: Issue[];
};

type AuditEvent = {
  event_id: string;
  type: string;
  actor: string;
  resource?: string;
  timestamp: string;
};

type TenantMember = {
  subject_id: string;
  email?: string;
  role: string;
};

const apiBase =
  process.env.NEXT_PUBLIC_TG_API_BASE?.replace(/\/$/, "") ||
  "http://localhost:8080";

export default function Home() {
  const [tenant, setTenant] = useState("demo-tenant");
  const [summary, setSummary] = useState<Envelope<Summary> | null>(null);
  const [workers, setWorkers] = useState<Envelope<Worker[]> | null>(null);
  const [events, setEvents] = useState<Envelope<TokenEvent[]> | null>(null);
  const [recommendations, setRecommendations] =
    useState<Envelope<Recommendation[]> | null>(null);
  const [analyses, setAnalyses] = useState<Envelope<OutputAnalysis[]> | null>(
    null,
  );
  const [auditEvents, setAuditEvents] = useState<Envelope<AuditEvent[]> | null>(
    null,
  );
  const [members, setMembers] = useState<Envelope<TenantMember[]> | null>(null);
  const [busyAction, setBusyAction] = useState("");

  const load = useCallback(async () => {
    const headers = { "x-tenant-id": tenant };
    const [
      overviewRes,
      workersRes,
      eventsRes,
      recsRes,
      analysesRes,
      auditRes,
      membersRes,
    ] =
      await Promise.all([
        fetch(`${apiBase}/api/dashboard/overview`, { headers }),
        fetch(`${apiBase}/api/dashboard/workers`, { headers }),
        fetch(`${apiBase}/api/dashboard/events?limit=12`, { headers }),
        fetch(`${apiBase}/api/dashboard/recommendations`, { headers }),
        fetch(`${apiBase}/api/dashboard/output-analysis?limit=8`, { headers }),
        fetch(`${apiBase}/api/audit/events?limit=6`, { headers }),
        fetch(`${apiBase}/api/tenant/members`, { headers }),
      ]);

    setSummary(await overviewRes.json());
    setWorkers(await workersRes.json());
    setEvents(await eventsRes.json());
    setRecommendations(await recsRes.json());
    setAnalyses(await analysesRes.json());
    setAuditEvents(await auditRes.json());
    setMembers(await membersRes.json());
  }, [tenant]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [load]);

  const totalTokens = useMemo(
    () =>
      summary?.data?.cost_by_worker?.reduce(
        (total, worker) => total + worker.total_tokens,
        0,
      ) ?? 0,
    [summary],
  );
  const avgOutputSize =
    summary?.data && summary.data.output_count > 0
      ? Math.round(summary.data.total_output_tokens / summary.data.output_count)
      : 0;

  async function runAction(path: string, method: "POST" | "DELETE") {
    setBusyAction(path);
    try {
      await fetch(`${apiBase}${path}`, {
        method,
        headers: { "x-tenant-id": tenant },
      });
      await load();
    } finally {
      setBusyAction("");
    }
  }

  async function setRecommendationStatus(recommendationID: string, status: string) {
    setBusyAction(`${recommendationID}:${status}`);
    try {
      await fetch(
        `${apiBase}/api/dashboard/recommendations/${encodeURIComponent(
          recommendationID,
        )}/status`,
        {
          method: "POST",
          headers: {
            "content-type": "application/json",
            "x-tenant-id": tenant,
          },
          body: JSON.stringify({ status }),
        },
      );
      await load();
    } finally {
      setBusyAction("");
    }
  }

  const empty = summary?.data?.total_events === 0;
  const degraded = [
    ...(summary?.degraded ?? []),
    ...(workers?.degraded ?? []),
    ...(events?.degraded ?? []),
    ...(recommendations?.degraded ?? []),
    ...(analyses?.degraded ?? []),
    ...(auditEvents?.degraded ?? []),
    ...(members?.degraded ?? []),
  ];

  return (
    <main className="min-h-screen bg-[#f7f8f3] text-[#171915]">
      <section className="border-b border-[#d7dccf] bg-[#fbfcf8]">
        <div className="mx-auto flex w-full max-w-7xl flex-col gap-5 px-5 py-6 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.16em] text-[#61705a]">
              TokenGoblin
            </p>
            <h1 className="mt-2 text-3xl font-semibold tracking-normal md:text-5xl">
              Token efficiency review
            </h1>
            <p className="mt-3 max-w-2xl text-sm leading-6 text-[#52604e]">
              Estimated spend, output bloat, worker patterns, and deterministic
              recommendations from persisted usage evidence.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <input
              className="h-10 w-44 border border-[#c5cdbb] bg-white px-3 text-sm outline-none focus:border-[#426b51]"
              value={tenant}
              onChange={(event) => setTenant(event.target.value)}
              aria-label="Tenant ID"
            />
            <button
              className="h-10 border border-[#171915] bg-[#171915] px-4 text-sm font-medium text-white disabled:opacity-50"
              disabled={busyAction !== ""}
              onClick={() => runAction("/api/dashboard/seed", "POST")}
            >
              Seed Demo
            </button>
            <a
              className="flex h-10 items-center border border-[#c5cdbb] bg-white px-4 text-sm font-medium"
              href={`${apiBase}/api/dashboard/report.md`}
              target="_blank"
              rel="noopener noreferrer"
            >
              Report
            </a>
          </div>
        </div>
      </section>

      <section className="mx-auto grid w-full max-w-7xl gap-4 px-5 py-5 md:grid-cols-4">
        <Metric label="Total tokens" value={formatInt(totalTokens)} />
        <Metric
          label="Estimated cost"
          value={`$${formatMoney(summary?.data?.total_cost_usd ?? 0)}`}
        />
        <Metric label="Output count" value={formatInt(summary?.data?.output_count ?? 0)} />
        <Metric label="Avg output size" value={`${formatInt(avgOutputSize)} tokens`} />
      </section>

      <section className="mx-auto grid w-full max-w-7xl gap-5 px-5 pb-8 lg:grid-cols-[1.3fr_0.7fr]">
        <div className="space-y-5">
          {empty && (
            <div className="border border-[#d7dccf] bg-white p-5">
              <h2 className="text-lg font-semibold">No review data yet</h2>
              <p className="mt-2 text-sm leading-6 text-[#52604e]">
                Ingest token usage events or seed demo data. Until records exist,
                TokenGoblin will not invent cost leaks, quality scores, or worker
                recommendations.
              </p>
            </div>
          )}

          <Panel title="Waste Signals">
            <div className="grid gap-3 md:grid-cols-2">
              {(summary?.data?.top_cost_drivers ?? []).map((driver) => (
                <div key={`${driver.type}:${driver.key}`} className="border border-[#e0e4d8] p-4">
                  <p className="text-xs uppercase text-[#61705a]">{driver.type}</p>
                  <h3 className="mt-1 font-semibold">{driver.label}</h3>
                  <p className="mt-2 text-sm text-[#52604e]">
                    ${formatMoney(driver.total_cost_usd)} estimated across{" "}
                    {driver.event_count} events.
                  </p>
                </div>
              ))}
            </div>
          </Panel>

          <Panel title="Worker Review">
            <div className="overflow-x-auto">
              <table className="w-full min-w-[680px] text-left text-sm">
                <thead className="border-b border-[#e0e4d8] text-xs uppercase text-[#61705a]">
                  <tr>
                    <th className="py-2">Worker</th>
                    <th>Tokens</th>
                    <th>Cost</th>
                    <th>Outputs</th>
                    <th>Waste</th>
                    <th>Trend</th>
                  </tr>
                </thead>
                <tbody>
                  {(workers?.data ?? []).map((worker) => (
                    <tr key={worker.worker_id} className="border-b border-[#edf0e8]">
                      <td className="py-3 font-medium">{worker.worker_name}</td>
                      <td>{formatInt(worker.total_tokens)}</td>
                      <td>${formatMoney(worker.total_cost_usd)}</td>
                      <td>{worker.output_count}</td>
                      <td>{worker.unknown_cost_event_count} unknown-cost events</td>
                      <td>{worker.trend || worker.efficiency_rating || "stable"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Panel>

          <Panel title="Recent Review Runs">
            <div className="space-y-2">
              {(events?.data ?? []).map((event) => (
                <div key={event.event_id} className="flex flex-wrap items-center justify-between gap-3 border border-[#e0e4d8] p-3 text-sm">
                  <div>
                    <p className="font-medium">{event.worker_name || event.worker_id}</p>
                    <p className="text-[#52604e]">{event.provider}:{event.model_id} · {event.task_category || "uncategorized"}</p>
                  </div>
                  <div className="text-right">
                    <p>{formatInt(event.total_tokens)} tokens</p>
                    <p className="text-[#52604e]">
                      {event.cost_estimate_usd == null ? "Cost unknown" : `$${formatMoney(event.cost_estimate_usd)} estimated`}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </Panel>
        </div>

        <aside className="space-y-5">
          <Panel title="Degraded States">
            {degraded.length === 0 ? (
              <p className="text-sm text-[#52604e]">No degraded states reported by the API.</p>
            ) : (
              <div className="space-y-2">
                {degraded.map((issue, index) => (
                  <p key={`${issue.code}-${index}`} className="border border-[#e0e4d8] p-3 text-sm">
                    <span className="font-medium">{issue.code}</span>: {issue.message}
                  </p>
                ))}
              </div>
            )}
          </Panel>

          <Panel title="Efficiency Wins">
            <div className="space-y-3">
              {(recommendations?.data ?? []).map((rec) => (
                <div key={rec.recommendation_id} className="border border-[#e0e4d8] p-3">
                  <p className="text-sm font-medium">{rec.task_category}</p>
                  <p className="mt-1 text-sm text-[#52604e]">{rec.reason}</p>
                  <p className="mt-2 text-xs uppercase text-[#61705a]">
                    {rec.confidence} confidence · {rec.evidence_count} events · {rec.status}
                  </p>
                  {rec.status_note && (
                    <p className="mt-2 text-xs text-[#52604e]">{rec.status_note}</p>
                  )}
                  <div className="mt-3 flex gap-2">
                    <button
                      className="border border-[#426b51] px-3 py-1 text-xs font-medium text-[#426b51] disabled:opacity-50"
                      disabled={busyAction !== ""}
                      onClick={() =>
                        setRecommendationStatus(rec.recommendation_id, "accepted")
                      }
                    >
                      Accept
                    </button>
                    <button
                      className="border border-[#c5cdbb] px-3 py-1 text-xs font-medium text-[#52604e] disabled:opacity-50"
                      disabled={busyAction !== ""}
                      onClick={() =>
                        setRecommendationStatus(rec.recommendation_id, "rejected")
                      }
                    >
                      Reject
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </Panel>

          <Panel title="Goblin Score">
            <div className="space-y-3">
              {(analyses?.data ?? []).map((item) => (
                <div key={item.analysis_id} className="border border-[#e0e4d8] p-3">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium">{item.worker_id}</p>
                    <p className="text-lg font-semibold">{item.goblin_score}</p>
                  </div>
                  <p className="mt-1 text-xs text-[#61705a]">event {item.event_id}</p>
                  <p className="mt-2 text-sm text-[#52604e]">
                    {item.issues[0]?.message || "No deterministic waste issue found."}
                  </p>
                </div>
              ))}
            </div>
          </Panel>

          <Panel title="Team Boundary">
            <div className="space-y-2">
              {(members?.data ?? []).map((member) => (
                <div key={member.subject_id} className="border border-[#e0e4d8] p-3 text-sm">
                  <p className="font-medium">{member.email || member.subject_id}</p>
                  <p className="text-xs uppercase text-[#61705a]">{member.role}</p>
                </div>
              ))}
              {(members?.data ?? []).length === 0 && (
                <p className="text-sm text-[#52604e]">
                  No explicit tenant members have been configured.
                </p>
              )}
            </div>
          </Panel>

          <Panel title="Audit Trail">
            <div className="space-y-2">
              {(auditEvents?.data ?? []).map((event) => (
                <div key={event.event_id} className="border border-[#e0e4d8] p-3 text-sm">
                  <p className="font-medium">{event.type}</p>
                  <p className="text-[#52604e]">
                    {event.actor} · {event.resource || "tenant"} ·{" "}
                    {new Date(event.timestamp).toLocaleString()}
                  </p>
                </div>
              ))}
              {(auditEvents?.data ?? []).length === 0 && (
                <p className="text-sm text-[#52604e]">
                  No persisted audit events yet.
                </p>
              )}
            </div>
          </Panel>

          <button
            className="w-full border border-[#9f2f2f] bg-white px-4 py-3 text-sm font-medium text-[#9f2f2f] disabled:opacity-50"
            disabled={busyAction !== ""}
            onClick={() => runAction("/api/dashboard/reset", "DELETE")}
          >
            Reset Tenant Data
          </button>
        </aside>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-[#d7dccf] bg-white p-4">
      <p className="text-xs font-medium uppercase text-[#61705a]">{label}</p>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="border border-[#d7dccf] bg-white p-5">
      <h2 className="mb-4 text-lg font-semibold">{title}</h2>
      {children}
    </section>
  );
}

function formatMoney(value: number) {
  return value.toFixed(value >= 10 ? 2 : 4);
}

function formatInt(value: number) {
  return new Intl.NumberFormat("en-US").format(value);
}

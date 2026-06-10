"use client";

import { useEffect, useRef, useState } from "react";

type Envelope<T> = {
  ok: boolean;
  status: string;
  data?: T;
  error?: { code: string; message: string };
};

type Scorecard = {
  maturity_score: number;
  grade: string;
  roi_multiplier: number;
  total_agents: number;
  active_agents: number;
  avg_latency_ms: number;
  failure_rate_pct: number;
  total_waste_usd: number;
  waste_pct: number;
};

export default function ExecutivePage() {
  const [tenantId, setTenantId] = useState("demo-tenant");
  const [scorecard, setScorecard] = useState<Scorecard | null>(null);
  const [error, setError] = useState<string | null>(null);
  const initialized = useRef(false);

  const load = async () => {
    setError(null);
    try {
      const res = await fetch("/v2/executive/scorecard", { headers: { "x-tenant-id": tenantId } });
      const payload: Envelope<Scorecard> = await res.json();
      if (!res.ok || !payload?.ok) {
        throw new Error(payload?.error?.message || "Scorecard failed");
      }
      setScorecard(payload.data || null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    }
  };

  useEffect(() => {
    if (!initialized.current) {
      initialized.current = true;
      const timer = window.setTimeout(() => {
        void load();
      }, 0);
      return () => window.clearTimeout(timer);
    }
  }, []);

  return (
    <main className="min-h-screen bg-background text-text-primary">
      <section className="mx-auto max-w-5xl px-6 py-12 space-y-8">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-accent">Executive</p>
            <h1 className="mt-2 text-3xl font-semibold text-white">Leadership Scorecard</h1>
            <p className="mt-2 text-sm text-text-secondary">AI maturity, fleet ROI, and waste.</p>
          </div>
          <div className="flex items-center gap-2">
            <input value={tenantId} onChange={(e) => setTenantId(e.target.value)} className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-white outline-none" placeholder="tenant-id" />
            <button onClick={load} className="rounded-lg bg-[#00ff9d] px-3 py-2 text-sm font-semibold text-black">Refresh</button>
          </div>
        </div>

        {error && (
          <div className="rounded-xl border border-[#ff4d4d]/40 bg-[#1b0505] p-4 text-sm text-red-300">
            {error}
          </div>
        )}

        <div className="grid gap-4 md:grid-cols-4">
          <Metric label="Maturity" value={`${scorecard?.maturity_score ?? 0}/100`} />
          <Metric label="Grade" value={scorecard?.grade ?? "N/A"} highlight />
          <Metric label="ROI" value={`${(scorecard?.roi_multiplier ?? 0).toFixed(1)}x`} />
          <Metric label="Waste" value={`${(scorecard?.waste_pct ?? 0).toFixed(1)}%`} />
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          <Metric label="Total Agents" value={`${scorecard?.total_agents ?? 0}`} />
          <Metric label="Active" value={`${scorecard?.active_agents ?? 0}`} />
          <Metric label="Avg Latency" value={`${scorecard?.avg_latency_ms ?? 0}ms`} />
        </div>
      </section>
    </main>
  );
}

function Metric({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className={`rounded-xl border border-border bg-surface p-4`}>
      <p className="text-xs text-text-muted">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${highlight ? "text-[#00ff9d]" : "text-white"}`}>{value}</p>
    </div>
  );
}

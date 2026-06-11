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
    <main className="min-h-screen bg-black text-zinc-300 font-mono pb-20 selection:bg-[#ffb000] selection:text-black">
      <section className="mx-auto max-w-[1400px] px-6 py-12 space-y-8">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between border-b border-[#333] pb-4">
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#ffb000] mb-2">[ Executive ]</p>
            <h1 className="text-xl font-bold text-white tracking-widest uppercase">Leadership Scorecard</h1>
            <p className="mt-2 text-xs text-zinc-500 uppercase tracking-widest">{'>>'} AI maturity, fleet ROI, and waste.</p>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-zinc-600 text-xs">--tenant</span>
            <input value={tenantId} onChange={(e) => setTenantId(e.target.value)} className="bg-black border border-[#333] text-[#ffb000] text-sm px-3 py-1 focus:outline-none focus:border-[#ffb000] transition-colors w-48" placeholder="tenant-id" />
            <button onClick={load} className="bg-[#ffb000] hover:bg-[#ff8c00] text-black font-bold text-xs px-4 py-1.5 transition-all uppercase tracking-widest">[ Sync ]</button>
          </div>
        </div>

        {error && (
          <div className="border border-red-900 bg-[#0a0000] p-4 text-xs text-red-500 font-bold uppercase tracking-widest">
            [ERR] {error}
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
    <div className={`border ${highlight ? "border-[#ffb000]" : "border-[#333]"} bg-black p-4 relative group hover:border-zinc-500 transition-colors`}>
      {highlight && <div className="absolute top-0 left-0 w-full h-[1px] bg-[#ffb000]"></div>}
      <p className="text-[10px] uppercase tracking-widest text-zinc-500 mb-4">[{label}]</p>
      <p className={`text-2xl font-bold tracking-tight ${highlight ? "text-[#ffb000]" : "text-zinc-200"}`}>{value}</p>
    </div>
  );
}

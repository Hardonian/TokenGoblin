"use client";

import { useEffect, useRef, useState } from "react";

type Envelope<T> = {
  ok: boolean;
  status: string;
  data?: T;
  error?: { code: string; message: string };
};

type Model = {
  model_id: string;
  provider: string;
  event_count: number;
  total_cost_usd: number;
  avg_cost_per_call: number;
  total_tokens: number;
  acceptance_rate: number;
  avg_latency_ms: number;
  cost_per_outcome: number;
};

export default function ModelsPage() {
  const [tenantId, setTenantId] = useState("demo-tenant");
  const [models, setModels] = useState<Model[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const initialized = useRef(false);

  const load = async () => {
    setError(null);
    try {
      const res = await fetch("/v2/analytics/models", { headers: { "x-tenant-id": tenantId } });
      const payload: Envelope<{ models: Model[]; count: number }> = await res.json();
      if (!res.ok || !payload?.ok) {
        throw new Error(payload?.error?.message || "Models failed");
      }
      setModels(payload.data?.models || []);
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
      <section className="mx-auto max-w-6xl px-6 py-12 space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-accent">Analytics</p>
            <h1 className="mt-2 text-3xl font-semibold text-white">Model Comparison</h1>
            <p className="mt-2 text-sm text-text-secondary">Cost, quality, and latency by model.</p>
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

        <div className="rounded-2xl border border-border bg-surface">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr>
                  <th className="px-5 py-4 text-xs text-text-muted">Model</th>
                  <th className="px-5 py-4 text-xs text-text-muted text-right">Calls</th>
                  <th className="px-5 py-4 text-xs text-text-muted text-right">Spend</th>
                  <th className="px-5 py-4 text-xs text-text-muted text-right">Cost/Call</th>
                  <th className="px-5 py-4 text-xs text-text-muted text-right">Outcome Cost</th>
                  <th className="px-5 py-4 text-xs text-text-muted text-right">Latency</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {(models || []).map((m, i) => (
                  <tr key={i} className="hover:bg-white/[0.02]">
                    <td className="px-5 py-4">
                      <div className="text-white">{m.model_id}</div>
                      <div className="text-xs text-text-muted">{m.provider}</div>
                    </td>
                    <td className="px-5 py-4 text-right">{m.event_count}</td>
                    <td className="px-5 py-4 text-right font-mono">${m.total_cost_usd.toFixed(2)}</td>
                    <td className="px-5 py-4 text-right font-mono">${m.avg_cost_per_call.toFixed(2)}</td>
                    <td className="px-5 py-4 text-right font-mono text-white">${m.cost_per_outcome.toFixed(2)}</td>
                    <td className="px-5 py-4 text-right">{m.avg_latency_ms.toFixed(1)}ms</td>
                  </tr>
                ))}
                {(!models || models.length === 0) && (
                  <tr>
                    <td colSpan={6} className="px-5 py-8 text-center text-text-muted">No model data. Seed demo data to populate.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </main>
  );
}

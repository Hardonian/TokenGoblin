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
    <main className="min-h-screen bg-black text-zinc-300 font-mono pb-20 selection:bg-[#ffb000] selection:text-black">
      <section className="mx-auto max-w-[1400px] px-6 py-12 space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between border-b border-[#333] pb-4">
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#ffb000] mb-2">[ Analytics ]</p>
            <h1 className="text-xl font-bold text-white tracking-widest uppercase">Model Matrix</h1>
            <p className="mt-2 text-xs text-zinc-500 uppercase tracking-widest">{'>>'} Cost, quality, and latency by model.</p>
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

        <div className="border border-[#333] bg-black">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333]">
                <tr>
                  <th className="px-5 py-4 font-normal">Model</th>
                  <th className="px-5 py-4 font-normal text-right">Calls</th>
                  <th className="px-5 py-4 font-normal text-right">Spend</th>
                  <th className="px-5 py-4 font-normal text-right">Cost/Call</th>
                  <th className="px-5 py-4 font-normal text-right">Outcome Cost</th>
                  <th className="px-5 py-4 font-normal text-right">Latency</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#222]">
                {(models || []).map((m, i) => (
                  <tr key={i} className="hover:bg-[#0a0a0a] transition-colors">
                    <td className="px-5 py-4">
                      <div className="text-zinc-200 font-bold">{m.model_id}</div>
                      <div className="text-[10px] text-zinc-500 uppercase tracking-widest">{m.provider}</div>
                    </td>
                    <td className="px-5 py-4 text-right text-zinc-400">{m.event_count}</td>
                    <td className="px-5 py-4 text-right font-mono text-zinc-300">${m.total_cost_usd.toFixed(2)}</td>
                    <td className="px-5 py-4 text-right font-mono text-zinc-400">${m.avg_cost_per_call.toFixed(2)}</td>
                    <td className="px-5 py-4 text-right font-mono font-bold text-[#ffb000]">${m.cost_per_outcome.toFixed(2)}</td>
                    <td className="px-5 py-4 text-right text-zinc-400">{m.avg_latency_ms.toFixed(1)}ms</td>
                  </tr>
                ))}
                {(!models || models.length === 0) && (
                  <tr>
                    <td colSpan={6} className="px-5 py-8 text-center text-zinc-600 uppercase tracking-widest">{'>>'} No model data. Seed demo data to populate.</td>
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

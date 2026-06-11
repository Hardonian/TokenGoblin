"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

type Envelope<T> = {
  ok: boolean;
  status: string;
  data?: T;
  error?: { code: string; message: string };
};

type HallucinationCell = {
  model: string;
  category: string;
  failure_rate: number;
};

type Waste = {
  total_waste_usd: number;
  wasteful_prompts: Array<{ fingerprint: string; total_cost_usd: number; event_count: number }>;
};

type Duplicates = {
  duplicate_clusters: Array<{ fingerprint: string; count: number; redundant_cost_usd: number }>;
};

export default function IntelligencePage() {
    const [waste, setWaste] = useState<Waste | null>(null);
  const [graveyard, setGraveyard] = useState<{ graveyard_prompts: Waste["wasteful_prompts"]; total_waste_usd: number; count: number } | null>(null);
  const [zombies, setZombies] = useState<{ zombie_agents: Array<{ worker_id: string; event_count: number; acceptance_rate: number; total_cost_usd: number }>; count: number } | null>(null);
  const [leaks, setLeaks] = useState<{ cost_leaks: Array<{ pattern_type: string; cost_usd: number; event_count: number }>; total_leak_cost: number; count: number } | null>(null);
  const [duplicates, setDuplicates] = useState<Duplicates | null>(null);
  const [heatmap, setHeatmap] = useState<{ heatmap_cells: HallucinationCell[]; count: number } | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const initialized = useRef(false);

  const loadAll = async () => {
    setStatus("loading");
    setError(null);
    try {
      const fetch2 = async <T,>(path: string) => {
        const res = await fetch(path);
        const payload: Envelope<T> = await res.json();
        if (!res.ok || !payload?.ok) throw new Error(payload?.error?.message || `Failed ${path}`);
        return payload.data as T;
      };
      const [w, g, z, l, d, h] = await Promise.all([
        fetch2<Waste>("/v2/intelligence/waste"),
        fetch2<typeof graveyard>("/v2/intelligence/prompt-graveyard"),
        fetch2<typeof zombies>("/v2/intelligence/zombie-agents"),
        fetch2<typeof leaks>("/v2/intelligence/cost-leaks"),
        fetch2<Duplicates>("/v2/intelligence/duplicates"),
        fetch2<typeof heatmap>("/v2/intelligence/hallucination-map"),
      ]);
      setWaste(w);
      setGraveyard(g);
      setZombies(z);
      setLeaks(l);
      setDuplicates(d);
      setHeatmap(h);
      setStatus("ready");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
      setStatus("error");
    }
  };

  useEffect(() => {
    if (!initialized.current) {
      initialized.current = true;
      const timer = window.setTimeout(() => {
        void loadAll();
      }, 0);
      return () => window.clearTimeout(timer);
    }
  }, []);

  return (
    <main className="min-h-screen bg-black text-zinc-300 font-mono pb-20 selection:bg-[#ffb000] selection:text-black">
      <section className="mx-auto max-w-[1400px] px-6 py-12 space-y-8">
        
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between border-b border-[#333] pb-4">
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#ffb000] mb-2">[ Intelligence ]</p>
            <h1 className="text-xl font-bold text-white tracking-widest uppercase">Fleet Intelligence</h1>
            <p className="mt-2 text-xs text-zinc-500 uppercase tracking-widest">{'>>'} Passive signals only — no agent modifications.</p>
          </div>
          
        </div>

        {error && (
          <div className="border border-red-900 bg-[#0a0000] p-4 text-xs text-red-500 font-bold uppercase tracking-widest">
            [ERR] {error}
          </div>
        )}

        <div className="grid gap-4 md:grid-cols-4">
          <Stat label="Total Waste" value={`$${(waste?.total_waste_usd ?? 0).toFixed(2)}`} />
          <Stat label="Waste Prompts" value={`${(waste?.wasteful_prompts?.length ?? 0)}`} />
          <Stat label="Leak Events" value={`${(leaks?.count ?? 0)}`} />
          <Stat label="Hallucination Clusters" value={`${(heatmap?.count ?? 0)}`} />
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <Card title="Cost Leaks">
            <LeaksTable leaks={leaks?.cost_leaks || []} />
          </Card>
          <Card title="Prompt Graveyard">
            <GraveyardTable items={graveyard?.graveyard_prompts || []} totalWaste={graveyard?.total_waste_usd || 0} />
          </Card>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <Card title="Duplicate Clusters">
            <DuplicatesTable items={duplicates?.duplicate_clusters || []} />
          </Card>
          <Card title="Hallucination Map">
            <div className="space-y-2 text-sm text-zinc-400">
              {heatmap?.heatmap_cells && heatmap.heatmap_cells.length > 0 ? (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333]">
                      <tr>
                        <th className="px-4 py-3 font-normal">Model</th>
                        <th className="px-4 py-3 font-normal">Category</th>
                        <th className="px-4 py-3 font-normal">Failure Rate</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[#222]">
                      {heatmap.heatmap_cells.slice(0, 10).map((row, i) => (
                        <tr key={i} className="hover:bg-[#0a0a0a] transition-colors">
                          <td className="px-4 py-3 text-zinc-200">{row.model}</td>
                          <td className="px-4 py-3 text-zinc-400">{row.category}</td>
                          <td className="px-4 py-3 text-[#ffb000]">{(row.failure_rate * 100).toFixed(1)}%</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className="text-zinc-600 text-xs uppercase tracking-widest p-4">{'>>'} No hallucination clusters.</p>
              )}
            </div>
          </Card>
        </div>
      </section>
    </main>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="border border-[#333] bg-black group hover:border-zinc-500 transition-colors">
      <div className="border-b border-[#333] px-4 py-3 flex items-center justify-between bg-[#0a0a0a]">
        <h2 className="text-sm font-bold tracking-widest text-zinc-300 uppercase">
          // {title}
        </h2>
        <span className="text-[10px] text-zinc-600 uppercase tracking-widest">INTELLIGENCE</span>
      </div>
      <div className="p-0">
        {children}
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-[#333] bg-black p-4 relative group hover:border-zinc-500 transition-colors">
      <p className="text-[10px] uppercase tracking-widest text-zinc-500 mb-4">[{label}]</p>
      <p className="text-2xl font-bold tracking-tight text-[#ffb000]">{value}</p>
    </div>
  );
}

function LeaksTable({ leaks }: { leaks: Array<{ pattern_type: string; cost_usd: number; event_count: number }> }) {
  return (
    <div className="text-xs">
      {leaks.length === 0 ? (
        <p className="text-zinc-600 uppercase tracking-widest p-4">{'>>'} No significant cost leaks.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333]">
              <tr>
                <th className="px-4 py-3 font-normal">Pattern</th>
                <th className="px-4 py-3 font-normal text-right">Events</th>
                <th className="px-4 py-3 font-normal text-right">Cost</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#222]">
              {leaks.slice(0, 5).map((leak, i) => (
                <tr key={i} className="hover:bg-[#0a0a0a] transition-colors">
                  <td className="px-4 py-3 text-zinc-200">{leak.pattern_type}</td>
                  <td className="px-4 py-3 text-right text-zinc-400">{leak.event_count}</td>
                  <td className="px-4 py-3 text-right text-red-500 font-bold">${leak.cost_usd.toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function GraveyardTable({ items, totalWaste }: { items: Array<{ fingerprint: string; total_cost_usd: number; event_count: number }>; totalWaste: number }) {
  return (
    <div className="text-xs">
      <div className="flex items-center justify-between text-[10px] uppercase tracking-widest text-zinc-500 px-4 py-2 border-b border-[#222]">
        <span>Total lost</span>
        <span className="text-red-500">${totalWaste.toFixed(2)}</span>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-left">
          <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333]">
            <tr>
              <th className="px-4 py-3 font-normal">Fingerprint</th>
              <th className="px-4 py-3 font-normal text-right">Events</th>
              <th className="px-4 py-3 font-normal text-right">Cost</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#222]">
            {items.slice(0, 5).map((item, i) => (
              <tr key={i} className="hover:bg-[#0a0a0a] transition-colors">
                <td className="px-4 py-3 font-mono text-zinc-500">{item.fingerprint.substring(0, 12)}…</td>
                <td className="px-4 py-3 text-right text-zinc-300">{item.event_count}</td>
                <td className="px-4 py-3 text-right text-[#ffb000]">${item.total_cost_usd.toFixed(2)}</td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={3} className="px-4 py-6 text-center text-zinc-600 uppercase tracking-widest">{'>>'} Prompt graveyard is empty.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function DuplicatesTable({ items }: { items: Array<{ fingerprint: string; count: number; redundant_cost_usd: number }> }) {
  return (
    <div className="text-xs">
      {items.length === 0 ? (
        <p className="text-zinc-600 uppercase tracking-widest p-4">{'>>'} No duplicate clusters.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333]">
              <tr>
                <th className="px-4 py-3 font-normal">Fingerprint</th>
                <th className="px-4 py-3 font-normal text-right">Count</th>
                <th className="px-4 py-3 font-normal text-right">Redundant</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#222]">
              {items.slice(0, 5).map((item, i) => (
                <tr key={i} className="hover:bg-[#0a0a0a] transition-colors">
                  <td className="px-4 py-3 font-mono text-zinc-500">{item.fingerprint.substring(0, 12)}…</td>
                  <td className="px-4 py-3 text-right text-zinc-300">{item.count}</td>
                  <td className="px-4 py-3 text-right text-red-500">${item.redundant_cost_usd.toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

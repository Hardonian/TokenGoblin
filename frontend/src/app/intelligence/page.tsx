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
  const [tenantId, setTenantId] = useState("demo-tenant");
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
        const res = await fetch(path, { headers: { "x-tenant-id": tenantId } });
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
    <main className="min-h-screen bg-background text-text-primary">
      <section className="mx-auto max-w-6xl px-6 py-12 space-y-8">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-accent">Intelligence</p>
            <h1 className="mt-2 text-3xl font-semibold text-white">Fleet Intelligence</h1>
            <p className="mt-2 text-sm text-text-secondary">Passive signals only — no agent modifications.</p>
          </div>
          <div className="flex items-center gap-2">
            <input
              value={tenantId}
              onChange={(e) => setTenantId(e.target.value)}
              className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-white outline-none"
              placeholder="tenant-id"
            />
            <button
              onClick={loadAll}
              className="rounded-lg bg-[#00ff9d] px-3 py-2 text-sm font-semibold text-black"
            >
              Refresh
            </button>
          </div>
        </div>

        {error && (
          <div className="rounded-xl border border-[#ff4d4d]/40 bg-[#1b0505] p-4 text-sm text-red-300">
            {error}
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
            <div className="space-y-2 text-sm text-text-secondary">
              {heatmap?.heatmap_cells && heatmap.heatmap_cells.length > 0 ? (
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr>
                      <th className="px-3 py-2 text-xs text-text-muted">Model</th>
                      <th className="px-3 py-2 text-xs text-text-muted">Category</th>
                      <th className="px-3 py-2 text-xs text-text-muted">Failure Rate</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-white/5">
                    {heatmap.heatmap_cells.slice(0, 10).map((row, i) => (
                      <tr key={i}>
                        <td className="px-3 py-2">{row.model}</td>
                        <td className="px-3 py-2">{row.category}</td>
                        <td className="px-3 py-2">{(row.failure_rate * 100).toFixed(1)}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p>No hallucination clusters.</p>
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
    <div className="rounded-2xl border border-border bg-surface p-5">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-white">{title}</h2>
        <span className="text-xs text-text-muted">INTELLIGENCE</span>
      </div>
      {children}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-surface p-4">
      <p className="text-xs text-text-muted">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-white">{value}</p>
    </div>
  );
}

function LeaksTable({ leaks }: { leaks: Array<{ pattern_type: string; cost_usd: number; event_count: number }> }) {
  return (
    <div className="space-y-2 text-sm">
      {leaks.length === 0 ? (
        <p className="text-text-muted">No significant cost leaks.</p>
      ) : (
        <table className="w-full text-left text-sm">
          <thead>
            <tr>
              <th className="px-3 py-2 text-xs text-text-muted">Pattern</th>
              <th className="px-3 py-2 text-xs text-text-muted text-right">Events</th>
              <th className="px-3 py-2 text-xs text-text-muted text-right">Cost</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {leaks.slice(0, 5).map((leak, i) => (
              <tr key={i}>
                <td className="px-3 py-2">{leak.pattern_type}</td>
                <td className="px-3 py-2 text-right">{leak.event_count}</td>
                <td className="px-3 py-2 text-right text-red-300">${leak.cost_usd.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function GraveyardTable({ items, totalWaste }: { items: Array<{ fingerprint: string; total_cost_usd: number; event_count: number }>; totalWaste: number }) {
  return (
    <div className="space-y-2 text-sm">
      <div className="flex items-center justify-between text-xs text-text-muted">
        <span>Total lost</span>
        <span>${totalWaste.toFixed(2)}</span>
      </div>
      <table className="w-full text-left text-sm">
        <thead>
          <tr>
            <th className="px-3 py-2 text-xs text-text-muted">Fingerprint</th>
            <th className="px-3 py-2 text-xs text-text-muted text-right">Events</th>
            <th className="px-3 py-2 text-xs text-text-muted text-right">Cost</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/5">
          {items.slice(0, 5).map((item, i) => (
            <tr key={i}>
              <td className="px-3 py-2 font-mono text-xs text-text-secondary">{item.fingerprint.substring(0, 12)}…</td>
              <td className="px-3 py-2 text-right">{item.event_count}</td>
              <td className="px-3 py-2 text-right text-white">${item.total_cost_usd.toFixed(2)}</td>
            </tr>
          ))}
          {items.length === 0 && (
            <tr>
              <td colSpan={3} className="px-3 py-4 text-text-muted">Prompt graveyard is empty.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function DuplicatesTable({ items }: { items: Array<{ fingerprint: string; count: number; redundant_cost_usd: number }> }) {
  return (
    <div className="space-y-2 text-sm">
      {items.length === 0 ? (
        <p className="text-text-muted">No duplicate clusters.</p>
      ) : (
        <table className="w-full text-left text-sm">
          <thead>
            <tr>
              <th className="px-3 py-2 text-xs text-text-muted">Fingerprint</th>
              <th className="px-3 py-2 text-xs text-text-muted text-right">Count</th>
              <th className="px-3 py-2 text-xs text-text-muted text-right">Redundant</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {items.slice(0, 5).map((item, i) => (
              <tr key={i}>
                <td className="px-3 py-2 font-mono text-xs text-text-secondary">{item.fingerprint.substring(0, 12)}…</td>
                <td className="px-3 py-2 text-right">{item.count}</td>
                <td className="px-3 py-2 text-right text-red-300">${item.redundant_cost_usd.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

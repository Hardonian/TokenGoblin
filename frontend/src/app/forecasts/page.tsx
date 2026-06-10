"use client";

import { useEffect, useRef, useState } from "react";

type Envelope<T> = {
  ok: boolean;
  status: string;
  data?: T;
  error?: { code: string; message: string };
};

type Forecast = {
  projected_spend_usd: number;
  confidence_interval_low_usd: number;
  confidence_interval_high_usd: number;
  daily_trend: Array<{ date: string; spend_usd: number }>;
};

export default function ForecastPage() {
  const [tenantId, setTenantId] = useState("demo-tenant");
  const [forecast, setForecast] = useState<Forecast | null>(null);
  const [error, setError] = useState<string | null>(null);
  const initialized = useRef(false);

  const load = async () => {
    setError(null);
    try {
      const res = await fetch("/v2/forecasts/spend", { headers: { "x-tenant-id": tenantId } });
      const payload: Envelope<Forecast> = await res.json();
      if (!res.ok || !payload?.ok) {
        throw new Error(payload?.error?.message || "Forecast failed");
      }
      setForecast(payload.data || null);
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
            <p className="text-xs font-semibold uppercase tracking-widest text-accent">Forecast</p>
            <h1 className="mt-2 text-3xl font-semibold text-white">Spend Forecast</h1>
            <p className="mt-2 text-sm text-text-secondary">Projected monthly spend and confidence intervals.</p>
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

        <div className="grid gap-4 md:grid-cols-3">
          <Metric label="Projected Spend" value={`$${(forecast?.projected_spend_usd ?? 0).toFixed(2)}`} highlight />
          <Metric label="Low" value={`$${(forecast?.confidence_interval_low_usd ?? 0).toFixed(2)}`} />
          <Metric label="High" value={`$${(forecast?.confidence_interval_high_usd ?? 0).toFixed(2)}`} />
        </div>

        <div className="rounded-2xl border border-border bg-surface p-5">
          <div className="mb-2 text-xs text-text-muted">Daily Trend</div>
          <div className="space-y-1 text-sm">
            {(forecast?.daily_trend || []).slice(-10).map((row, i) => (
              <div key={i} className="flex items-center justify-between">
                <span className="font-mono text-xs text-text-secondary">{row.date}</span>
                <span className="text-white">${row.spend_usd.toFixed(2)}</span>
              </div>
            ))}
            {!forecast?.daily_trend?.length && <p className="text-text-muted">No forecast data.</p>}
          </div>
        </div>
      </section>
    </main>
  );
}

function Metric({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="rounded-xl border border-border bg-surface p-4">
      <p className="text-xs text-text-muted">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${highlight ? "text-[#00ff9d]" : "text-white"}`}>{value}</p>
    </div>
  );
}

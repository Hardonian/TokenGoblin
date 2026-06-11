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
    const [forecast, setForecast] = useState<Forecast | null>(null);
  const [error, setError] = useState<string | null>(null);
  const initialized = useRef(false);

  const load = async () => {
    setError(null);
    try {
      const res = await fetch("/v2/forecasts/spend");
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
    <main className="min-h-screen bg-black text-zinc-300 font-mono pb-20 selection:bg-[#ffb000] selection:text-black">
      <section className="mx-auto max-w-[1400px] px-6 py-12 space-y-8">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between border-b border-[#333] pb-4">
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#ffb000] mb-2">[ Forecast ]</p>
            <h1 className="text-xl font-bold text-white tracking-widest uppercase">Spend Forecast</h1>
            <p className="mt-2 text-xs text-zinc-500 uppercase tracking-widest">{'>>'} Projected monthly spend and confidence intervals.</p>
          </div>
          
        </div>

        {error && (
          <div className="border border-red-900 bg-[#0a0000] p-4 text-xs text-red-500 font-bold uppercase tracking-widest">
            [ERR] {error}
          </div>
        )}

        <div className="grid gap-4 md:grid-cols-3">
          <Metric label="Projected Spend" value={`$${(forecast?.projected_spend_usd ?? 0).toFixed(2)}`} highlight />
          <Metric label="Low" value={`$${(forecast?.confidence_interval_low_usd ?? 0).toFixed(2)}`} />
          <Metric label="High" value={`$${(forecast?.confidence_interval_high_usd ?? 0).toFixed(2)}`} />
        </div>

        <div className="border border-[#333] bg-black">
          <div className="border-b border-[#333] px-4 py-3 bg-[#0a0a0a]">
            <h2 className="text-zinc-300 font-bold tracking-widest text-sm uppercase">{/* Daily_Trend.dat */}</h2>
          </div>
          <div className="p-4 space-y-2 text-sm">
            {(forecast?.daily_trend || []).slice(-10).map((row, i) => (
              <div key={i} className="flex items-center justify-between border-b border-[#222] pb-2 last:border-0 last:pb-0">
                <span className="font-mono text-xs text-zinc-500 uppercase tracking-widest">{row.date}</span>
                <span className="text-[#ffb000] font-bold">${row.spend_usd.toFixed(2)}</span>
              </div>
            ))}
            {!forecast?.daily_trend?.length && <p className="text-zinc-600 text-xs uppercase tracking-widest">{'>>'} No forecast data. Seed required.</p>}
          </div>
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

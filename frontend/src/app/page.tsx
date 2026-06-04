"use client";

import { useCallback, useEffect, useState } from "react";
import { formatMoney, formatInt } from "@/components/shared";

// ------------------------------------------------------------------
// API Types
// ------------------------------------------------------------------

type Envelope<T> = {
  ok: boolean;
  status: string;
  data?: T;
  error?: { code: string; message: string };
};

type ExecutiveScorecard = {
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

type SpendForecast = {
  projected_spend_usd: number;
  confidence_interval_low_usd: number;
  confidence_interval_high_usd: number;
  daily_trend: Array<{ date: string; spend_usd: number }>;
};

type CostLeak = {
  pattern_type: string;
  cost_usd: number;
  event_count: number;
  description: string;
};

type ZombieAgent = {
  worker_id: string;
  event_count: number;
  acceptance_rate: number;
  total_cost_usd: number;
};

type PromptGraveyardResult = {
  graveyard_prompts: Array<{
    fingerprint: string;
    total_cost_usd: number;
    acceptance_rate: number;
    event_count: number;
  }>;
  total_waste_usd: number;
  count: number;
};

type ModelStats = {
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

// ------------------------------------------------------------------
// Main Component
// ------------------------------------------------------------------

export default function CommandCenter() {
  const [tenant, setTenant] = useState("demo-tenant");
  const [scorecard, setScorecard] = useState<ExecutiveScorecard | null>(null);
  const [forecast, setForecast] = useState<SpendForecast | null>(null);
  const [costLeaks, setCostLeaks] = useState<CostLeak[]>([]);
  const [zombieAgents, setZombieAgents] = useState<ZombieAgent[]>([]);
  const [graveyard, setGraveyard] = useState<PromptGraveyardResult | null>(null);
  const [models, setModels] = useState<ModelStats[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchV2 = async <T,>(path: string): Promise<T | null> => {
    try {
      const res = await fetch(path, { headers: { "x-tenant-id": tenant } });
      const env: Envelope<T> = await res.json();
      return env.data || null;
    } catch (e) {
      console.error(`Failed to fetch ${path}`, e);
      return null;
    }
  };

  const loadAll = useCallback(async () => {
    setLoading(true);
    const [sc, fc, cl, za, gy, md] = await Promise.all([
      fetchV2<ExecutiveScorecard>("/v2/executive/scorecard"),
      fetchV2<SpendForecast>("/v2/forecasts/spend"),
      fetchV2<{ cost_leaks: CostLeak[] }>("/v2/intelligence/cost-leaks"),
      fetchV2<{ zombie_agents: ZombieAgent[] }>("/v2/intelligence/zombie-agents"),
      fetchV2<PromptGraveyardResult>("/v2/intelligence/prompt-graveyard"),
      fetchV2<{ models: ModelStats[] }>("/v2/analytics/models"),
    ]);
    
    if (sc) setScorecard(sc);
    if (fc) setForecast(fc);
    if (cl) setCostLeaks(cl.cost_leaks || []);
    if (za) setZombieAgents(za.zombie_agents || []);
    if (gy) setGraveyard(gy);
    if (md) setModels(md.models || []);
    
    setLoading(false);
  }, [tenant]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadAll();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadAll]);

  const seedDemo = async () => {
    setLoading(true);
    await fetch("/api/dashboard/seed", {
      method: "POST",
      headers: { "x-tenant-id": tenant },
    });
    await loadAll();
  };

  return (
    <main className="min-h-screen bg-[#0a0a0a] text-gray-300 font-sans selection:bg-[#00FF41] selection:text-black pb-20">
      {/* HEADER */}
      <header className="border-b border-[#1f1f1f] bg-black/50 backdrop-blur-md sticky top-0 z-50">
        <div className="max-w-[1400px] mx-auto px-6 py-4 flex flex-col md:flex-row justify-between items-center gap-4">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded bg-[#00FF41] flex items-center justify-center text-black font-black text-xl tracking-tighter">
              TG
            </div>
            <div>
              <h1 className="text-xl font-semibold text-white tracking-tight">TokenGoblin</h1>
              <p className="text-xs text-[#00FF41] uppercase tracking-[0.2em] font-mono">Command Center</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 text-xs font-mono">ID:</span>
              <input
                className="bg-[#141414] border border-[#2a2a2a] rounded text-white text-sm pl-9 pr-3 py-1.5 focus:outline-none focus:border-[#00FF41] transition-colors font-mono w-48"
                value={tenant}
                onChange={(e) => setTenant(e.target.value)}
              />
            </div>
            <button 
              onClick={seedDemo}
              className="bg-[#141414] hover:bg-[#1f1f1f] border border-[#2a2a2a] hover:border-gray-500 text-gray-300 text-sm px-4 py-1.5 rounded transition-all active:scale-95"
            >
              Seed Data
            </button>
            <button 
              onClick={loadAll}
              className="bg-[#00FF41] hover:bg-[#00cc33] text-black font-medium text-sm px-4 py-1.5 rounded shadow-[0_0_15px_rgba(0,255,65,0.2)] transition-all active:scale-95"
            >
              Refresh
            </button>
          </div>
        </div>
      </header>

      {loading && (
        <div className="h-1 bg-[#1a1a1a] w-full overflow-hidden">
          <div className="h-full bg-[#00FF41] w-1/3 animate-[slide_1.5s_infinite_ease-in-out]"></div>
        </div>
      )}

      <div className="max-w-[1400px] mx-auto px-6 pt-8 space-y-6">
        {/* TOP ROW: EXECUTIVE SCORECARD */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <ScoreCard 
            title="AI Maturity Grade" 
            value={scorecard?.grade || "N/A"} 
            subValue={`Score: ${scorecard?.maturity_score || 0}/100`}
            highlight
          />
          <ScoreCard 
            title="Total Spend (Forecast)" 
            value={`$${formatMoney(forecast?.projected_spend_usd || 0)}`} 
            subValue={`Waste: ${scorecard?.waste_pct.toFixed(1)}%`}
          />
          <ScoreCard 
            title="Agent Fleet ROI" 
            value={`${scorecard?.roi_multiplier?.toFixed(1) || 0}x`} 
            subValue={`${scorecard?.active_agents || 0} active workers`}
          />
          <ScoreCard 
            title="Platform Reliability" 
            value={`${(100 - (scorecard?.failure_rate_pct || 0)).toFixed(1)}%`} 
            subValue={`${formatInt(scorecard?.avg_latency_ms || 0)}ms avg latency`}
          />
        </div>

        {/* MIDDLE SECTION: THE WASTE RADAR */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            
            {/* Cost Leaks */}
            <div className="bg-[#0f0f0f] border border-[#2a2a2a] rounded-xl overflow-hidden shadow-xl">
              <div className="border-b border-[#2a2a2a] px-5 py-4 flex justify-between items-center bg-[#141414]">
                <h2 className="text-white font-medium flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse"></span>
                  Active Cost Leaks
                </h2>
                <span className="text-xs font-mono text-gray-500">INTELLIGENCE_ENGINE v2.0</span>
              </div>
              <div className="p-5">
                {costLeaks.length === 0 ? (
                  <p className="text-gray-500 text-sm">No significant cost leaks detected in the current window.</p>
                ) : (
                  <div className="grid gap-4">
                    {costLeaks.map((leak, i) => (
                      <div key={i} className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 bg-[#1a1a1a] border border-[#333] rounded-lg p-4 hover:border-red-900/50 transition-colors group">
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-red-400 font-mono text-xs border border-red-900/50 bg-red-900/10 px-2 py-0.5 rounded">
                              {leak.pattern_type}
                            </span>
                            <span className="text-gray-400 text-sm">{leak.event_count} events</span>
                          </div>
                          <p className="text-gray-300 text-sm mt-2">{leak.description}</p>
                        </div>
                        <div className="text-right whitespace-nowrap">
                          <p className="text-red-400 font-medium text-lg">-${formatMoney(leak.cost_usd)}</p>
                          <p className="text-xs text-gray-500 uppercase tracking-widest">Wasted</p>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Prompt Graveyard */}
            <div className="bg-[#0f0f0f] border border-[#2a2a2a] rounded-xl overflow-hidden shadow-xl">
              <div className="border-b border-[#2a2a2a] px-5 py-4 flex justify-between items-center bg-[#141414]">
                <h2 className="text-white font-medium flex items-center gap-2">
                  <span className="text-gray-400">🪦</span> Prompt Graveyard
                </h2>
                <span className="text-xs font-mono text-red-400">
                  Total Lost: ${formatMoney(graveyard?.total_waste_usd || 0)}
                </span>
              </div>
              <div className="p-0 overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead className="bg-[#111] text-gray-500 font-mono text-xs uppercase">
                    <tr>
                      <th className="px-5 py-3 font-medium">Prompt Fingerprint</th>
                      <th className="px-5 py-3 font-medium text-right">Event Count</th>
                      <th className="px-5 py-3 font-medium text-right">Cost Drain</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#2a2a2a]">
                    {(graveyard?.graveyard_prompts || []).slice(0, 5).map((prompt, i) => (
                      <tr key={i} className="hover:bg-[#141414] transition-colors">
                        <td className="px-5 py-4 font-mono text-xs text-gray-400">
                          {prompt.fingerprint.substring(0, 16)}...
                        </td>
                        <td className="px-5 py-4 text-right text-gray-300">{prompt.event_count}</td>
                        <td className="px-5 py-4 text-right text-red-400 font-medium">
                          ${formatMoney(prompt.total_cost_usd)}
                        </td>
                      </tr>
                    ))}
                    {graveyard?.count === 0 && (
                      <tr>
                        <td colSpan={3} className="px-5 py-8 text-center text-gray-500">
                          The graveyard is empty.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
            
          </div>

          <div className="space-y-6">
            
            {/* Zombie Agents */}
            <div className="bg-[#0f0f0f] border border-[#2a2a2a] rounded-xl overflow-hidden shadow-xl h-full">
              <div className="border-b border-[#2a2a2a] px-5 py-4 bg-[#141414]">
                <h2 className="text-white font-medium flex items-center gap-2">
                  <span className="text-green-500">🧟</span> Zombie Agents
                </h2>
              </div>
              <div className="p-5">
                {zombieAgents.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-48 text-center">
                    <p className="text-gray-500 text-sm">No low-value, high-activity agents detected.</p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {zombieAgents.map((zombie, i) => (
                      <div key={i} className="p-4 bg-[#1a1a1a] border border-[#333] rounded-lg">
                        <div className="flex justify-between items-center mb-2">
                          <span className="text-white font-medium">{zombie.worker_id}</span>
                          <span className="text-red-400 font-mono text-sm">${formatMoney(zombie.total_cost_usd)}</span>
                        </div>
                        <div className="w-full bg-[#222] rounded-full h-1.5 mb-1 mt-3">
                          <div className="bg-[#00FF41] h-1.5 rounded-full" style={{ width: `${zombie.acceptance_rate * 100}%` }}></div>
                        </div>
                        <div className="flex justify-between text-xs text-gray-500">
                          <span>Acceptance Rate</span>
                          <span>{(zombie.acceptance_rate * 100).toFixed(1)}%</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

          </div>
        </div>

        {/* BOTTOM SECTION: MODEL PERFORMANCE GRID */}
        <div className="bg-[#0f0f0f] border border-[#2a2a2a] rounded-xl overflow-hidden shadow-xl">
          <div className="border-b border-[#2a2a2a] px-5 py-4 flex justify-between items-center bg-[#141414]">
            <h2 className="text-white font-medium">Model Analytics Matrix</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="bg-[#111] text-gray-500 font-mono text-xs uppercase border-b border-[#2a2a2a]">
                <tr>
                  <th className="px-5 py-4 font-medium">Model / Provider</th>
                  <th className="px-5 py-4 font-medium text-right">Volume</th>
                  <th className="px-5 py-4 font-medium text-right">Spend</th>
                  <th className="px-5 py-4 font-medium text-right">Cost/Call</th>
                  <th className="px-5 py-4 font-medium text-right">Cost/Outcome</th>
                  <th className="px-5 py-4 font-medium text-right">Quality</th>
                  <th className="px-5 py-4 font-medium text-right">Latency</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#2a2a2a]">
                {models.map((m, i) => (
                  <tr key={i} className="hover:bg-[#141414] transition-colors">
                    <td className="px-5 py-4">
                      <div className="font-medium text-white">{m.model_id}</div>
                      <div className="text-xs text-gray-500">{m.provider}</div>
                    </td>
                    <td className="px-5 py-4 text-right text-gray-400">{formatInt(m.event_count)}</td>
                    <td className="px-5 py-4 text-right text-gray-300 font-mono">${formatMoney(m.total_cost_usd)}</td>
                    <td className="px-5 py-4 text-right text-gray-400 font-mono">${formatMoney(m.avg_cost_per_call)}</td>
                    <td className="px-5 py-4 text-right text-white font-mono font-medium">${formatMoney(m.cost_per_outcome)}</td>
                    <td className="px-5 py-4 text-right">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${m.acceptance_rate > 0.8 ? 'bg-green-900/20 text-[#00FF41]' : m.acceptance_rate > 0.5 ? 'bg-yellow-900/20 text-yellow-400' : 'bg-red-900/20 text-red-400'}`}>
                        {(m.acceptance_rate * 100).toFixed(1)}%
                      </span>
                    </td>
                    <td className="px-5 py-4 text-right text-gray-400">{formatInt(m.avg_latency_ms)}ms</td>
                  </tr>
                ))}
                {models.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-5 py-8 text-center text-gray-500">
                      No model data available. Seed demo data to populate.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
        
      </div>
      
      <style dangerouslySetInnerHTML={{__html: `
        @keyframes slide {
          0% { transform: translateX(-100%); }
          100% { transform: translateX(300%); }
        }
      `}} />
    </main>
  );
}

function ScoreCard({ title, value, subValue, highlight }: { title: string, value: string, subValue: string, highlight?: boolean }) {
  return (
    <div className={`bg-[#0f0f0f] border ${highlight ? 'border-[#00FF41]/30 shadow-[0_0_20px_rgba(0,255,65,0.05)]' : 'border-[#2a2a2a] shadow-lg'} rounded-xl p-5 flex flex-col justify-between group hover:border-[#444] transition-colors`}>
      <h3 className="text-gray-500 text-xs font-mono uppercase tracking-wider mb-2">{title}</h3>
      <p className={`text-3xl font-semibold mb-3 ${highlight ? 'text-[#00FF41]' : 'text-white'}`}>{value}</p>
      <div className="mt-auto flex items-center justify-between">
        <p className="text-xs text-gray-400">{subValue}</p>
        <div className={`w-8 h-8 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity ${highlight ? 'bg-[#00FF41]/10 text-[#00FF41]' : 'bg-[#1f1f1f] text-gray-400'}`}>
          →
        </div>
      </div>
    </div>
  );
}

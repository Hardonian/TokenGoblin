"use client";

import { useState } from "react";
import { formatMoney, formatInt } from "@/components/shared";
import useSWR from "swr";
import { motion, AnimatePresence } from "framer-motion";
import { GoblinSpinner } from "@/components/GoblinSpinner";
import { DemoMode } from "@/components/DemoMode";
import { OnboardingTour } from "@/components/OnboardingTour";
import { authFetcher, useAuth } from "@/lib/auth";

// ------------------------------------------------------------------
// API Types
// ------------------------------------------------------------------

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

type TokenEvent = {
  event_id: string;
  worker_id: string;
  provider: string;
  model_id: string;
  total_tokens: number;
  cost_estimate_usd: number;
  task_category: string;
  occurred_at: string;
};

// ------------------------------------------------------------------
// Fetcher
// ------------------------------------------------------------------
// Replaced with authFetcher from @/lib/auth

// ------------------------------------------------------------------
// Main Component
// ------------------------------------------------------------------

export default function CommandCenter() {
  const { apiKey, tenantId, isLoading: authLoading } = useAuth();
  const [toast, setToast] = useState<string | null>(null);
  
  // Real check for enterprise tier (we will add logic later, defaulting to true for now so we can see the data)
  const isPro = true;



  const { data: scorecard, mutate: mutSC } = useSWR<ExecutiveScorecard>(tenantId ? "/v2/executive/scorecard" : null, authFetcher);
  const { data: forecast, mutate: mutFC } = useSWR<SpendForecast>(tenantId ? "/v2/forecasts/spend" : null, authFetcher);
  const { data: clData, mutate: mutCL } = useSWR<{ cost_leaks: CostLeak[] }>(tenantId ? "/v2/intelligence/cost-leaks" : null, authFetcher);
  const { data: zaData, mutate: mutZA } = useSWR<{ zombie_agents: ZombieAgent[] }>(tenantId ? "/v2/intelligence/zombie-agents" : null, authFetcher);
  const { data: graveyard, mutate: mutGY } = useSWR<PromptGraveyardResult>(tenantId ? "/v2/intelligence/prompt-graveyard" : null, authFetcher);
  const { data: mdData, mutate: mutMD } = useSWR<{ models: ModelStats[] }>(tenantId ? "/v2/analytics/models" : null, authFetcher);
  const { data: eventsData, mutate: mutEvents } = useSWR<{ events: TokenEvent[] }>(tenantId ? "/api/dashboard/events" : null, authFetcher, { refreshInterval: 5000 });

  const costLeaks = clData?.cost_leaks || [];
  const zombieAgents = zaData?.zombie_agents || [];
  const models = mdData?.models || [];
  const events = eventsData?.events || [];
  const loading = authLoading || (!scorecard && !forecast && !clData && !zaData && !graveyard && !mdData && !eventsData);

  const loadAll = () => {
    mutSC();
    mutFC();
    mutCL();
    mutZA();
    mutGY();
    mutMD();
    mutEvents();
  };

  const seedDemo = async () => {
    await fetch("/api/dashboard/seed", {
      method: "POST",
      headers: {
        "x-tenant-id": tenantId || "",
        "Authorization": `Bearer ${apiKey || ""}`
      }
    });
    loadAll();
  };

  return (
    <main className="min-h-screen bg-black text-zinc-300 font-mono selection:bg-[#ffb000] selection:text-black pb-20">
      {/* HEADER */}
      <header className="border-b border-[#333] bg-black sticky top-0 z-30">
        <div className="max-w-[1400px] mx-auto px-6 py-4 flex flex-col md:flex-row justify-between items-center gap-4">
          <div className="flex items-center gap-4">
            <div className="text-[#10b981] font-black text-xl">
              [GOBLIN_CAVERN_OS]
            </div>
            <div>
              <h1 className="text-lg font-bold text-white tracking-widest uppercase">Chief Goblin&apos;s War Room</h1>
              <p className="text-xs text-[#ffb000] uppercase tracking-[0.2em]">Sniffing Tokens in Real-time</p>
            </div>
          </div>
          
          <div className="flex items-center gap-4">
            <button 
              onClick={() => window.location.href = "/keys"}
              className="bg-black hover:bg-[#111] border border-[#333] hover:border-zinc-500 text-[#ffb000] text-xs px-4 py-1.5 transition-all uppercase tracking-widest mr-2"
            >
              [ API Keys ]
            </button>
            <button 
              onClick={seedDemo}
              className="bg-black hover:bg-[#111] border border-[#333] hover:border-zinc-500 text-zinc-400 text-xs px-4 py-1.5 transition-all uppercase tracking-widest"
            >
              [ Seed ]
            </button>
            <button 
              onClick={() => {
                if (!tenantId) return;
                window.location.href = `/api/dashboard/export.csv`;
              }}
              className="bg-black hover:bg-[#111] border border-var(--color-accent-goblin) text-var(--color-accent-goblin) text-xs px-4 py-1.5 transition-all uppercase tracking-widest mr-2"
            >
              [ Export ]
            </button>
            <button 
              onClick={() => window.location.href = "/pricing/overrides"}
              className="bg-black hover:bg-[#111] border border-[#333] hover:border-zinc-500 text-zinc-400 text-xs px-4 py-1.5 transition-all uppercase tracking-widest mr-2"
            >
              [ Overrides ]
            </button>
            <button 
              onClick={loadAll}
              className="bg-[#ffb000] hover:bg-[#ff8c00] text-black font-bold text-xs px-4 py-1.5 transition-all uppercase tracking-widest"
            >
              [ Sync ]
            </button>
          </div>
        </div>

        {/* TOAST NOTIFICATION */}
        <AnimatePresence>
          {toast && (
            <motion.div
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              className="fixed top-20 left-1/2 -translate-x-1/2 z-50 bg-[#111] border border-var(--color-accent-goblin) px-6 py-3 rounded text-var(--color-accent-goblin) font-bold uppercase tracking-widest text-xs shadow-[0_0_15px_rgba(16,185,129,0.2)]"
            >
              {toast}
            </motion.div>
          )}
        </AnimatePresence>
      </header>

      {loading && (
        <div className="w-full flex justify-center py-4 bg-[#0a0a0a] border-b border-[#333]">
          <GoblinSpinner />
        </div>
      )}

      <div className="max-w-[1400px] mx-auto px-6 pt-8 space-y-8">
        
        {/* TOP ROW: EXECUTIVE SCORECARD */}
        <div>
          <div className="text-xs text-[#10b981] uppercase tracking-[0.3em] mb-4 border-b border-[#333] pb-2">
            :: The_Hoard_Status
          </div>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <ScoreCard 
              title="AI Maturity Grade" 
              value={scorecard?.grade || "N/A"} 
              subValue={`Score: ${scorecard?.maturity_score || 0}/100`}
              highlight
            />
            <ScoreCard 
              title="Total Spend (FCST)" 
              value={`$${formatMoney(forecast?.projected_spend_usd || 0)}`} 
              subValue={`Waste: ${scorecard?.waste_pct.toFixed(1)}%`}
            />
            <ScoreCard 
              title="Agent Fleet ROI" 
              value={`${scorecard?.roi_multiplier?.toFixed(1) || 0}x`} 
              subValue={`${scorecard?.active_agents || 0} workers`}
            />
            <ScoreCard 
              title="Sys_Reliability" 
              value={`${(100 - (scorecard?.failure_rate_pct || 0)).toFixed(1)}%`} 
              subValue={`${formatInt(scorecard?.avg_latency_ms || 0)}ms lag`}
            />
          </div>
        </div>

        {/* MIDDLE SECTION: THE WASTE RADAR */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-8">
            
            {/* Cost Leaks */}
            <div className="border border-[#333] bg-black">
              <div className="border-b border-[#333] px-4 py-3 flex justify-between items-center bg-[#0a0a0a]">
                <h2 className="text-zinc-300 font-bold tracking-widest text-sm flex items-center gap-2 uppercase">
                  <span className="w-1.5 h-1.5 bg-red-500 animate-pulse"></span>
                  Waste_Sniffer_Active
                </h2>
                <span className="text-xs text-zinc-600 uppercase">Goblin_Nose_v2.0</span>
              </div>
              <div className="p-4">
                {costLeaks.length === 0 ? (
                  <p className="text-zinc-600 text-xs uppercase tracking-widest">{'>>'} System clean. No leaks detected.</p>
                ) : (
                  <div className="grid gap-3">
                    {costLeaks.map((leak, i) => (
                      <div key={i} className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 bg-[#0a0a0a] border border-[#222] p-3 hover:border-red-900 transition-colors">
                        <div>
                          <div className="flex items-center gap-3">
                            <span className="text-red-500 font-bold text-xs">
                              [{leak.pattern_type.toUpperCase()}]
                            </span>
                            <span className="text-zinc-500 text-xs">Events: {leak.event_count}</span>
                          </div>
                          <p className="text-zinc-400 text-xs mt-2 font-mono">{leak.description}</p>
                        </div>
                        <div className="text-right whitespace-nowrap">
                          <p className="text-red-500 font-bold text-sm">-${formatMoney(leak.cost_usd)}</p>
                          <p className="text-[10px] text-zinc-600 uppercase tracking-widest">Drain</p>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Prompt Graveyard */}
            <div className="border border-[#333] bg-black">
              <div className="border-b border-[#333] px-4 py-3 flex justify-between items-center bg-[#0a0a0a]">
                <h2 className="text-zinc-300 font-bold tracking-widest text-sm uppercase">
                  [!] Graveyard_Dump
                </h2>
                <span className="text-xs text-red-500 font-bold tracking-widest">
                  Total Lost: ${formatMoney(graveyard?.total_waste_usd || 0)}
                </span>
              </div>
              <div className="p-0 overflow-x-auto relative">
                {!isPro && (
                  <div className="absolute inset-0 z-10 backdrop-blur-[4px] bg-black/60 flex flex-col items-center justify-center p-6 text-center">
                    <span className="text-[#10b981] text-3xl mb-2">🧌</span>
                    <h3 className="text-white font-bold uppercase tracking-widest mb-2">Chief Demands a Toll!</h3>
                    <p className="text-zinc-400 text-xs mb-4">You need a Hoarder Subscription to see the Graveyard Forensics.</p>
                    <a href="/pricing" className="bg-[#10b981] text-black px-4 py-2 text-xs font-bold uppercase tracking-widest hover:bg-[#059669]">Pay the Chief</a>
                  </div>
                )}
                <table className="w-full text-xs text-left">
                  <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333]">
                    <tr>
                      <th className="px-4 py-3 font-normal">Fingerprint</th>
                      <th className="px-4 py-3 font-normal text-right">Events</th>
                      <th className="px-4 py-3 font-normal text-right">Cost</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#222]">
                    {(graveyard?.graveyard_prompts || []).slice(0, 5).map((prompt, i) => (
                      <tr key={i} className="hover:bg-[#0a0a0a] transition-colors">
                        <td className="px-4 py-3 text-zinc-400">
                          {prompt.fingerprint.substring(0, 16)}...
                        </td>
                        <td className="px-4 py-3 text-right text-zinc-300">{prompt.event_count}</td>
                        <td className="px-4 py-3 text-right text-red-500">
                          ${formatMoney(prompt.total_cost_usd)}
                        </td>
                      </tr>
                    ))}
                    {graveyard?.count === 0 && (
                      <tr>
                        <td colSpan={3} className="px-4 py-6 text-center text-zinc-600 uppercase tracking-widest">
                          {'>>'} Buffer empty.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
            
          </div>

          <div className="space-y-8">
            
            {/* Zombie Agents */}
            <div className="border border-[#333] bg-black h-full">
              <div className="border-b border-[#333] px-4 py-3 bg-[#0a0a0a]">
                <h2 className="text-zinc-300 font-bold tracking-widest text-sm uppercase">
                  [?] Zombie_Procs
                </h2>
              </div>
              <div className="p-4 relative min-h-[200px]">
                {!isPro && (
                  <div className="absolute inset-0 z-10 backdrop-blur-[4px] bg-black/60 flex flex-col items-center justify-center p-6 text-center">
                    <span className="text-[#10b981] text-3xl mb-2">🧟‍♂️</span>
                    <h3 className="text-white font-bold uppercase tracking-widest mb-2">Toll Required</h3>
                    <p className="text-zinc-400 text-xs mb-4">Only Hoarders can track down Zombie Agents.</p>
                    <a href="/pricing" className="bg-white text-black px-4 py-2 text-xs font-bold uppercase tracking-widest hover:bg-zinc-200">Unlock Hoarder Tier</a>
                  </div>
                )}
                {zombieAgents.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-48 text-center">
                    <p className="text-zinc-600 text-xs uppercase tracking-widest">{'>>'} No dead agents found.</p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {zombieAgents.map((zombie, i) => (
                      <div key={i} className="p-3 bg-[#0a0a0a] border border-[#222]">
                        <div className="flex justify-between items-center mb-2">
                          <span className="text-zinc-300 font-bold text-xs">{zombie.worker_id}</span>
                          <span className="text-red-500 text-xs">-${formatMoney(zombie.total_cost_usd)}</span>
                        </div>
                        <div className="w-full bg-[#111] h-1 mb-1 mt-3">
                          <style>{`.zombie-bar-${i} { width: ${zombie.acceptance_rate * 100}%; }`}</style>
                          <div className={`bg-[#ffb000] h-1 zombie-bar-${i}`}></div>
                        </div>
                        <div className="flex justify-between text-[10px] text-zinc-600 uppercase tracking-widest">
                          <span>Acceptance</span>
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

        {/* BOTTOM SECTION: LIVE EVENT FEED & MODEL MATRIX */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          
          {/* Live Event Feed */}
          <div className="border border-[#333] bg-black">
            <div className="border-b border-[#333] px-4 py-3 flex justify-between items-center bg-[#0a0a0a]">
              <h2 className="text-zinc-300 font-bold tracking-widest text-sm uppercase flex items-center gap-2">
                <span className="w-1.5 h-1.5 bg-green-500 animate-pulse"></span>
                [LIVE] Token_Firehose
              </h2>
            </div>
            <div className="p-0 overflow-x-auto h-[400px] overflow-y-auto">
              <table className="w-full text-xs text-left">
                <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333] sticky top-0">
                  <tr>
                    <th className="px-4 py-3 font-normal">Worker / Model</th>
                    <th className="px-4 py-3 font-normal text-right">Tokens</th>
                    <th className="px-4 py-3 font-normal text-right">Cost</th>
                    <th className="px-4 py-3 font-normal text-right">Time</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#222]">
                  {events.slice(0, 20).map((e, i) => (
                    <tr key={e.event_id || i} className="hover:bg-[#0a0a0a] transition-colors">
                      <td className="px-4 py-3">
                        <div className="text-zinc-200 font-bold">{e.worker_id}</div>
                        <div className="text-[10px] text-zinc-500 uppercase">{e.model_id}</div>
                      </td>
                      <td className="px-4 py-3 text-right text-zinc-400">{formatInt(e.total_tokens)}</td>
                      <td className="px-4 py-3 text-right text-[#ffb000]">${formatMoney(e.cost_estimate_usd)}</td>
                      <td className="px-4 py-3 text-right text-zinc-500">
                        {new Date(e.occurred_at).toLocaleTimeString()}
                      </td>
                    </tr>
                  ))}
                  {events.length === 0 && (
                    <tr>
                      <td colSpan={4} className="px-4 py-6 text-center text-zinc-600 uppercase tracking-widest">
                        {'>>'} Waiting for events...
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Model Matrix */}
          <div className="border border-[#333] bg-black">
            <div className="border-b border-[#333] px-4 py-3 bg-[#0a0a0a]">
              <h2 className="text-zinc-300 font-bold tracking-widest text-sm uppercase">{/* Model_Matrix */}</h2>
            </div>
            <div className="overflow-x-auto h-[400px] overflow-y-auto">
              <table className="w-full text-xs text-left">
                <thead className="bg-[#111] text-zinc-500 uppercase tracking-wider border-b border-[#333] sticky top-0">
                  <tr>
                    <th className="px-4 py-3 font-normal">Provider/ID</th>
                    <th className="px-4 py-3 font-normal text-right">Vol</th>
                    <th className="px-4 py-3 font-normal text-right">Spend</th>
                    <th className="px-4 py-3 font-normal text-right">Cost/Call</th>
                    <th className="px-4 py-3 font-normal text-right text-white">Cost/Outcome</th>
                    <th className="px-4 py-3 font-normal text-right">Quality</th>
                    <th className="px-4 py-3 font-normal text-right">Lag</th>
                  </tr>
                </thead>
              <tbody className="divide-y divide-[#222]">
                {models.map((m, i) => (
                  <tr key={i} className="hover:bg-[#0a0a0a] transition-colors">
                    <td className="px-4 py-3">
                      <div className="text-zinc-200 font-bold">{m.model_id}</div>
                      <div className="text-[10px] text-zinc-500 uppercase">{m.provider}</div>
                    </td>
                    <td className="px-4 py-3 text-right text-zinc-400">{formatInt(m.event_count)}</td>
                    <td className="px-4 py-3 text-right text-zinc-300">${formatMoney(m.total_cost_usd)}</td>
                    <td className="px-4 py-3 text-right text-zinc-400">${formatMoney(m.avg_cost_per_call)}</td>
                    <td className="px-4 py-3 text-right text-[#ffb000] font-bold">${formatMoney(m.cost_per_outcome)}</td>
                    <td className="px-4 py-3 text-right">
                      <span className={`px-2 py-0.5 text-[10px] tracking-widest uppercase border ${m.acceptance_rate > 0.8 ? 'border-green-500/30 text-green-500' : m.acceptance_rate > 0.5 ? 'border-yellow-500/30 text-yellow-500' : 'border-red-500/30 text-red-500'}`}>
                        {(m.acceptance_rate * 100).toFixed(1)}%
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right text-zinc-400">{formatInt(m.avg_latency_ms)}ms</td>
                  </tr>
                ))}
                {models.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-4 py-6 text-center text-zinc-600 uppercase tracking-widest">
                      {'>>'} No data. Seed to proceed.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
        </div>
      </div>
       {/* Demo Mode & Onboarding */}
       <DemoMode />
       <OnboardingTour isOpen={true} onClose={() => {}} onComplete={() => {}} />
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
    <div className={`bg-black border ${highlight ? 'border-[#ffb000]' : 'border-[#333]'} p-4 flex flex-col justify-between group hover:border-zinc-500 transition-colors relative`}>
      {highlight && <div className="absolute top-0 left-0 w-full h-[1px] bg-[#ffb000]"></div>}
      <h3 className="text-zinc-500 text-[10px] uppercase tracking-widest mb-4">[{title}]</h3>
      <p className={`text-2xl font-bold mb-2 tracking-tight ${highlight ? 'text-[#ffb000]' : 'text-zinc-200'}`}>{value}</p>
      <div className="mt-auto flex items-center justify-between">
        <p className="text-[10px] text-zinc-600 uppercase tracking-widest">{subValue}</p>
        <div className={`opacity-0 group-hover:opacity-100 transition-opacity text-xs ${highlight ? 'text-[#ffb000]' : 'text-zinc-500'}`}>
          {'>'}
        </div>
      </div>
    </div>
  );
}

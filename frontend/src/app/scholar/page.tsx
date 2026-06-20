"use client";

import { useState } from "react";
import useSWR from "swr";
import { authFetcher, useAuth } from "@/lib/auth";
import { motion, AnimatePresence } from "framer-motion";

type ScholarInsights = {
  discovered_waste_patterns: string[];
  suggested_optimizations: string[];
  metrics: {
    prompts_analyzed: number;
    patterns_discovered: number;
    estimated_waste_percent: number;
  };
};

export default function ScholarPage() {
  const { tenantId, apiKey } = useAuth();
  const [training, setTraining] = useState(false);
  const [toast, setToast] = useState<string | null>(null);

  const { data: insights, mutate, isLoading } = useSWR<ScholarInsights>(
    tenantId ? "/v2/intelligence/insights" : null,
    authFetcher
  );

  const showToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 3000);
  };

  const handleTrain = async () => {
    if (!tenantId || !apiKey) return;
    setTraining(true);
    try {
      const res = await fetch("/api/admin/scholar/train", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "x-tenant-id": tenantId,
          Authorization: `Bearer ${apiKey}`,
        },
      });
      const data = await res.json();
      if (!res.ok || !data.ok) throw new Error(data.error?.message || "Training failed");
      showToast("[SUCCESS] Scholar has completed analysis of recent events.");
      mutate();
    } catch (err) {
      showToast(`[ERR] ${(err as Error).message}`);
    } finally {
      setTraining(false);
    }
  };

  return (
    <main className="flex-1 p-6 sm:p-12 max-w-6xl mx-auto w-full">
      {/* Header */}
      <div className="mb-10 flex flex-col sm:flex-row justify-between items-start sm:items-end gap-4">
        <div>
          <h1 className="text-3xl font-mono text-[#e5e5e5] mb-2 flex items-center gap-3">
            <span className="text-[#ffb000]">/</span>goblin_scholar
            {training && (
              <span className="flex h-3 w-3 relative">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[#ffb000] opacity-75"></span>
                <span className="relative inline-flex rounded-full h-3 w-3 bg-[#ffb000]"></span>
              </span>
            )}
          </h1>
          <p className="text-[#a1a1aa] font-mono text-sm max-w-xl">
            Self-improving AI agent that mines your request logs, discovers slop patterns, and suggests structural prompt optimizations to save tokens.
          </p>
        </div>
        <button
          onClick={handleTrain}
          disabled={training || isLoading}
          className="bg-[#000] border border-[#ffb000] text-[#ffb000] hover:bg-[#ffb000] hover:text-black transition-colors font-mono text-xs uppercase tracking-widest py-2 px-6 disabled:opacity-50"
        >
          {training ? "[ RUNNING_ANALYSIS... ]" : "[ TRIGGER_TRAINING_RUN ]"}
        </button>
      </div>

      <AnimatePresence>
        {toast && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            className="fixed top-20 left-1/2 -translate-x-1/2 z-50 bg-[#ffb000] text-black px-4 py-2 font-mono text-sm shadow-2xl"
          >
            {toast}
          </motion.div>
        )}
      </AnimatePresence>

      {isLoading ? (
        <div className="flex items-center justify-center p-20 font-mono text-[#71717a] text-sm tracking-widest">
          [LOADING_INSIGHTS]...
        </div>
      ) : !insights ? (
        <div className="flex items-center justify-center p-20 font-mono text-red-500 text-sm tracking-widest">
          [ERR] UNABLE TO FETCH SCHOLAR DATA
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Metrics Column */}
          <div className="lg:col-span-1 flex flex-col gap-6">
            <div className="bg-[#0a0a0a] border border-[#333] p-6 relative overflow-hidden group">
              <div className="absolute top-0 right-0 w-32 h-32 bg-[#ffb000]/5 rounded-bl-full -z-0 transition-transform group-hover:scale-110"></div>
              <h3 className="text-xs font-mono text-[#a1a1aa] mb-1 tracking-widest uppercase relative z-10">
                PROMPTS_MINED
              </h3>
              <p className="text-3xl font-mono text-white relative z-10">
                {insights.metrics.prompts_analyzed.toLocaleString()}
              </p>
            </div>
            <div className="bg-[#0a0a0a] border border-[#333] p-6 relative overflow-hidden group">
              <div className="absolute top-0 right-0 w-32 h-32 bg-green-500/5 rounded-bl-full -z-0 transition-transform group-hover:scale-110"></div>
              <h3 className="text-xs font-mono text-[#a1a1aa] mb-1 tracking-widest uppercase relative z-10">
                PATTERNS_DISCOVERED
              </h3>
              <p className="text-3xl font-mono text-green-400 relative z-10">
                {insights.metrics.patterns_discovered}
              </p>
            </div>
            <div className="bg-[#0a0a0a] border border-[#333] p-6 relative overflow-hidden group">
              <div className="absolute top-0 right-0 w-32 h-32 bg-red-500/5 rounded-bl-full -z-0 transition-transform group-hover:scale-110"></div>
              <h3 className="text-xs font-mono text-[#a1a1aa] mb-1 tracking-widest uppercase relative z-10">
                EST_WASTE_PERCENT
              </h3>
              <p className="text-3xl font-mono text-red-400 relative z-10">
                {insights.metrics.estimated_waste_percent}%
              </p>
            </div>
          </div>

          {/* Insights Column */}
          <div className="lg:col-span-2 flex flex-col gap-6">
            <div className="bg-[#0a0a0a] border border-[#333]">
              <div className="border-b border-[#333] bg-[#111] p-3 px-6">
                <h3 className="text-xs font-mono text-[#ffb000] tracking-widest uppercase">
                  [!] Discovered_Waste_Patterns
                </h3>
              </div>
              <div className="p-6">
                {insights.discovered_waste_patterns.length === 0 ? (
                  <p className="text-sm font-mono text-[#71717a]">No slop patterns discovered yet.</p>
                ) : (
                  <ul className="space-y-4">
                    {insights.discovered_waste_patterns.map((pattern, idx) => (
                      <li key={idx} className="flex items-start gap-4 text-sm font-mono text-[#e5e5e5]">
                        <span className="text-red-500 font-bold shrink-0">[{idx + 1}]</span>
                        <span className="bg-[#1a0f0f] border border-red-900/30 p-2 block w-full leading-relaxed">
                          {pattern}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>

            <div className="bg-[#0a0a0a] border border-[#333]">
              <div className="border-b border-[#333] bg-[#111] p-3 px-6">
                <h3 className="text-xs font-mono text-green-400 tracking-widest uppercase">
                  [+] Suggested_Optimizations
                </h3>
              </div>
              <div className="p-6">
                {insights.suggested_optimizations.length === 0 ? (
                  <p className="text-sm font-mono text-[#71717a]">No optimizations available.</p>
                ) : (
                  <ul className="space-y-4">
                    {insights.suggested_optimizations.map((opt, idx) => (
                      <li key={idx} className="flex items-start gap-4 text-sm font-mono text-[#e5e5e5]">
                        <span className="text-green-500 font-bold shrink-0">[{idx + 1}]</span>
                        <span className="bg-[#0f1a0f] border border-green-900/30 p-2 block w-full leading-relaxed">
                          {opt}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}

"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth";
import Link from "next/link";
import { ArrowLeft, Zap, FileText, ChevronRight } from "lucide-react";

export default function GoblinRefinerPage() {
  const { tenantId, apiKey } = useAuth();
  const [inputPrompt, setInputPrompt] = useState("");
  const [refinedPrompt, setRefinedPrompt] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [stats, setStats] = useState<{
    original_length: number;
    refined_length: number;
    savings_percent: number;
  } | null>(null);

  const handleRefine = async () => {
    if (!inputPrompt.trim() || !tenantId || !apiKey) return;

    setIsLoading(true);
    setRefinedPrompt("");
    setStats(null);

    try {
      const res = await fetch("/v2/intelligence/refine", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "x-tenant-id": tenantId,
          "Authorization": `Bearer ${apiKey}`
        },
        body: JSON.stringify({ prompt: inputPrompt }),
      });

      const data = await res.json();
      if (res.ok && data.status === "success") {
        setRefinedPrompt(data.data.refined_prompt);
        setStats({
          original_length: data.data.original_length,
          refined_length: data.data.refined_length,
          savings_percent: data.data.savings_percent,
        });
      } else {
        console.error("Refine error:", data.error);
        alert(data.error?.message || "Failed to refine prompt");
      }
    } catch (err) {
      console.error(err);
      alert("Network error.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans selection:bg-[#3fb950] selection:text-white">
      <header className="border-b border-[#30363d] bg-[#161b22] px-6 py-4 flex items-center justify-between sticky top-0 z-10 shadow-sm">
        <div className="flex items-center gap-4">
          <Link href="/" className="text-[#8b949e] hover:text-[#c9d1d9] transition-colors">
            <ArrowLeft size={20} />
          </Link>
          <div className="flex items-center gap-2">
            <span className="text-xl">👺</span>
            <h1 className="text-lg font-semibold tracking-tight text-white m-0">The Goblin Refiner</h1>
          </div>
        </div>
      </header>

      <main className="max-w-6xl mx-auto p-6 lg:p-8 space-y-8">
        <div className="text-center space-y-4 py-8">
          <h2 className="text-3xl md:text-4xl font-bold text-white tracking-tight">
            Stop feeding the <span className="text-[#f85149]">AI Slop Monster</span>.
          </h2>
          <p className="text-[#8b949e] max-w-2xl mx-auto text-lg leading-relaxed">
            Paste your bloated, polite, overly-verbose prompt below. The Goblin will chew off the fat, leaving only the dense instructions that the LLM actually needs. Save tokens, save money.
          </p>
        </div>

        <div className="grid md:grid-cols-2 gap-6">
          {/* Input Panel */}
          <div className="bg-[#161b22] border border-[#30363d] rounded-xl overflow-hidden shadow-sm flex flex-col">
            <div className="px-4 py-3 border-b border-[#30363d] bg-[#0d1117] flex justify-between items-center">
              <span className="font-semibold text-white flex items-center gap-2">
                <FileText size={16} className="text-[#8b949e]" /> Raw Prompt
              </span>
              <span className="text-xs text-[#8b949e] font-mono">{inputPrompt.length} chars</span>
            </div>
            <textarea
              className="flex-1 w-full bg-transparent text-[#c9d1d9] p-4 resize-none outline-none focus:ring-1 focus:ring-[#3fb950] font-mono text-sm leading-relaxed min-h-[400px]"
              placeholder={`e.g. "Please, as an AI language model, if you don't mind, I would like you to literally and basically write a function..."`}
              value={inputPrompt}
              onChange={(e) => setInputPrompt(e.target.value)}
            />
            <div className="p-4 bg-[#0d1117] border-t border-[#30363d] flex justify-end">
              <button
                onClick={handleRefine}
                disabled={isLoading || !inputPrompt.trim()}
                className="bg-[#2ea043] hover:bg-[#3fb950] disabled:bg-[#2ea043]/50 disabled:cursor-not-allowed text-white px-6 py-2.5 rounded-md font-semibold flex items-center gap-2 transition-all"
              >
                {isLoading ? (
                  <div className="animate-spin h-5 w-5 border-2 border-white border-t-transparent rounded-full" />
                ) : (
                  <>
                    Goblinize It <Zap size={16} fill="currentColor" />
                  </>
                )}
              </button>
            </div>
          </div>

          {/* Output Panel */}
          <div className="bg-[#161b22] border border-[#30363d] rounded-xl overflow-hidden shadow-sm flex flex-col relative">
            <div className="px-4 py-3 border-b border-[#30363d] bg-[#0d1117] flex justify-between items-center">
              <span className="font-semibold text-white flex items-center gap-2">
                <span className="text-lg">💎</span> Refined Prompt
              </span>
              {stats && (
                <span className="text-xs font-mono text-[#3fb950] bg-[#3fb950]/10 px-2 py-1 rounded">
                  -{stats.savings_percent.toFixed(1)}% Waste
                </span>
              )}
            </div>
            
            {refinedPrompt ? (
              <div className="flex-1 p-4 overflow-auto">
                <pre className="font-mono text-sm leading-relaxed text-[#58a6ff] whitespace-pre-wrap">
                  {refinedPrompt}
                </pre>
              </div>
            ) : (
              <div className="flex-1 flex flex-col items-center justify-center text-[#8b949e] p-8 text-center space-y-4">
                <div className="w-16 h-16 rounded-full bg-[#0d1117] border border-[#30363d] flex items-center justify-center mb-2">
                  <ChevronRight size={24} className="text-[#30363d]" />
                </div>
                <p>The refined prompt will appear here.</p>
              </div>
            )}
            
            {stats && (
              <div className="p-4 bg-[#0d1117] border-t border-[#30363d]">
                 <div className="flex items-center justify-between text-sm">
                   <div className="space-y-1">
                     <p className="text-[#8b949e]">Original Length</p>
                     <p className="font-mono text-white">{stats.original_length}</p>
                   </div>
                   <div className="h-8 w-px bg-[#30363d]"></div>
                   <div className="space-y-1">
                     <p className="text-[#8b949e]">Refined Length</p>
                     <p className="font-mono text-white">{stats.refined_length}</p>
                   </div>
                   <div className="h-8 w-px bg-[#30363d]"></div>
                   <div className="space-y-1">
                     <p className="text-[#3fb950] font-semibold">Tokens Saved</p>
                     <p className="font-mono text-[#3fb950]">~{Math.round((stats.original_length - stats.refined_length) / 4)}</p>
                   </div>
                 </div>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

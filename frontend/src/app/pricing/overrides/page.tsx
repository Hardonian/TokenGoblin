"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth";
import Link from "next/link";
import { ArrowLeft, Save } from "lucide-react";

export default function PricingOverridesPage() {
  const { tenantId, apiKey } = useAuth();
  const [provider, setProvider] = useState("");
  const [modelId, setModelId] = useState("");
  const [promptPrice, setPromptPrice] = useState("");
  const [completionPrice, setCompletionPrice] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tenantId || !apiKey) return;
    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const res = await fetch("/v1/pricing/overrides", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "x-tenant-id": tenantId,
          "Authorization": `Bearer ${apiKey}`
        },
        body: JSON.stringify({
          provider,
          model_id: modelId,
          prompt_price_per_million: parseFloat(promptPrice),
          completion_price_per_million: parseFloat(completionPrice)
        })
      });

      const data = await res.json();
      if (!res.ok || !data.ok) {
        throw new Error(data.error?.message || "Failed to set override");
      }

      setSuccess(`Override for ${provider}/${modelId} saved successfully.`);
      setProvider("");
      setModelId("");
      setPromptPrice("");
      setCompletionPrice("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Network error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9] font-sans selection:bg-[#3fb950] selection:text-white">
      <header className="border-b border-[#30363d] bg-[#161b22] px-6 py-4 flex items-center justify-between sticky top-0 z-10 shadow-sm">
        <div className="flex items-center gap-4">
          <Link href="/pricing" className="text-[#8b949e] hover:text-[#c9d1d9] transition-colors">
            <ArrowLeft size={20} />
          </Link>
          <div className="flex items-center gap-2">
            <span className="text-xl">💰</span>
            <h1 className="text-lg font-semibold tracking-tight text-white m-0">Pricing Overrides</h1>
          </div>
        </div>
      </header>

      <main className="max-w-2xl mx-auto p-6 lg:p-8 space-y-8">
        <div className="bg-[#161b22] border border-[#30363d] rounded-xl overflow-hidden shadow-sm p-6">
          <h2 className="text-xl font-bold text-white mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-[#ffb000]"></span>
            Custom Model Pricing
          </h2>
          <p className="text-[#8b949e] text-sm mb-6">
            Force a custom pricing definition for a specific provider and model. Future token events will use these rates instead of the global defaults.
          </p>

          <form onSubmit={handleSave} className="space-y-4 font-mono text-sm">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-[#8b949e]">Provider</label>
                <input
                  type="text"
                  required
                  value={provider}
                  onChange={(e) => setProvider(e.target.value)}
                  placeholder="e.g. openai"
                  className="w-full bg-[#0d1117] border border-[#30363d] rounded px-3 py-2 text-[#c9d1d9] focus:border-[#ffb000] focus:outline-none transition-colors"
                />
              </div>
              <div className="space-y-2">
                <label className="text-[#8b949e]">Model ID</label>
                <input
                  type="text"
                  required
                  value={modelId}
                  onChange={(e) => setModelId(e.target.value)}
                  placeholder="e.g. gpt-4"
                  className="w-full bg-[#0d1117] border border-[#30363d] rounded px-3 py-2 text-[#c9d1d9] focus:border-[#ffb000] focus:outline-none transition-colors"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-[#8b949e]">Prompt Price (per 1M)</label>
                <div className="relative">
                  <span className="absolute left-3 top-2 text-[#8b949e]">$</span>
                  <input
                    type="number"
                    step="0.001"
                    required
                    value={promptPrice}
                    onChange={(e) => setPromptPrice(e.target.value)}
                    placeholder="30.00"
                    className="w-full bg-[#0d1117] border border-[#30363d] rounded pl-7 pr-3 py-2 text-[#c9d1d9] focus:border-[#ffb000] focus:outline-none transition-colors"
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-[#8b949e]">Completion Price (per 1M)</label>
                <div className="relative">
                  <span className="absolute left-3 top-2 text-[#8b949e]">$</span>
                  <input
                    type="number"
                    step="0.001"
                    required
                    value={completionPrice}
                    onChange={(e) => setCompletionPrice(e.target.value)}
                    placeholder="60.00"
                    className="w-full bg-[#0d1117] border border-[#30363d] rounded pl-7 pr-3 py-2 text-[#c9d1d9] focus:border-[#ffb000] focus:outline-none transition-colors"
                  />
                </div>
              </div>
            </div>

            {error && (
              <div className="bg-red-900/20 border border-red-900 text-red-400 p-3 rounded">
                [ERR] {error}
              </div>
            )}
            {success && (
              <div className="bg-green-900/20 border border-green-900 text-green-400 p-3 rounded">
                [OK] {success}
              </div>
            )}

            <div className="pt-4 flex justify-end">
              <button
                type="submit"
                disabled={loading}
                className="bg-[#ffb000] hover:bg-[#ff8c00] text-black font-bold px-6 py-2 rounded flex items-center gap-2 transition-colors disabled:opacity-50"
              >
                {loading ? "Saving..." : <><Save size={16} /> Save Override</>}
              </button>
            </div>
          </form>
        </div>
      </main>
    </div>
  );
}

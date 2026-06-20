"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";

export default function LoginPage() {
  const [apiKey, setApiKey] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const { login } = useAuth();
  
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch("/api/tenant/login", {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${apiKey}`
        }
      });

      const data = await res.json();
      if (!res.ok || !data.ok) {
        throw new Error(data.error?.message || "Login failed - Invalid API Key");
      }

      // Backend returns tenant_id in Data payload
      const tenantId = data.data?.tenant_id;
      if (!tenantId) throw new Error("No tenant ID returned from server");

      login(apiKey, tenantId);
      router.push("/");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="flex-1 flex items-center justify-center p-6 sm:p-12 relative overflow-hidden">
      {/* Decorative TUI Background Elements */}
      <div className="absolute top-0 left-0 w-full h-full pointer-events-none opacity-20">
        <div className="absolute top-10 left-10 text-xs text-[#ffb000]">
          [SYSTEM INITIALIZATION]... OK<br/>
          [LOADING MODULES]... OK<br/>
          [AWAITING AUTHENTICATION]...
        </div>
      </div>

      <div className="w-full max-w-md relative z-10">
        <div className="backdrop-blur-xl bg-[#0a0a0a]/80 border border-[#333333] shadow-2xl overflow-hidden rounded-md">
          {/* TUI Window Header */}
          <div className="bg-[#111111] border-b border-[#333333] p-2 flex items-center justify-between">
            <span className="text-xs text-[#a1a1aa] font-mono tracking-widest">
              SECURE_LOGIN_TERMINAL
            </span>
            <div className="flex gap-2">
              <div className="w-2.5 h-2.5 rounded-full bg-red-500/80"></div>
              <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/80"></div>
              <div className="w-2.5 h-2.5 rounded-full bg-green-500/80"></div>
            </div>
          </div>

          <div className="p-8">
            <h2 className="text-2xl mb-2 font-mono text-[#e5e5e5]">
              <span className="text-[#ffb000]">&gt;</span> AUTH_REQUIRED
            </h2>
            <p className="text-[#a1a1aa] text-sm mb-6 font-mono">
              Please enter your system API Key to continue.
            </p>

            <form onSubmit={handleLogin} className="space-y-4">
              <div>
                <label className="block text-xs text-[#71717a] font-mono mb-1 uppercase tracking-wider">
                  System API Key
                </label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  className="w-full bg-[#000000] border border-[#333333] focus:border-[#ffb000] rounded px-3 py-2 text-[#e5e5e5] font-mono text-sm outline-none transition-colors"
                  placeholder="tg_..."
                  required
                />
              </div>

              {error && (
                <div className="text-red-400 text-xs font-mono bg-red-900/20 border border-red-900 p-2 rounded">
                  [ERR] {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="w-full bg-[#ffb000] text-black hover:bg-[#ff8c00] transition-colors font-mono font-bold py-2 px-4 rounded disabled:opacity-50 mt-4"
              >
                {loading ? "AUTHENTICATING..." : "INITIATE_LOGIN()"}
              </button>
            </form>

            <div className="mt-6 pt-4 border-t border-[#333333] text-center">
              <span className="text-xs text-[#71717a] font-mono">
                No access? <a href="/signup" className="text-[#ffb000] hover:underline">/register_new_tenant</a>
              </span>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}

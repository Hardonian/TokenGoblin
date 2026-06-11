"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { registerTenant } from "@/lib/billing";
import { SiteFooter } from "@/components/layout";
import Link from "next/link";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export default function SignupPage() {
  const router = useRouter();
  const [tenantId, setTenantId] = useState("");
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{
    tenant_id: string;
    name: string;
    tier: string;
    api_key: string;
    created_at: string;
  } | null>(null);

  useEffect(() => {
    if (!result) return;
    const timer = setTimeout(() => {
      router.push(`/billing?tenant_id=${encodeURIComponent(result.tenant_id)}`);
    }, 4000);
    return () => clearTimeout(timer);
  }, [result, router]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const data = await registerTenant(tenantId, name);
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen bg-black text-zinc-300 font-mono selection:bg-[#ffb000] selection:text-black">
      <section className="mx-auto flex min-h-[85vh] max-w-md flex-col justify-center px-6">
        <div className="space-y-6 border border-[#333] bg-[#0a0a0a] p-8">
          <div className="space-y-2 border-b border-[#333] pb-4">
            <p className="text-xs font-bold uppercase tracking-widest text-[#ffb000]">
              [ TokenGoblin ]
            </p>
            <h1 className="text-xl font-bold text-white tracking-widest uppercase">
              Init Workspace
            </h1>
            <p className="text-xs text-zinc-500 uppercase tracking-widest">
              {'>>'} Allocate a free tenant slot.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="text-xs font-bold text-zinc-400 uppercase tracking-widest block mb-2">
                [ Tenant ID ]
              </label>
              <input
                value={tenantId}
                onChange={(e) =>
                  setTenantId(
                    e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "")
                  )
                }
                placeholder="company-id"
                className="h-10 w-full border border-[#333] bg-black px-3 text-sm text-[#ffb000] outline-none transition focus:border-[#ffb000]"
              />
              <p className="mt-2 text-[10px] text-zinc-600 uppercase tracking-widest">{/* Lowercase letters, numbers, hyphens */}</p>
            </div>
            <div>
              <label className="text-xs font-bold text-zinc-400 uppercase tracking-widest block mb-2">
                [ Organization ]
              </label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Acme Corp"
                className="h-10 w-full border border-[#333] bg-black px-3 text-sm text-[#ffb000] outline-none transition focus:border-[#ffb000]"
              />
            </div>
            {error && (
              <div className="border border-red-900 bg-[#0a0000] p-3 text-xs text-red-500 font-bold uppercase tracking-widest">
                [ERR] {error}
              </div>
            )}
            
            {result && (
              <div className="border border-green-900 bg-[#000a00] p-3 text-xs text-green-500 font-bold uppercase tracking-widest">
                [SUCCESS] Tenant created. Redirecting to console...
              </div>
            )}

            <button
              type="submit"
              disabled={loading || !tenantId || !name || !!result}
              className="mt-4 h-10 w-full bg-[#ffb000] text-xs font-bold uppercase tracking-widest text-black transition hover:bg-[#ff8c00] disabled:opacity-50"
            >
              {loading ? "[ Executing... ]" : "[ Allocate ]"}
            </button>
          </form>

          <div className="space-y-4 pt-4 border-t border-[#333]">
            <Link
              href="/pricing"
              className="flex w-full items-center justify-center border border-[#333] bg-black px-4 py-2 text-xs font-bold text-zinc-400 uppercase tracking-widest transition hover:border-[#ffb000] hover:text-[#ffb000]"
            >
              [ View Allocations ]
            </Link>
            <p className="text-center text-[10px] text-zinc-600 uppercase tracking-widest">
              By proceeding, you agree to our ToS & Privacy protocols.
            </p>
          </div>
        </div>
      </section>
      <SiteFooter />
    </main>
  );
}

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
    <main className="min-h-screen bg-background text-text-primary">
      <section className="mx-auto flex min-h-[85vh] max-w-md flex-col justify-center px-6">
        <div className="space-y-6">
          <div className="space-y-2">
            <p className="text-sm font-semibold uppercase tracking-[0.2em] text-[#00ff9d]">
              TokenGoblin
            </p>
            <h1 className="text-3xl font-semibold text-white">
              Create your workspace
            </h1>
            <p className="text-sm text-text-secondary">
              Start with a free tenant. Upgrade later when you need more
              capacity, forecasting, or dedicated support.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="text-sm font-medium text-text-secondary">
                Tenant ID
              </label>
              <input
                value={tenantId}
                onChange={(e) =>
                  setTenantId(
                    e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "")
                  )
                }
                placeholder="my-company"
                className="mt-1 h-11 w-full rounded-lg border border-border bg-surface px-3 text-sm text-white outline-none transition focus:border-[#00ff9d]"
              />
              <p className="mt-1 text-xs text-text-muted">
                Lowercase letters, numbers, and hyphens only
              </p>
            </div>
            <div>
              <label className="text-sm font-medium text-text-secondary">
                Company / project name
              </label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Acme Corp"
                className="mt-1 h-11 w-full rounded-lg border border-border bg-surface px-3 text-sm text-white outline-none transition focus:border-[#00ff9d]"
              />
            </div>
            {error && (
              <div className="rounded-lg border border-red-500/40 bg-[#1b0505] p-3 text-sm text-red-300">
                {error}
              </div>
            )}
            <button
              type="submit"
              disabled={loading || !tenantId || !name}
              className="h-11 w-full rounded-lg bg-[#00ff9d] text-sm font-semibold text-black transition hover:bg-[#00e08a] disabled:opacity-60"
            >
              {loading ? "Creating…" : "Create workspace"}
            </button>
          </form>

          <div className="space-y-2">
            <Link
              href="/pricing"
              className="flex w-full items-center justify-center rounded-lg border border-border px-4 py-2.5 text-sm font-semibold text-white transition hover:border-[#00ff9d] hover:text-[#00ff9d]"
            >
              Review plans
            </Link>
            <p className="text-center text-xs text-text-muted">
              By creating an account you agree to the Terms of Service and
              Privacy Policy.
            </p>
          </div>
        </div>
      </section>
      <SiteFooter />
    </main>
  );
}

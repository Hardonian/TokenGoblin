"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createCheckoutSession, getBillingStatus } from "@/lib/billing";
import { SiteFooter } from "@/components/layout";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

function BillingInner() {
  const search = useSearchParams();
  const tenantParam = search.get("tenant_id");
  const effectiveTenant = tenantParam || "";
  const [status, setStatus] = useState<{
    tenant_id: string;
    tier: string;
    stripe_customer_id?: string;
    current_month_cost_usd: number;
    usage_limit_usd: number;
    usage_percent: number;
    subscription_id?: string;
    needs_upgrade: boolean;
    near_limit: boolean;
    at_limit: boolean;
  } | null>(null);
    const [error, setError] = useState<string | null>(null);
  const [action, setAction] = useState<string | null>(null);

  

  const load = useCallback(async () => {
    if (!effectiveTenant) return;
    
    setError(null);
    try {
      const data = await getBillingStatus(effectiveTenant);
      setStatus(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Billing status failed");
    } finally {
      
    }
  }, [effectiveTenant]);

  useEffect(() => {
    if (!effectiveTenant) return;
    const timer = setTimeout(() => {
      void load();
    }, 0);
    return () => clearTimeout(timer);
  }, [effectiveTenant, load]);

  async function handleUpgrade(priceId?: string, planId?: string) {
    if (!effectiveTenant || !priceId) return;
    setAction("upgrade");
    setError(null);
    try {
      const data = await createCheckoutSession({
        tenantId: effectiveTenant,
        priceId,
        successUrl: `${window.location.origin}/billing?plan=${encodeURIComponent(planId || "")}`,
        cancelUrl: `${window.location.origin}/billing`,
      });
      if (data.checkout_url) {
        window.location.href = data.checkout_url;
        return;
      }
      setError("Checkout session did not return a URL.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Checkout failed");
    } finally {
      setAction(null);
    }
  }

  async function handlePortal() {
    if (!effectiveTenant) return;
    setAction("portal");
    setError(null);
    try {
      const res = await fetch("/api/billing/portal", {
        method: "POST",
        headers: {
          "content-type": "application/json",
          "x-tenant-id": effectiveTenant,
        },
        body: JSON.stringify({ return_url: `${window.location.origin}/billing` }),
      });
      const payload = await res.json();
      if (!res.ok || !payload.ok) {
        throw new Error(payload.error?.message || "Portal failed");
      }
      if (payload.data?.portal_url) {
        window.location.href = payload.data.portal_url;
        return;
      }
      setError("Portal session did not return a URL.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to open billing portal");
    } finally {
      setAction(null);
    }
  }

  const proPriceId = process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO;
  const enterprisePriceId = process.env.NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE;
  const planFromQuery = search.get("plan");

  return (
    <main className="min-h-screen bg-black text-zinc-300 font-mono pb-20 selection:bg-[#ffb000] selection:text-black">
      <section className="mx-auto max-w-[1000px] px-6 py-12 space-y-8">
        
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between border-b border-[#333] pb-4">
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#ffb000] mb-2">[ Billing ]</p>
            <h1 className="text-xl font-bold text-white tracking-widest uppercase">Subscription_Control</h1>
            <p className="mt-2 text-xs text-zinc-500 uppercase tracking-widest">{'>>'} Stripe integration & limits.</p>
          </div>
          
        </div>

        {error && (
          <div className="border border-red-900 bg-[#0a0000] p-4 text-xs text-red-500 font-bold uppercase tracking-widest">
            [ERR] {error}
          </div>
        )}

        {planFromQuery && (
          <div className="border border-[#ffb000] bg-[#ffb000]/10 p-4 text-xs text-[#ffb000] font-bold uppercase tracking-widest">
            [SYS] Selected plan: {planFromQuery}. Awaiting confirmation.
          </div>
        )}

        {status && (
          <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-4">
              <div className="border border-[#333] bg-black p-4 group hover:border-zinc-500 transition-colors">
                <p className="text-[10px] uppercase tracking-widest text-zinc-500 mb-4">[Plan_Tier]</p>
                <p className="text-2xl font-bold tracking-tight text-white capitalize">{status.tier}</p>
              </div>
              <div className="border border-[#333] bg-black p-4 group hover:border-zinc-500 transition-colors">
                <p className="text-[10px] uppercase tracking-widest text-zinc-500 mb-4">[Month_Usage]</p>
                <p className="text-2xl font-bold tracking-tight text-[#ffb000]">${status.current_month_cost_usd.toFixed(2)}</p>
              </div>
              <div className="border border-[#333] bg-black p-4 group hover:border-zinc-500 transition-colors">
                <p className="text-[10px] uppercase tracking-widest text-zinc-500 mb-4">[Hard_Limit]</p>
                <p className="text-2xl font-bold tracking-tight text-white">${status.usage_limit_usd.toFixed(0)}</p>
              </div>
              <div className="border border-[#333] bg-black p-4 group hover:border-zinc-500 transition-colors">
                <p className="text-[10px] uppercase tracking-widest text-zinc-500 mb-4">[Burn_Rate]</p>
                <p className={`text-2xl font-bold tracking-tight ${status.at_limit ? 'text-red-500' : 'text-white'}`}>
                  {status.usage_percent.toFixed(1)}%
                </p>
              </div>
            </div>

            <div className="space-y-2 border border-[#333] bg-black p-4">
              <div className="h-1 w-full overflow-hidden bg-[#111]">
                <div
                  className={`h-full transition-all duration-1000 ${
                    status.at_limit
                      ? "bg-red-500"
                      : status.near_limit
                        ? "bg-yellow-500"
                        : "bg-[#ffb000]"
                  }`}
                  style={{ width: `${Math.min(status.usage_percent, 100)}%` }}
                />
              </div>
              <div className="flex items-center justify-between text-[10px] text-zinc-600 uppercase tracking-widest">
                <span>[ Capacity_Status ]</span>
                {status.near_limit && !status.at_limit && (
                  <span className="text-yellow-500 font-bold">WARNING: Approaching limit</span>
                )}
                {status.at_limit && (
                  <span className="text-red-500 font-bold">CRITICAL: Limit reached</span>
                )}
              </div>
            </div>

            {status.tier !== "enterprise" && (
              <div className="border border-[#ffb000]/30 bg-black p-5 relative">
                <div className="absolute top-0 left-0 w-full h-[1px] bg-[#ffb000]"></div>
                <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                  <div>
                    <p className="text-sm font-bold text-white uppercase tracking-widest">Upgrade Protocol</p>
                    <p className="text-xs text-zinc-500 font-mono mt-1">
                      {'>>'} Unlock increased limits, deeper forecasts, and SLA support.
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    {status.tier === "free" && (
                      <button
                        disabled={action !== null || !proPriceId}
                        onClick={() => handleUpgrade(proPriceId, "pro")}
                        className="border border-[#ffb000] bg-[#ffb000]/10 hover:bg-[#ffb000] hover:text-black text-[#ffb000] px-4 py-2 text-xs font-bold transition-colors uppercase tracking-widest disabled:opacity-50"
                      >
                        {action === "upgrade" ? "[ Redirecting ]" : "[ Init: PRO ]"}
                      </button>
                    )}
                    <button
                      disabled={action !== null || !enterprisePriceId}
                      onClick={() => handleUpgrade(enterprisePriceId, "enterprise")}
                      className="border border-[#333] hover:border-zinc-400 text-zinc-300 px-4 py-2 text-xs font-bold transition-colors uppercase tracking-widest disabled:opacity-50"
                    >
                      {action === "upgrade" ? "[ Redirecting ]" : "[ Init: ENTERPRISE ]"}
                    </button>
                  </div>
                </div>
              </div>
            )}

            {status.subscription_id && (
              <div className="border border-[#333] bg-[#0a0a0a] p-5">
                <p className="text-[10px] font-bold uppercase tracking-widest text-zinc-500 mb-4">{/* Subscription_Context */}</p>
                <div className="space-y-3 text-xs text-zinc-400 font-mono">
                  <div className="flex items-center justify-between border-b border-[#222] pb-2">
                    <span className="uppercase tracking-widest">SUB_ID</span>
                    <span className="text-white">{status.subscription_id}</span>
                  </div>
                  {status.stripe_customer_id && (
                    <div className="flex items-center justify-between border-b border-[#222] pb-2">
                      <span className="uppercase tracking-widest">CUST_ID</span>
                      <span className="text-white">{status.stripe_customer_id}</span>
                    </div>
                  )}
                  <div className="pt-2">
                    <button
                      onClick={handlePortal}
                      disabled={action === "portal"}
                      className="inline-flex h-8 items-center border border-[#333] px-3 text-xs font-bold text-zinc-300 transition hover:border-[#ffb000] hover:text-[#ffb000] uppercase tracking-widest disabled:opacity-50"
                    >
                      {action === "portal" ? "[ Accessing... ]" : "[ Open_Portal ]"}
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </section>
      <SiteFooter />
    </main>
  );
}

export default function BillingPage() {
  return (
    <Suspense fallback={<div className="p-6 text-[#ffb000] font-mono text-xs uppercase">[Loading_Module...]</div>}>
      <BillingInner />
    </Suspense>
  );
}

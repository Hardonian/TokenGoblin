"use client";

import { Suspense, useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createCheckoutSession, getBillingStatus } from "@/lib/billing";
import { SiteFooter } from "@/components/layout";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

function BillingInner() {
  const search = useSearchParams();
  const tenantParam = search.get("tenant_id");
  const [tenantId, setTenantId] = useState("");
  const [effectiveTenant, setEffectiveTenant] = useState("");
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [action, setAction] = useState<string | null>(null);
  const [successPlan, setSuccessPlan] = useState<string | null>(null);

  useEffect(() => {
    if (tenantParam) {
      setTenantId(tenantParam);
      setEffectiveTenant(tenantParam);
    }
  }, [tenantParam]);

  async function load() {
    if (!effectiveTenant) return;
    setLoading(true);
    setError(null);
    try {
      const data = await getBillingStatus(effectiveTenant);
      setStatus(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Billing status failed");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!effectiveTenant) return;
    const timer = setTimeout(() => {
      void load();
    }, 0);
    return () => clearTimeout(timer);
  }, [effectiveTenant]);

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
    <main className="min-h-screen bg-background text-text-primary">
      <section className="mx-auto max-w-[1000px] px-6 py-12">
        {error && (
          <div className="mb-6 rounded-xl border border-[#ff4d4d]/40 bg-[#1b0505] p-4 text-sm text-red-300">
            {error}
          </div>
        )}

        {planFromQuery && (
          <div className="mb-6 rounded-xl border border-[#00ff9d]/30 bg-accent-muted p-4 text-sm text-[#00ff9d]">
            Selected plan: {planFromQuery}. You can upgrade below or manage your subscription in-place.
          </div>
        )}

        <div className="rounded-2xl border border-border bg-surface p-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs font-semibold uppercase tracking-widest text-text-muted">
                Tenant
              </p>
              <input
                value={tenantId}
                onChange={(e) => {
                  setTenantId(e.target.value);
                  setEffectiveTenant(e.target.value);
                }}
                placeholder="demo-tenant"
                className="mt-1 h-10 w-full max-w-xs rounded-lg border border-border bg-[#0a0a0a] px-3 text-sm text-white outline-none transition focus:border-[#00ff9d]"
              />
              <button
                onClick={load}
                disabled={loading || !effectiveTenant}
                className="mt-2 inline-flex h-9 items-center rounded-lg bg-white px-3 text-xs font-semibold text-black transition hover:bg-zinc-200 disabled:opacity-60"
              >
                {loading ? "Loading" : "Load billing"}
              </button>
            </div>

            <div className="text-right">
              <p className="text-xs text-text-muted">Stripe</p>
              <p className="text-sm text-text-secondary">Payments, portal, and invoices are managed securely by Stripe.</p>
            </div>
          </div>

          {status && (
            <div className="mt-6 space-y-6">
              <div className="grid gap-4 md:grid-cols-4">
                <div className="rounded-xl border border-border bg-[#0a0a0a] p-4">
                  <p className="text-xs text-text-muted">Current plan</p>
                  <p className="mt-1 text-2xl font-semibold text-white capitalize">{status.tier}</p>
                </div>
                <div className="rounded-xl border border-border bg-[#0a0a0a] p-4">
                  <p className="text-xs text-text-muted">Monthly usage</p>
                  <p className="mt-1 text-2xl font-semibold text-white">
                    ${status.current_month_cost_usd.toFixed(2)}
                  </p>
                </div>
                <div className="rounded-xl border border-border bg-[#0a0a0a] p-4">
                  <p className="text-xs text-text-muted">Limit</p>
                  <p className="mt-1 text-2xl font-semibold text-white">
                    ${status.usage_limit_usd.toFixed(0)}
                  </p>
                </div>
                <div className="rounded-xl border border-border bg-[#0a0a0a] p-4">
                  <p className="text-xs text-text-muted">Consumed</p>
                  <p className="mt-1 text-2xl font-semibold text-white">
                    {status.usage_percent.toFixed(1)}%
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <div className="h-2 w-full overflow-hidden rounded-full bg-[#13151a]">
                  <div
                    className={`h-full rounded-full transition-all duration-1000 ${
                      status.at_limit
                        ? "bg-red-500"
                        : status.near_limit
                          ? "bg-amber-400"
                          : "bg-[#00ff9d]"
                    }`}
                    style={{ width: `${Math.min(status.usage_percent, 100)}%` }}
                  />
                </div>
                <div className="flex items-center justify-between text-xs text-zinc-400">
                  <span>Usage</span>
                  {status.near_limit && !status.at_limit && (
                    <span className="text-amber-300">Approaching limit</span>
                  )}
                  {status.at_limit && (
                    <span className="text-red-300">At limit</span>
                  )}
                </div>
              </div>

              {status.tier !== "enterprise" && (
                <div className="rounded-xl border border-[#00ff9d]/40 bg-accent-muted p-5">
                  <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                    <div>
                      <p className="text-sm font-semibold text-white">Upgrade capacity or governance</p>
                      <p className="text-xs text-zinc-400">
                        Increase usage, unlock forecasts, or move to dedicated support.
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {status.tier === "free" && (
                        <button
                          disabled={action !== null || !proPriceId}
                          onClick={() => handleUpgrade(proPriceId, "pro")}
                          className="rounded-lg bg-[#00ff9d] px-4 py-2 text-xs font-semibold text-black transition hover:bg-[#00e08a] disabled:opacity-60"
                        >
                          {action === "upgrade" ? "Redirecting…" : "Upgrade to Pro"}
                        </button>
                      )}
                      <button
                        disabled={action !== null || !enterprisePriceId}
                        onClick={() => handleUpgrade(enterprisePriceId, "enterprise")}
                        className="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-white transition hover:border-[#00ff9d] hover:text-[#00ff9d] disabled:opacity-60"
                      >
                        {action === "upgrade" ? "Redirecting…" : "Upgrade to Enterprise"}
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {status.subscription_id && (
                <div className="rounded-xl border border-border bg-surface p-5">
                  <p className="text-xs font-semibold uppercase tracking-widest text-text-muted">
                    Subscription details
                  </p>
                  <div className="mt-3 space-y-2 text-sm text-zinc-300">
                    <div className="flex items-center justify-between gap-4">
                      <span>Subscription</span>
                      <span className="font-mono text-xs text-white">{status.subscription_id}</span>
                    </div>
                    {status.stripe_customer_id && (
                      <div className="flex items-center justify-between gap-4">
                        <span>Customer</span>
                        <span className="font-mono text-xs text-white">{status.stripe_customer_id}</span>
                      </div>
                    )}
                    <button
                      onClick={handlePortal}
                      disabled={action === "portal"}
                      className="mt-2 inline-flex h-9 items-center rounded-lg border border-border px-3 text-xs font-semibold text-white transition hover:border-[#00ff9d] hover:text-[#00ff9d] disabled:opacity-60"
                    >
                      {action === "portal" ? "Opening portal…" : "Open billing portal"}
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </section>
      <SiteFooter />
    </main>
  );
}

export default function BillingPage() {
  return <BillingInner />;
}

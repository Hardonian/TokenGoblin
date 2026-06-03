"use client";

import { useCallback, useEffect, useState } from "react";
import UsageBar from "@/components/UsageBar";

type BillingStatus = {
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
};

const apiBase =
  process.env.NEXT_PUBLIC_TG_API_BASE?.replace(/\/$/, "") ||
  "http://localhost:8080";

export default function BillingPage() {
  const [tenant, setTenant] = useState("");
  const [status, setStatus] = useState<BillingStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [actionLoading, setActionLoading] = useState("");

  const load = useCallback(async () => {
    if (!tenant) return;
    setLoading(true);
    setError("");
    try {
      const res = await fetch(`${apiBase}/api/billing/status`, {
        headers: { "x-tenant-id": tenant },
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data?.error?.message || "Failed to load billing status");
        return;
      }
      setStatus(data.data);
    } catch {
      setError("Could not connect to server");
    } finally {
      setLoading(false);
    }
  }, [tenant]);

  useEffect(() => {
    if (tenant) load();
  }, [tenant, load]);

  async function handleUpgrade(priceID: string) {
    setActionLoading(priceID);
    try {
      const res = await fetch(`${apiBase}/api/billing/checkout`, {
        method: "POST",
        headers: {
          "content-type": "application/json",
          "x-tenant-id": tenant,
        },
        body: JSON.stringify({
          success_url: `${window.location.origin}/billing?success=true`,
          cancel_url: `${window.location.origin}/billing`,
          price_id: priceID,
        }),
      });
      const data = await res.json();
      if (data?.data?.checkout_url) {
        window.location.href = data.data.checkout_url;
      } else {
        setError(data?.error?.message || "Checkout failed");
      }
    } catch {
      setError("Could not start checkout");
    } finally {
      setActionLoading("");
    }
  }

  async function handlePortal() {
    setActionLoading("portal");
    try {
      const res = await fetch(`${apiBase}/api/billing/portal`, {
        method: "POST",
        headers: {
          "content-type": "application/json",
          "x-tenant-id": tenant,
        },
        body: JSON.stringify({
          return_url: `${window.location.origin}/billing`,
        }),
      });
      const data = await res.json();
      if (data?.data?.portal_url) {
        window.location.href = data.data.portal_url;
      } else {
        setError(data?.error?.message || "Portal access failed");
      }
    } catch {
      setError("Could not open billing portal");
    } finally {
      setActionLoading("");
    }
  }

  const pricePro =
    process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO || "price_pro_placeholder";
  const priceEnterprise =
    process.env.NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE || "price_enterprise_placeholder";

  return (
    <main className="min-h-screen bg-[#f7f8f3] text-[#171915]">
      <section className="border-b border-[#d7dccf] bg-[#fbfcf8]">
        <div className="mx-auto w-full max-w-4xl px-5 py-10">
          <h1 className="text-3xl font-bold">Billing & Usage</h1>
          <p className="mt-2 text-sm text-[#52604e]">
            Manage your subscription, view usage, and update payment methods.
          </p>
        </div>
      </section>

      <section className="mx-auto w-full max-w-4xl px-5 py-8">
        {/* Tenant input */}
        <div className="mb-8 rounded-xl border border-[#d7dccf] bg-white p-5">
          <label className="block text-sm font-medium">Tenant ID</label>
          <div className="mt-2 flex gap-3">
            <input
              className="h-10 flex-1 rounded-lg border border-[#c5cdbb] bg-white px-3 text-sm outline-none focus:border-[#426b51]"
              placeholder="Enter your tenant ID"
              value={tenant}
              onChange={(e) => setTenant(e.target.value)}
            />
            <button
              onClick={load}
              disabled={loading || !tenant}
              className="h-10 rounded-lg bg-[#171915] px-5 text-sm font-medium text-white disabled:opacity-50"
            >
              {loading ? "Loading…" : "Load"}
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-6 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            {error}
          </div>
        )}

        {status && (
          <div className="space-y-6">
            {/* Current plan */}
            <div className="rounded-xl border border-[#d7dccf] bg-white p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide text-[#61705a]">
                    Current Plan
                  </p>
                  <h2 className="mt-1 text-2xl font-bold capitalize">
                    {status.tier}
                  </h2>
                </div>
                {status.tier !== "enterprise" && (
                  <button
                    onClick={handlePortal}
                    disabled={actionLoading !== ""}
                    className="rounded-lg border border-[#d7dccf] px-4 py-2 text-sm font-medium text-[#52604e] hover:bg-[#f7f8f3] disabled:opacity-50"
                  >
                    Manage Billing
                  </button>
                )}
              </div>

              {/* Usage */}
              <div className="mt-6">
                <div className="flex items-baseline justify-between">
                  <span className="text-sm text-[#52604e]">Monthly Usage</span>
                  <span className="text-sm font-medium">
                    ${status.current_month_cost_usd.toFixed(2)} / $
                    {status.usage_limit_usd.toFixed(0)}
                  </span>
                </div>
                <div className="mt-2">
                  <UsageBar
                    percent={status.usage_percent}
                    label={status.at_limit ? "At limit" : status.near_limit ? "Near limit" : undefined}
                  />
                </div>
                {status.near_limit && !status.at_limit && (
                  <p className="mt-2 text-sm text-amber-600">
                    ⚠ You're approaching your monthly limit. Consider upgrading.
                  </p>
                )}
                {status.at_limit && (
                  <p className="mt-2 text-sm text-red-600">
                    ✕ You've reached your monthly limit. Upgrade to continue
                    tracking.
                  </p>
                )}
              </div>
            </div>

            {/* Upgrade options */}
            {status.needs_upgrade && (
              <div className="rounded-xl border border-[#426b51] bg-white p-6">
                <h3 className="text-lg font-semibold">Upgrade Your Plan</h3>
                <p className="mt-1 text-sm text-[#52604e]">
                  Get more events and advanced features.
                </p>
                <div className="mt-4 grid gap-4 sm:grid-cols-2">
                  {status.tier === "free" && (
                    <button
                      onClick={() => handleUpgrade(pricePro)}
                      disabled={actionLoading !== ""}
                      className="rounded-lg border border-[#426b51] p-4 text-left transition-colors hover:bg-[#426b51]/5 disabled:opacity-50"
                    >
                      <p className="font-semibold">Pro — $29/mo</p>
                      <p className="mt-1 text-xs text-[#52604e]">
                        100K events, output analysis, recommendations
                      </p>
                    </button>
                  )}
                  <button
                    onClick={() => handleUpgrade(priceEnterprise)}
                    disabled={actionLoading !== ""}
                    className="rounded-lg border border-[#d7dccf] p-4 text-left transition-colors hover:bg-[#f7f8f3] disabled:opacity-50"
                  >
                    <p className="font-semibold">Enterprise — $99/mo</p>
                    <p className="mt-1 text-xs text-[#52604e]">
                      Unlimited events, audit trail, RBAC, SLA
                    </p>
                  </button>
                </div>
              </div>
            )}

            {/* Subscription info */}
            {status.subscription_id && (
              <div className="rounded-xl border border-[#d7dccf] bg-white p-6">
                <h3 className="text-sm font-semibold uppercase tracking-wide text-[#61705a]">
                  Subscription Details
                </h3>
                <div className="mt-3 space-y-2 text-sm">
                  <p>
                    <span className="text-[#52604e]">Subscription ID:</span>{" "}
                    <code className="rounded bg-[#f7f8f3] px-2 py-0.5 font-mono text-xs">
                      {status.subscription_id}
                    </code>
                  </p>
                  {status.stripe_customer_id && (
                    <p>
                      <span className="text-[#52604e]">Customer ID:</span>{" "}
                      <code className="rounded bg-[#f7f8f3] px-2 py-0.5 font-mono text-xs">
                        {status.stripe_customer_id}
                      </code>
                    </p>
                  )}
                </div>
                <button
                  onClick={handlePortal}
                  disabled={actionLoading !== ""}
                  className="mt-4 rounded-lg bg-[#426b51] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[#365a43] disabled:opacity-50"
                >
                  {actionLoading === "portal"
                    ? "Opening…"
                    : "Open Billing Portal"}
                </button>
              </div>
            )}
          </div>
        )}

        {!status && !loading && tenant && !error && (
          <p className="text-center text-sm text-[#52604e]">
            No billing data found for this tenant.
          </p>
        )}
      </section>
    </main>
  );
}

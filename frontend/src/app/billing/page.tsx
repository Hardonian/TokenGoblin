"use client";

import { useCallback, useEffect, useState } from "react";

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
    <main className="min-h-screen bg-[#0a0a0a] text-gray-300 font-sans selection:bg-[#00FF41] selection:text-black">
      {/* HEADER */}
      <header className="border-b border-[#1f1f1f] bg-black/50 backdrop-blur-md sticky top-0 z-50">
        <div className="max-w-[1000px] mx-auto px-6 py-4 flex flex-col md:flex-row justify-between items-center gap-4">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded bg-[#00FF41] flex items-center justify-center text-black font-black text-xl tracking-tighter">
              TG
            </div>
            <div>
              <h1 className="text-xl font-semibold text-white tracking-tight">TokenGoblin</h1>
              <p className="text-xs text-[#00FF41] uppercase tracking-[0.2em] font-mono">Billing Center</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 text-xs font-mono">ID:</span>
              <input
                aria-label="Tenant ID"
                className="bg-[#141414] border border-[#2a2a2a] rounded text-white text-sm pl-9 pr-3 py-1.5 focus:outline-none focus:border-[#00FF41] transition-colors font-mono w-48"
                value={tenant}
                placeholder="Enter tenant ID"
                onChange={(e) => setTenant(e.target.value)}
              />
            </div>
            <button 
              onClick={load}
              disabled={loading || !tenant}
              className="bg-[#00FF41] hover:bg-[#00cc33] disabled:bg-[#141414] disabled:text-gray-500 disabled:border-[#2a2a2a] disabled:border border border-[#00FF41] text-black font-medium text-sm px-4 py-1.5 rounded shadow-[0_0_15px_rgba(0,255,65,0.2)] transition-all active:scale-95"
            >
              {loading ? "Loading..." : "Load"}
            </button>
          </div>
        </div>
      </header>

      <section className="max-w-[1000px] mx-auto px-6 py-12">
        {error && (
          <div className="mb-6 rounded-lg border border-red-900/50 bg-red-900/10 p-4 text-sm text-red-400">
            {error}
          </div>
        )}

        {status && (
          <div className="space-y-6">
            
            {/* CURRENT PLAN AND USAGE */}
            <div className="bg-[#0f0f0f] border border-[#2a2a2a] rounded-xl p-6 shadow-xl relative overflow-hidden">
              <div className="absolute top-0 right-0 p-6">
                {status.tier !== "enterprise" && (
                  <button
                    onClick={handlePortal}
                    disabled={actionLoading !== ""}
                    className="bg-[#141414] hover:bg-[#1f1f1f] border border-[#2a2a2a] text-gray-300 px-4 py-2 rounded text-sm font-medium transition-all"
                  >
                    Manage Billing
                  </button>
                )}
              </div>
              
              <div className="mb-8">
                <p className="text-gray-500 text-xs font-mono uppercase tracking-widest mb-1">Current Plan</p>
                <h2 className="text-4xl font-bold text-white capitalize">{status.tier}</h2>
              </div>

              <div>
                <div className="flex items-baseline justify-between mb-2">
                  <span className="text-gray-400 text-sm font-medium">Monthly Usage</span>
                  <span className="text-white font-mono">
                    ${status.current_month_cost_usd.toFixed(2)} / ${status.usage_limit_usd.toFixed(0)}
                  </span>
                </div>
                <div className="w-full bg-[#222] rounded-full h-2 mb-2 overflow-hidden">
                  <div 
                    className={`h-2 rounded-full transition-all duration-1000 ${status.at_limit ? 'bg-red-500' : status.near_limit ? 'bg-yellow-500' : 'bg-[#00FF41]'}`} 
                    style={(() => ({ width: `${Math.min(status.usage_percent, 100)}%` }))()}
                  ></div>
                </div>
                <div className="flex justify-between items-center text-xs">
                  <span className="text-gray-500">{status.usage_percent.toFixed(1)}% consumed</span>
                  {status.near_limit && !status.at_limit && (
                    <span className="text-yellow-500">⚠ Approaching limit</span>
                  )}
                  {status.at_limit && (
                    <span className="text-red-500 font-medium">✕ Limit reached</span>
                  )}
                </div>
              </div>
            </div>

            {/* UPGRADE OPTIONS */}
            {status.needs_upgrade && (
              <div className="bg-[#0f0f0f] border border-[#00FF41]/30 rounded-xl p-6 shadow-[0_0_20px_rgba(0,255,65,0.05)]">
                <h3 className="text-white text-lg font-semibold mb-1">Upgrade Your Plan</h3>
                <p className="text-gray-400 text-sm mb-6">Increase your limits and unlock full intelligence features.</p>
                
                <div className="grid gap-4 md:grid-cols-2">
                  {status.tier === "free" && (
                    <button
                      onClick={() => handleUpgrade(pricePro)}
                      disabled={actionLoading !== ""}
                      className="bg-[#141414] hover:bg-[#1a1a1a] border border-[#00FF41]/50 p-5 rounded-lg text-left transition-all hover:border-[#00FF41] group relative overflow-hidden"
                    >
                      <div className="absolute inset-0 bg-gradient-to-r from-[#00FF41]/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
                      <p className="text-white font-semibold text-lg mb-1">Pro — $29/mo</p>
                      <p className="text-gray-400 text-xs">100K events, AI forecasting, cost leak detection</p>
                    </button>
                  )}
                  <button
                    onClick={() => handleUpgrade(priceEnterprise)}
                    disabled={actionLoading !== ""}
                    className="bg-[#141414] hover:bg-[#1a1a1a] border border-[#2a2a2a] p-5 rounded-lg text-left transition-all hover:border-gray-500"
                  >
                    <p className="text-white font-semibold text-lg mb-1">Enterprise — $99/mo</p>
                    <p className="text-gray-400 text-xs">Unlimited events, governance, advanced SLAs</p>
                  </button>
                </div>
              </div>
            )}

            {/* SUBSCRIPTION DETAILS */}
            {status.subscription_id && (
              <div className="bg-[#0f0f0f] border border-[#2a2a2a] rounded-xl p-6">
                <h3 className="text-gray-500 text-xs font-mono uppercase tracking-widest mb-4">Subscription Details</h3>
                <div className="space-y-3">
                  <div className="flex justify-between items-center border-b border-[#1f1f1f] pb-2">
                    <span className="text-gray-400 text-sm">Subscription ID</span>
                    <span className="text-white font-mono text-xs bg-[#1a1a1a] px-2 py-1 rounded">{status.subscription_id}</span>
                  </div>
                  {status.stripe_customer_id && (
                    <div className="flex justify-between items-center pb-2">
                      <span className="text-gray-400 text-sm">Stripe Customer ID</span>
                      <span className="text-white font-mono text-xs bg-[#1a1a1a] px-2 py-1 rounded">{status.stripe_customer_id}</span>
                    </div>
                  )}
                </div>
              </div>
            )}

          </div>
        )}

        {!status && !loading && !error && tenant === "" && (
          <div className="h-64 flex items-center justify-center border border-[#1f1f1f] rounded-xl bg-[#0a0a0a]">
            <p className="text-gray-500 font-mono text-sm">Enter a tenant ID to view billing status.</p>
          </div>
        )}
      </section>
    </main>
  );
}

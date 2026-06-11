"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { createCheckoutSession } from "@/lib/billing";
import { SiteFooter } from "@/components/layout";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const plans = [
  {
    id: "free",
    name: "FREE",
    price: "$0",
    period: "forever",
    description: "Exploratory visibility for small operators.",
    features: [
      "1 tenant",
      "10,000 events/mo",
      "Dashboard & analytics",
      "CSV export",
      "Email support",
    ],
    highlighted: false,
    priceId: null,
  },
  {
    id: "pro",
    name: "PRO",
    price: "$29",
    period: "per month",
    description: "Forecasting and cost-leak analysis for active AI fleets.",
    features: [
      "5 tenants",
      "100,000 events/mo",
      "Spend forecasting",
      "Cost leak & zombie-agent detection",
      "Priority support & route recommendations",
    ],
    highlighted: true,
    priceId: process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO,
  },
  {
    id: "enterprise",
    name: "ENTERPRISE",
    price: "$99",
    period: "per month",
    description: "Unlimited observability for platform teams.",
    features: [
      "Unlimited tenants",
      "Unlimited events",
      "Custom pricing overrides & output analysis",
      "Audit trail & RBAC",
      "SLA & dedicated support",
    ],
    highlighted: false,
    priceId: process.env.NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE,
  },
];

export default function PricingPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
    const [annual, setAnnual] = useState(false);
  const [loadingPlan, setLoadingPlan] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const initializedPlanRef = useRef<string | null>(null);

  const handleStart = useCallback(async (plan: string) => {
    setLoadingPlan(plan);
    setError(null);

    if (plan === "free") {
      router.push("/signup");
      return;
    }

    try {
      const planConfig = plans.find((p) => p.id === plan);
      if (!planConfig) {
        throw new Error("Select a valid plan.");
      }

      const origin = typeof window !== "undefined" ? window.location.origin : "";
      const successUrl = `${origin}/billing?plan=${encodeURIComponent(plan)}`;
      const cancelUrl = `${origin}/pricing`;

      const source =
        planConfig.id === "pro"
          ? process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO
          : process.env.NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE;

      if (!source) {
        throw new Error("Stripe price id is not configured.");
      }

      const data = await createCheckoutSession({
        tenantId: "",
        priceId: source,
        successUrl,
        cancelUrl,
      });

      if (data.checkout_url) {
        window.location.href = data.checkout_url;
        return;
      }

      throw new Error("Checkout session did not return a URL.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to start checkout.");
    } finally {
      setLoadingPlan(null);
    }
  }, [router]);

  useEffect(() => {
    const plan = searchParams.get("plan");
    if (plan && initializedPlanRef.current !== plan) {
      initializedPlanRef.current = plan;
      handleStart(plan);
    }
  }, [searchParams, handleStart]);
  function renderPlan(plan: typeof plans[number]) {
    let price = plan.price;

    if (plan.id === "pro" && annual) {
      price = "$23";
    } else if (plan.id === "enterprise" && annual) {
      price = "$79";
    }

    const buttonLabel =
      plan.id === "free"
        ? "[ Allocate Free ]"
        : plan.id === "enterprise"
          ? "[ Init: Enterprise ]"
          : "[ Init: Pro Trial ]";

    return (
      <div
        key={plan.id}
        className={`relative flex flex-col border p-6 bg-black transition-colors ${
          plan.highlighted
            ? "border-[#ffb000]"
            : "border-[#333] hover:border-zinc-500"
        }`}
      >
        {plan.highlighted && (
          <div className="absolute top-0 left-0 w-full h-[1px] bg-[#ffb000]"></div>
        )}

        {plan.highlighted && (
          <div className="mb-4 text-[#ffb000] text-[10px] uppercase tracking-widest font-bold">
            {'>>'} Recommended Protocol
          </div>
        )}

        <div className="space-y-2">
          <p className="text-xs font-bold text-zinc-500 uppercase tracking-widest">
            [{plan.name}]
          </p>
          <div className="flex items-baseline gap-2">
            <span className={`text-4xl font-bold tracking-tight ${plan.highlighted ? "text-[#ffb000]" : "text-white"}`}>
              {price}
            </span>
            <span className="text-[10px] text-zinc-500 uppercase tracking-widest">
              /{annual && plan.id !== "free" ? "mo, billed annually" : plan.period}
            </span>
          </div>
          {plan.id !== "free" && (
            <p className="text-xs text-zinc-400 mt-2 font-mono">
              {plan.description}
            </p>
          )}
        </div>

        <ul className="mt-8 flex-1 space-y-4">
          {plan.features.map((feature) => (
            <li
              key={feature}
              className="flex gap-3 text-xs text-zinc-300 font-mono"
            >
              <span className="text-[#ffb000]">{'>>'}</span>
              {feature}
            </li>
          ))}
        </ul>

        <div className="mt-8">
          <button
            onClick={() => handleStart(plan.id)}
            disabled={loadingPlan === plan.id}
            className={`w-full px-4 py-3 text-xs font-bold uppercase tracking-widest transition-colors ${
              plan.highlighted
                ? "bg-[#ffb000] text-black hover:bg-[#ff8c00]"
                : "border border-[#333] text-zinc-300 hover:border-[#ffb000] hover:text-[#ffb000]"
            } disabled:opacity-50`}
          >
            {loadingPlan === plan.id ? "[ Redirecting... ]" : buttonLabel}
          </button>
        </div>
      </div>
    );
  }

  return (
    <main className="min-h-screen bg-black text-zinc-300 font-mono selection:bg-[#ffb000] selection:text-black">
      <section className="relative overflow-hidden border-b border-[#333] bg-[#050505]">
        <div className="absolute left-0 top-0 h-full w-full bg-[linear-gradient(rgba(255,176,0,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(255,176,0,0.03)_1px,transparent_1px)] bg-[size:20px_20px]" />
        <div className="relative mx-auto max-w-5xl px-6 pb-20 pt-24">
          <div className="mx-auto max-w-2xl text-center">
            <p className="text-[10px] font-bold uppercase tracking-[0.3em] text-[#ffb000]">
              [ Pricing_Matrix ]
            </p>
            <h1 className="mt-6 text-3xl font-bold tracking-widest text-white uppercase md:text-4xl">
              Autonomous Spend Control
            </h1>
            <p className="mt-6 text-sm text-zinc-400 font-mono">
              Start free. Upgrade when you&apos;re ready to move from data to action. 
              No hidden fees, no installer lock-in.
            </p>
          </div>

          <div className="mx-auto mt-12 flex max-w-sm items-center justify-center gap-1 border border-[#333] bg-black p-1">
            <button
              onClick={() => setAnnual(false)}
              className={`flex-1 py-2 text-xs font-bold uppercase tracking-widest transition-colors ${
                !annual
                  ? "bg-[#ffb000] text-black"
                  : "text-zinc-500 hover:text-white"
              }`}
            >
              [ Monthly ]
            </button>
            <button
              onClick={() => setAnnual(true)}
              className={`flex-1 py-2 text-xs font-bold uppercase tracking-widest transition-colors ${
                annual
                  ? "bg-[#ffb000] text-black"
                  : "text-zinc-500 hover:text-white"
              }`}
            >
              [ Annual ]
              <span className="block text-[8px] text-zinc-500">save 20%</span>
            </button>
          </div>
        </div>
      </section>

      {error && (
        <div className="mx-auto max-w-5xl px-6 pt-12">
          <div className="border border-red-900 bg-[#0a0000] p-4 text-xs text-red-500 font-bold uppercase tracking-widest">
            [ERR] {error}
          </div>
        </div>
      )}

      <section className="mx-auto max-w-[1200px] px-6 pb-24 pt-16">
        <div className="grid gap-6 md:grid-cols-3">{plans.map(renderPlan)}</div>
      </section>

      <section className="mx-auto max-w-4xl px-6 pb-24 border-t border-[#333] pt-24">
        <h2 className="text-xl font-bold text-white uppercase tracking-widest mb-12 text-center">
          [ Interrogation_Logs ]
        </h2>
        <div className="grid md:grid-cols-2 gap-8">
          {[
            {
              q: "What happens if I exceed my event limit?",
              a: "We’ll warn you before you burn the limit. Free operators see degraded estimates. Paid operators stay online with higher caps.",
            },
            {
              q: "Can I cancel anytime?",
              a: "Yes. Cancellation is immediate, and your data remains available in read-only mode for 30 days.",
            },
            {
              q: "Is there an API?",
              a: "Yes. All plans include API access. Pro and Enterprise get higher rate limits and output-level tooling.",
            },
            {
              q: "Do you integrate with StripedBank and Settler?",
              a: "Yes — for B2B pipelines, Settler is the canonical counterpart for Stripe, bank, and invoice reconciliation.",
            },
          ].map((faq, i) => (
            <div
              key={i}
              className="border border-[#333] bg-black p-6 group hover:border-zinc-500 transition-colors"
            >
              <h3 className="text-xs font-bold text-[#ffb000] uppercase tracking-widest mb-4">
                Q: {faq.q}
              </h3>
              <p className="text-xs text-zinc-400 font-mono leading-relaxed">
                {'>>'} {faq.a}
              </p>
            </div>
          ))}
        </div>
      </section>

      <SiteFooter />
    </main>
  );
}

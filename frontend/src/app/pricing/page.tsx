"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { createCheckoutSession, createBillingPortalSession, getBillingStatus } from "@/lib/billing";
import { SiteFooter } from "@/components/layout";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const plans = [
  {
    id: "free",
    name: "Free",
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
    name: "Pro",
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
    name: "Enterprise",
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
  const [tenantId, setTenantId] = useState("");
  const [annual, setAnnual] = useState(false);
  const [loadingPlan, setLoadingPlan] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const initializedPlanRef = useRef<string | null>(null);

  useEffect(() => {
    const plan = searchParams.get("plan");
    if (plan && initializedPlanRef.current !== plan) {
      initializedPlanRef.current = plan;
      handleStart(plan);
    }
  }, [searchParams]);

  async function handleStart(plan: string) {
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

      const origin =
        typeof window !== "undefined" ? window.location.origin : "";
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
        tenantId: tenantId || "",
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
      setError(
        err instanceof Error ? err.message : "Unable to start checkout."
      );
    } finally {
      setLoadingPlan(null);
    }
  }

  function renderPlan(plan: typeof plans[number]) {
    const isAnnualActive = annual;
    let price = plan.price;

    if (plan.id === "pro" && annual) {
      price = "$23";
    } else if (plan.id === "enterprise" && annual) {
      price = "$79";
    }

    const buttonLabel =
      plan.id === "free"
        ? "Create free account"
        : plan.id === "enterprise"
          ? "Contact sales"
          : "Start Pro trial";

    return (
      <div
        className={`relative flex flex-col rounded-2xl border p-6 ${
          plan.highlighted
            ? "border-[#00ff9d]/70 bg-surface-strong shadow-[0_0_35px_rgba(0,255,157,0.08)]"
            : "border-border bg-surface"
        }`}
      >
        {plan.id === "pro" && (
          <div className="pointer-events-none absolute -top-3 left-1/2 -translate-x-1/2">
            <span className="inline-flex rounded-full bg-[#00ff9d] px-3 py-1 text-xs font-semibold text-black">
              Most popular
            </span>
          </div>
        )}

        <div className="space-y-2">
          <p className="text-sm font-semibold text-text-secondary">
            {plan.name}
          </p>
          <div className="flex items-baseline gap-2">
            <span className="text-4xl font-semibold text-text-primary">
              {price}
            </span>
            <span className="text-sm text-text-muted">
              /{annual && plan.id !== "free" ? "mo, billed annually" : plan.period}
            </span>
          </div>
          {plan.id !== "free" && (
            <p className="text-sm text-text-secondary">
              {plan.description}
            </p>
          )}
        </div>

        <ul className="mt-6 flex-1 space-y-3">
          {plan.features.map((feature) => (
            <li
              key={feature}
              className="flex gap-2 text-sm text-text-secondary"
            >
              <span className="mt-0.5 h-2.5 w-2.5 rounded-full bg-[#00ff9d]/80" />
              {feature}
            </li>
          ))}
        </ul>

        <div className="mt-8">
          <button
            onClick={() => handleStart(plan.id)}
            disabled={loadingPlan === plan.id}
            className={`w-full rounded-lg px-4 py-2.5 text-sm font-semibold transition ${
              plan.highlighted
                ? "bg-[#00ff9d] text-black hover:bg-[#00e08a]"
                : "border border-border-strong text-text-primary hover:border-[#00ff9d] hover:text-[#00ff9d]"
            } disabled:opacity-60`}
          >
            {loadingPlan === plan.id ? "Redirecting…" : buttonLabel}
          </button>
        </div>
      </div>
    );
  }

  return (
    <main className="min-h-screen bg-background text-text-primary">
      <section className="relative overflow-hidden border-b border-border">
        <div className="absolute left-1/2 top-0 h-[520px] w-[920px] -translate-x-1/2 rounded-full bg-[radial-gradient(circle_at_top,rgba(0,255,157,0.12),transparent_68%)] opacity-70" />
        <div className="relative mx-auto max-w-5xl px-6 pb-20 pt-24">
          <div className="mx-auto max-w-2xl text-center">
            <p className="text-sm font-semibold uppercase tracking-[0.22em] text-[#00ff9d]">
              Pricing
            </p>
            <h1 className="mt-4 text-4xl font-semibold tracking-tight text-text-primary md:text-5xl">
              Transparent pricing for autonomous spend control
            </h1>
            <p className="mt-4 text-base text-text-secondary md:text-lg">
              Start free. Upgrade when you’re ready to move from data to
              action. No hidden fees, no installer lock-in.
            </p>
          </div>

          <div className="mx-auto mt-10 flex max-w-md items-center justify-center gap-3 rounded-full border border-border-strong bg-surface p-1">
            <button
              onClick={() => setAnnual(false)}
              className={`rounded-full px-4 py-2 text-sm font-medium transition ${
                !annual
                  ? "bg-text-primary text-black"
                  : "text-text-secondary hover:text-text-primary"
              }`}
            >
              Monthly
            </button>
            <button
              onClick={() => setAnnual(true)}
              className={`rounded-full px-4 py-2 text-sm font-medium transition ${
                annual
                  ? "bg-text-primary text-black"
                  : "text-text-secondary hover:text-text-primary"
              }`}
            >
              Annual
              <span className="ml-1 text-xs text-[#00ff9d]">
                save 20%
              </span>
            </button>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 pb-24 pt-14">
        <div className="grid gap-6 md:grid-cols-3">{plans.map(renderPlan)}</div>
      </section>

      <section className="mx-auto max-w-3xl px-6 pb-24">
        <h2 className="text-center text-2xl font-semibold text-text-primary">
          Frequently asked questions
        </h2>
        <div className="mt-8 space-y-4">
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
          ].map((faq) => (
            <div
              key={faq.q}
              className="rounded-2xl border border-border bg-surface px-6 py-5"
            >
              <h3 className="text-sm font-semibold text-text-primary">
                {faq.q}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-text-secondary">
                {faq.a}
              </p>
            </div>
          ))}
        </div>
      </section>

      <SiteFooter />
    </main>
  );
}

"use client";

import { useState } from "react";
import Link from "next/link";

type Plan = {
  name: string;
  price: string;
  priceNote: string;
  features: string[];
  cta: string;
  href: string;
  highlighted: boolean;
  badge?: string;
};

const plans: Plan[] = [
  {
    name: "Free",
    price: "$0",
    priceNote: "forever",
    features: [
      "10,000 events / month",
      "1 tenant",
      "Dashboard & analytics",
      "CSV export",
      "Email support",
    ],
    cta: "Get Started",
    href: "/signup",
    highlighted: false,
  },
  {
    name: "Pro",
    price: "$29",
    priceNote: "per month",
    features: [
      "100,000 events / month",
      "5 tenants",
      "Everything in Free",
      "Output analysis & Goblin Score",
      "Routing recommendations",
      "Priority support",
    ],
    cta: "Start Pro Trial",
    href: "/signup",
    highlighted: true,
    badge: "Most Popular",
  },
  {
    name: "Enterprise",
    price: "$99",
    priceNote: "per month",
    features: [
      "Unlimited events",
      "Unlimited tenants",
      "Everything in Pro",
      "Custom pricing overrides",
      "Audit trail & RBAC",
      "SLA & dedicated support",
    ],
    cta: "Contact Sales",
    href: "/signup",
    highlighted: false,
  },
];

export default function PricingPage() {
  const [annual, setAnnual] = useState(false);

  return (
    <main className="min-h-screen bg-[#f7f8f3] text-[#171915]">
      {/* Hero */}
      <section className="border-b border-[#d7dccf] bg-[#fbfcf8]">
        <div className="mx-auto w-full max-w-4xl px-5 py-16 text-center">
          <p className="text-sm font-semibold uppercase tracking-[0.16em] text-[#61705a]">
            Pricing
          </p>
          <h1 className="mt-3 text-4xl font-bold tracking-tight md:text-5xl">
            Simple, transparent pricing
          </h1>
          <p className="mt-4 text-lg text-[#52604e]">
            Start free. Scale when you need to. No hidden fees.
          </p>

          {/* Toggle */}
          <div className="mt-8 inline-flex items-center gap-3 rounded-full border border-[#d7dccf] bg-white p-1">
            <button
              onClick={() => setAnnual(false)}
              className={`rounded-full px-4 py-2 text-sm font-medium transition-colors ${!annual ? "bg-[#171915] text-white" : "text-[#52604e] hover:text-[#171915]"}`}
            >
              Monthly
            </button>
            <button
              onClick={() => setAnnual(true)}
              className={`rounded-full px-4 py-2 text-sm font-medium transition-colors ${annual ? "bg-[#171915] text-white" : "text-[#52604e] hover:text-[#171915]"}`}
            >
              Annual{" "}
              <span className="text-xs text-[#426b51]">
                {annual ? "" : "save 20%"}
              </span>
            </button>
          </div>
        </div>
      </section>

      {/* Plans */}
      <section className="mx-auto w-full max-w-6xl px-5 py-12">
        <div className="grid gap-6 md:grid-cols-3">
          {plans.map((plan) => {
            const displayPrice = annual
              ? `$${Math.round(parseInt(plan.price.replace("$", "")) * 0.8)}`
              : plan.price;

            return (
              <div
                key={plan.name}
                className={`relative flex flex-col rounded-2xl border p-6 ${plan.highlighted ? "border-[#426b51] bg-white shadow-lg shadow-[#426b51]/10" : "border-[#d7dccf] bg-white"}`}
              >
                {plan.badge && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-[#426b51] px-3 py-1 text-xs font-semibold text-white">
                    {plan.badge}
                  </div>
                )}

                <h2 className="text-xl font-bold">{plan.name}</h2>
                <div className="mt-4">
                  <span className="text-4xl font-bold">{displayPrice}</span>
                  <span className="ml-1 text-sm text-[#52604e]">
                    /{annual ? "mo (billed annually)" : plan.priceNote}
                  </span>
                </div>

                <ul className="mt-6 flex-1 space-y-3">
                  {plan.features.map((f) => (
                    <li key={f} className="flex items-start gap-2 text-sm text-[#52604e]">
                      <svg
                        className="mt-0.5 h-4 w-4 shrink-0 text-[#426b51]"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        strokeWidth={2}
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M5 13l4 4L19 7"
                        />
                      </svg>
                      {f}
                    </li>
                  ))}
                </ul>

                <Link
                  href={plan.href}
                  className={`mt-6 block rounded-lg py-3 text-center text-sm font-semibold transition-colors ${plan.highlighted ? "bg-[#426b51] text-white hover:bg-[#365a43]" : "border border-[#d7dccf] bg-white text-[#171915] hover:bg-[#f7f8f3]"}`}
                >
                  {plan.cta}
                </Link>
              </div>
            );
          })}
        </div>
      </section>

      {/* FAQ */}
      <section className="mx-auto w-full max-w-3xl px-5 py-12">
        <h2 className="mb-8 text-center text-2xl font-bold">
          Frequently asked questions
        </h2>
        <div className="space-y-6">
          {[
            {
              q: "What happens if I exceed my event limit?",
              a: "We'll notify you when you're approaching your limit. You can upgrade anytime or we'll gracefully degrade with degraded cost estimates.",
            },
            {
              q: "Can I cancel anytime?",
              a: "Yes. Cancel anytime and your data remains accessible in read-only mode for 30 days.",
            },
            {
              q: "Do you offer a free trial for Pro?",
              a: "The Free tier gives you full access to core features. Upgrade to Pro when you need more volume and advanced analytics.",
            },
            {
              q: "Is there an API?",
              a: "Yes. All plans include REST API access with API key authentication. Pro and Enterprise get higher rate limits.",
            },
          ].map((faq) => (
            <div
              key={faq.q}
              className="border-b border-[#e0e4d8] pb-6"
            >
              <h3 className="font-semibold">{faq.q}</h3>
              <p className="mt-2 text-sm leading-6 text-[#52604e]">{faq.a}</p>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}

"use client";

import { useEffect, useRef, useState } from "react";
import { SiteFooter } from "@/components/layout";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const brands = [
  {
    name: "TokenGoblin",
    description:
      "Operational intelligence and token-efficiency observability for autonomous AI workflows.",
    href: "/",
    github: "https://github.com/Hardonian/TokenGoblin",
  },
  {
    name: "Settler",
    description:
      "Reconciliation as a service for Stripe, banks, and invoices. Deterministic proofpacks and audit evidence.",
    href: "https://github.com/Hardonian/Settler",
    github: "https://github.com/Hardonian/Settler",
  },
  {
    name: "AIAS",
    description:
      "AI alignment substrate for safe, verifiable agent deployments and model policy enforcement.",
    href: "https://github.com/Hardonian/AIAS",
    github: "https://github.com/Hardonian/AIAS",
  },
];

export default function AboutPage() {
  const [now, setNow] = useState(new Date());
  const autoRedirectTimerRef = useRef<number | null>(null);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  return (
    <main className="min-h-screen bg-background text-text-primary">
      <section className="mx-auto max-w-6xl px-6 py-20">
        <div className="mx-auto max-w-2xl text-center">
          <span className="inline-flex rounded-full border border-[#00ff9d]/40 bg-accent-muted px-3 py-1 text-xs font-semibold text-[#00ff9d]">
            System context
          </span>
          <h1 className="mt-4 text-4xl font-semibold tracking-tight text-white md:text-5xl">
            Hardonian product system
          </h1>
          <p className="mt-4 text-base text-text-secondary">
            A coordinated stack across observability, reconciliation, and
            alignment. Each module is designed to reduce ambiguity, preserve
            evidence, and move operators from reaction to control.
          </p>
        </div>

        <div className="mx-auto mt-12 grid gap-4 md:grid-cols-3">
          {brands.map((brand) => (
            <a
              key={brand.name}
              href={brand.href}
              className="group rounded-2xl border border-border bg-surface p-5 transition hover:border-[#00ff9d]"
            >
              <div className="flex items-start justify-between">
                <div>
                  <h2 className="text-lg font-semibold text-white">{brand.name}</h2>
                  <p className="mt-2 text-sm text-text-secondary">{brand.description}</p>
                </div>
                <span className="text-xs text-text-muted">→</span>
              </div>
              <div className="mt-5 flex items-center gap-3">
                <span className="h-1.5 w-1.5 rounded-full bg-[#00ff9d]" />
                <span className="text-xs text-text-muted">{brand.github}</span>
              </div>
            </a>
          ))}
        </div>

        <div className="mx-auto mt-10 grid gap-4 md:grid-cols-3">
          {[
            {
              label: "Operator posture",
              value: "deterministic",
            },
            {
              label: "Evidence model",
              value: "hash-linked",
            },
            {
              label: "Local time",
              value: now.toUTCString(),
            },
          ].map((item) => (
            <div
              key={item.label}
              className="rounded-2xl border border-border bg-surface p-5"
            >
              <p className="text-xs text-text-muted">{item.label}</p>
              <p className="mt-1 text-sm text-white">{item.value}</p>
            </div>
          ))}
        </div>
      </section>
      <SiteFooter />
    </main>
  );
}

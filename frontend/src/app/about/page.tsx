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

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  return (
    <main className="min-h-screen bg-black text-zinc-300 font-mono selection:bg-[#ffb000] selection:text-black">
      <section className="mx-auto max-w-[1200px] px-6 py-24">
        
        <div className="mx-auto max-w-2xl text-center border-b border-[#333] pb-12">
          <span className="inline-block border border-[#ffb000] bg-[#ffb000]/10 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-[#ffb000] mb-6">
            [ System_Context ]
          </span>
          <h1 className="text-3xl font-bold tracking-widest text-white uppercase md:text-4xl">
            Hardonian Product System
          </h1>
          <p className="mt-6 text-xs text-zinc-500 font-mono uppercase tracking-widest leading-relaxed">
            {'>>'} A coordinated stack across observability, reconciliation, and alignment. 
            Each module is designed to reduce ambiguity, preserve evidence, and move 
            operators from reaction to control.
          </p>
        </div>

        <div className="mt-16 grid gap-6 md:grid-cols-3">
          {brands.map((brand) => (
            <a
              key={brand.name}
              href={brand.href}
              className="group border border-[#333] bg-black p-6 transition-colors hover:border-[#ffb000] flex flex-col justify-between"
            >
              <div>
                <div className="flex items-start justify-between border-b border-[#222] pb-4 mb-4">
                  <h2 className="text-sm font-bold text-white uppercase tracking-widest">
                    {brand.name}
                  </h2>
                  <span className="text-[#ffb000] opacity-0 group-hover:opacity-100 transition-opacity">
                    [↗]
                  </span>
                </div>
                <p className="text-xs text-zinc-400 font-mono leading-relaxed">
                  {brand.description}
                </p>
              </div>
              <div className="mt-8 flex items-center gap-3">
                <span className="text-[#ffb000] text-[10px]">{'>>'}{'>'}</span>
                <span className="text-[10px] text-zinc-600 uppercase tracking-widest">
                  {brand.github}
                </span>
              </div>
            </a>
          ))}
        </div>

        <div className="mt-12 grid gap-6 md:grid-cols-3">
          {[
            {
              label: "Operator posture",
              value: "DETERMINISTIC",
            },
            {
              label: "Evidence model",
              value: "HASH-LINKED",
            },
            {
              label: "Local time",
              value: now.toUTCString(),
            },
          ].map((item) => (
            <div
              key={item.label}
              className="border border-[#222] bg-[#0a0a0a] p-4 flex items-center justify-between"
            >
              <p className="text-[10px] text-zinc-600 uppercase tracking-widest">
                {item.label}
              </p>
              <p className="text-xs font-bold text-[#ffb000] uppercase tracking-widest">
                {item.value}
              </p>
            </div>
          ))}
        </div>
      </section>
      
      <SiteFooter />
    </main>
  );
}

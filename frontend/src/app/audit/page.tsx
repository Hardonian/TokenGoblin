"use client";

import { useState } from "react";
import Link from "next/link";
import { Header } from "@/components/Header";
import { SiteFooter } from "@/components/layout";

export default function AuditPage() {
  const [email, setEmail] = useState("");
  const [spend, setSpend] = useState("$10k - $25k / mo");
  const [providers, setProviders] = useState("OpenAI + Anthropic");
  const [notes, setNotes] = useState("");
  const [status, setStatus] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !email.includes("@")) {
      setErrorMessage("Please enter a valid work email.");
      setStatus("error");
      return;
    }

    setStatus("submitting");
    setErrorMessage("");

    try {
      const message = `[LLM Spend Audit Request] Spend Bracket: ${spend} | Providers: ${providers} | Notes: ${notes || "None"}`;
      const res = await fetch("/api/contact", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          type: "lead",
          email,
          message,
          company: "", // honeypot left blank intentionally
        }),
      });

      const data = await res.json();
      if (!res.ok || !data.ok) {
        throw new Error(data.error?.message || "Submission failed. Please try again.");
      }

      setStatus("success");
    } catch (err) {
      setStatus("error");
      setErrorMessage(err instanceof Error ? err.message : "Submission failed. Please try again.");
    }
  };

  return (
    <div className="min-h-screen bg-black text-zinc-300 font-mono selection:bg-[#ffb000] selection:text-black">
      <Header />

      <main className="max-w-[1200px] mx-auto px-6 py-16 space-y-24">
        {/* HERO */}
        <section className="text-center space-y-6 pt-8">
          <div className="inline-block border border-[#ffb000] bg-[#ffb000]/10 px-4 py-1 text-xs font-bold text-[#ffb000] uppercase tracking-[0.2em]">
            [ 72-Hour Executive LLM Spend Audit ]
          </div>
          <h1 className="text-3xl md:text-5xl font-black text-white uppercase tracking-tight max-w-4xl mx-auto leading-tight">
            Stop Bleeding Cash on Frontier Models. <br />
            <span className="text-[#10b981]">Reclaim 30% to 50% of Your LLM Spend.</span>
          </h1>
          <p className="text-sm md:text-base text-zinc-400 max-w-2xl mx-auto font-sans leading-relaxed">
            FinOps teams and engineering leaders use TokenGoblin&apos;s deterministic spend audit to identify
            prompt context padding, runaway output loops, cache misses, and multi-tier routing opportunities.
            Delivered in 72 hours with a 100% money-back guarantee.
          </p>

          <div className="pt-4 flex flex-wrap justify-center gap-4">
            <a
              href="#book-audit"
              className="bg-[#10b981] hover:bg-[#059669] text-black font-bold text-xs px-6 py-3 uppercase tracking-widest transition-all shadow-[0_0_20px_rgba(16,185,129,0.3)]"
            >
              [ Request 72h Audit ]
            </a>
            <a
              href="#packages"
              className="border border-[#333] hover:border-zinc-500 text-zinc-300 text-xs px-6 py-3 uppercase tracking-widest transition-all"
            >
              [ View Audit Packages ]
            </a>
          </div>
        </section>

        {/* METRICS BAR */}
        <section className="grid grid-cols-2 md:grid-cols-4 gap-4 border border-[#222] bg-[#080808] p-6 text-center">
          <div className="space-y-1">
            <div className="text-2xl md:text-3xl font-black text-[#10b981]">38.5%</div>
            <div className="text-[10px] text-zinc-500 uppercase tracking-widest">Avg Spend Reduction</div>
          </div>
          <div className="space-y-1">
            <div className="text-2xl md:text-3xl font-black text-[#ffb000]">72 Hours</div>
            <div className="text-[10px] text-zinc-500 uppercase tracking-widest">Fast Delivery SLA</div>
          </div>
          <div className="space-y-1">
            <div className="text-2xl md:text-3xl font-black text-white">100%</div>
            <div className="text-[10px] text-zinc-500 uppercase tracking-widest">Deterministic Proof</div>
          </div>
          <div className="space-y-1">
            <div className="text-2xl md:text-3xl font-black text-[#10b981]">3x ROI</div>
            <div className="text-[10px] text-zinc-500 uppercase tracking-widest">Or It&apos;s Free Guarantee</div>
          </div>
        </section>

        {/* WHAT YOU GET */}
        <section className="space-y-8">
          <div>
            <span className="text-xs text-[#ffb000] uppercase tracking-widest font-bold">
              [ Comprehensive Deliverables ]
            </span>
            <h2 className="text-2xl font-bold text-white uppercase tracking-wider mt-2">
              What the Spend Audit Delivers
            </h2>
          </div>

          <div className="grid md:grid-cols-2 gap-6">
            <div className="border border-[#222] bg-[#0a0a0a] p-6 space-y-3">
              <div className="text-xs font-bold text-[#10b981] uppercase tracking-wider">
                01 :: Executive Scorecard & Board Deck
              </div>
              <h3 className="text-lg font-bold text-white">CFO & VP Engineering Ready</h3>
              <p className="text-xs text-zinc-400 font-sans leading-relaxed">
                Clear executive summary quantifying total current spend, projected net savings, post-optimization
                monthly run rate, and annualized recovered budget with board-ready charts.
              </p>
            </div>

            <div className="border border-[#222] bg-[#0a0a0a] p-6 space-y-3">
              <div className="text-xs font-bold text-[#10b981] uppercase tracking-wider">
                02 :: Granular Burn Driver Breakdown
              </div>
              <h3 className="text-lg font-bold text-white">Model, Team & Feature Attribution</h3>
              <p className="text-xs text-zinc-400 font-sans leading-relaxed">
                Exhaustive tabular analysis mapping every dollar to models, engineering squads, and internal features.
                Pinpoint exactly which workflows are burning capital needlessly.
              </p>
            </div>

            <div className="border border-[#222] bg-[#0a0a0a] p-6 space-y-3">
              <div className="text-xs font-bold text-[#10b981] uppercase tracking-wider">
                03 :: 4 Technical Savings Playbooks
              </div>
              <h3 className="text-lg font-bold text-white">Actionable Engineering Blueprints</h3>
              <p className="text-xs text-zinc-400 font-sans leading-relaxed">
                Detailed playbooks covering 70/20/10 model routing, prompt & semantic prefix caching, targeted model
                right-sizing for structured tasks, and completion length guardrails.
              </p>
            </div>

            <div className="border border-[#222] bg-[#0a0a0a] p-6 space-y-3">
              <div className="text-xs font-bold text-[#10b981] uppercase tracking-wider">
                04 :: Gateway Proxy Configuration
              </div>
              <h3 className="text-lg font-bold text-white">Immediate Implementation Code</h3>
              <p className="text-xs text-zinc-400 font-sans leading-relaxed">
                Drop-in routing configs and prompt refiner middleware settings that your team can apply immediately
                without rewriting backend business logic.
              </p>
            </div>
          </div>
        </section>

        {/* PRICING PACKAGES */}
        <section id="packages" className="space-y-8">
          <div>
            <span className="text-xs text-[#ffb000] uppercase tracking-widest font-bold">
              [ Fixed-Fee Engagements ]
            </span>
            <h2 className="text-2xl font-bold text-white uppercase tracking-wider mt-2">
              Transparent, Guaranteed Audit Pricing
            </h2>
          </div>

          <div className="grid md:grid-cols-2 gap-8">
            {/* STANDARD AUDIT */}
            <div className="border border-[#333] bg-[#080808] p-8 space-y-6 flex flex-col justify-between">
              <div className="space-y-4">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="text-xl font-bold text-white uppercase tracking-wider">Standard Audit</h3>
                    <p className="text-xs text-zinc-500 uppercase">Up to $50,000 / mo LLM Spend</p>
                  </div>
                  <div className="text-right">
                    <div className="text-3xl font-black text-white">$1,500</div>
                    <div className="text-[10px] text-zinc-500 uppercase">Flat One-Time Fee</div>
                  </div>
                </div>

                <ul className="space-y-3 text-xs text-zinc-300 pt-4 border-t border-[#222]">
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> 72-Hour Guaranteed Delivery SLA
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Complete CSV / JSON Usage Export Ingestion
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Model, Team & Feature Breakdown Analysis
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> 4 Custom Savings Playbooks (Routing/Caching)
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Interactive HTML + Markdown + CSV Deliverables
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> 45-Min Executive & Technical Debrief Call
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#ffb000] font-bold">&gt;&gt;</span> 3x ROI Money-Back Guarantee
                  </li>
                </ul>
              </div>

              <a
                href="#book-audit"
                className="w-full text-center block bg-[#ffb000] hover:bg-[#ff8c00] text-black font-bold text-xs py-3 uppercase tracking-widest transition-all"
              >
                [ Select Standard Audit ]
              </a>
            </div>

            {/* ENTERPRISE AUDIT */}
            <div className="border border-[#10b981] bg-[#080808] p-8 space-y-6 flex flex-col justify-between relative shadow-[0_0_30px_rgba(16,185,129,0.1)]">
              <div className="absolute top-0 right-8 -translate-y-1/2 bg-[#10b981] text-black text-[10px] font-bold px-3 py-0.5 uppercase tracking-widest">
                Recommended for Scale
              </div>

              <div className="space-y-4">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="text-xl font-bold text-white uppercase tracking-wider">Enterprise Audit</h3>
                    <p className="text-xs text-zinc-500 uppercase">Unlimited Spend & Multi-Provider</p>
                  </div>
                  <div className="text-right">
                    <div className="text-3xl font-black text-[#10b981]">$3,000</div>
                    <div className="text-[10px] text-zinc-500 uppercase">Flat One-Time Fee</div>
                  </div>
                </div>

                <ul className="space-y-3 text-xs text-zinc-300 pt-4 border-t border-[#222]">
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Everything in Standard Audit
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Multi-Provider Ingestion (OpenAI, Anthropic, Google, xAI)
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Custom Eval & Model Downgrade Benchmarks
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Gateway Proxy Middleware Rule Generation
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> 90-Min Architecture & Engineering Deep-Dive
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#10b981] font-bold">&gt;&gt;</span> Dedicated Slack Channel for 14-Day Implementation
                  </li>
                  <li className="flex items-center gap-2">
                    <span className="text-[#ffb000] font-bold">&gt;&gt;</span> 3x ROI Money-Back Guarantee
                  </li>
                </ul>
              </div>

              <a
                href="#book-audit"
                className="w-full text-center block bg-[#10b981] hover:bg-[#059669] text-black font-bold text-xs py-3 uppercase tracking-widest transition-all"
              >
                [ Select Enterprise Audit ]
              </a>
            </div>
          </div>
        </section>

        {/* GUARANTEE BOX */}
        <section className="border border-[#10b981]/50 bg-[#0a1a12] p-8 md:p-12 relative">
          <div className="max-w-2xl space-y-4">
            <div className="text-xs text-[#10b981] font-bold uppercase tracking-widest">
              [ 100% Risk-Free Guarantee ]
            </div>
            <h3 className="text-2xl font-bold text-white uppercase tracking-wider">
              If We Don&apos;t Uncover At Least 3x Our Fee, It&apos;s Free.
            </h3>
            <p className="text-xs md:text-sm text-zinc-300 font-sans leading-relaxed">
              We stand behind our deterministic methodology. If our spend audit fails to identify actionable annual
              savings equal to at least three times the audit fee ($4,500 for Standard, $9,000 for Enterprise), you
              receive a full, immediate 100% refund. No disputes, no fine print.
            </p>
          </div>
        </section>

        {/* INTAKE / BOOKING FORM */}
        <section id="book-audit" className="border border-[#333] bg-[#080808] p-8 md:p-12 space-y-8">
          <div>
            <span className="text-xs text-[#ffb000] uppercase tracking-widest font-bold">
              [ Fast-Track Intake ]
            </span>
            <h2 className="text-2xl font-bold text-white uppercase tracking-wider mt-2">
              Book Your LLM Spend Audit
            </h2>
            <p className="text-xs text-zinc-400 font-sans mt-2">
              Submit your details below. We will send you secure upload instructions for your redacted usage exports
              and schedule your 72-hour delivery window immediately.
            </p>
          </div>

          {status === "success" ? (
            <div className="border border-[#10b981] bg-[#0a1a12] p-6 text-center space-y-4">
              <div className="text-xl font-bold text-[#10b981] uppercase tracking-wider">
                [✓] Audit Request Received
              </div>
              <p className="text-xs text-zinc-300 font-sans max-w-lg mx-auto">
                Thank you! Our FinOps team has queued your intake. We have dispatched secure upload instructions to{" "}
                <span className="text-white font-bold">{email}</span>. Your 72-hour delivery SLA begins upon receipt
                of your usage export.
              </p>
              <div className="pt-2">
                <Link
                  href="/"
                  className="inline-block border border-[#10b981] text-[#10b981] px-4 py-2 text-xs font-bold uppercase tracking-widest hover:bg-[#10b981] hover:text-black transition-colors"
                >
                  [ Return to Command Center ]
                </Link>
              </div>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-6 max-w-xl">
              <div className="space-y-2">
                <label className="text-xs font-bold uppercase tracking-wider text-zinc-300">
                  Work Email <span className="text-red-500">*</span>
                </label>
                <input
                  type="email"
                  required
                  placeholder="name@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full bg-black border border-[#333] px-4 py-2.5 text-xs text-white placeholder-zinc-600 focus:outline-none focus:border-[#ffb000]"
                />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-300">
                    Monthly LLM Spend Bracket
                  </label>
                  <select
                    value={spend}
                    onChange={(e) => setSpend(e.target.value)}
                    className="w-full bg-black border border-[#333] px-3 py-2.5 text-xs text-white focus:outline-none focus:border-[#ffb000]"
                  >
                    <option value="Under $10k / mo">Under $10k / mo</option>
                    <option value="$10k - $25k / mo">$10k - $25k / mo</option>
                    <option value="$25k - $50k / mo">$25k - $50k / mo</option>
                    <option value="$50k - $100k / mo">$50k - $100k / mo</option>
                    <option value="$100k+ / mo">$100k+ / mo</option>
                  </select>
                </div>

                <div className="space-y-2">
                  <label className="text-xs font-bold uppercase tracking-wider text-zinc-300">
                    Primary Model Providers
                  </label>
                  <select
                    value={providers}
                    onChange={(e) => setProviders(e.target.value)}
                    className="w-full bg-black border border-[#333] px-3 py-2.5 text-xs text-white focus:outline-none focus:border-[#ffb000]"
                  >
                    <option value="OpenAI Only">OpenAI Only</option>
                    <option value="Anthropic Only">Anthropic Only</option>
                    <option value="OpenAI + Anthropic">OpenAI + Anthropic</option>
                    <option value="Multi-Cloud (OpenAI/Claude/Gemini)">Multi-Cloud (OpenAI/Claude/Gemini)</option>
                    <option value="Open Source / Self-Hosted + Cloud">Open Source / Self-Hosted + Cloud</option>
                  </select>
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-bold uppercase tracking-wider text-zinc-300">
                  Notes or Specific Burning Concerns (Optional)
                </label>
                <textarea
                  rows={3}
                  placeholder="e.g. High o1 reasoning spend, runaway agents in test environment, etc."
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  className="w-full bg-black border border-[#333] px-4 py-2.5 text-xs text-white placeholder-zinc-600 focus:outline-none focus:border-[#ffb000]"
                />
              </div>

              {status === "error" && (
                <div className="border border-red-900 bg-red-950/20 p-3 text-xs text-red-400">
                  [ERR] {errorMessage}
                </div>
              )}

              <button
                type="submit"
                disabled={status === "submitting"}
                className="w-full bg-[#10b981] hover:bg-[#059669] text-black font-bold text-xs py-3 uppercase tracking-widest transition-all disabled:opacity-50"
              >
                {status === "submitting" ? "[ Queuing Audit... ]" : "[ Submit Audit Request — $1,500 / $3,000 ]"}
              </button>
            </form>
          )}
        </section>
      </main>

      <SiteFooter />
    </div>
  );
}

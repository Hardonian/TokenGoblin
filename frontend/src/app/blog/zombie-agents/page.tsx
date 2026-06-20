import { Metadata } from "next";
import { motion } from "framer-motion";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Zombie Agent Detection: How We Saved $12K/Month with Automated Quarantine",
  description: "Deep dive into acceptance-rate-based anomaly detection for AI agents. How TokenGoblin identifies and automatically quarantines zombie agents consuming budget with near-zero value.",
  keywords: [
    "zombie agent detection",
    "AI agent anomaly detection",
    "automated agent quarantine",
    "LLM cost optimization",
    "acceptance rate monitoring",
  ],
  authors: [{ name: "TokenGoblin Team" }],
  openGraph: {
    title: "Zombie Agent Detection: How We Saved $12K/Month",
    description: "Deep dive into acceptance-rate-based anomaly detection for autonomous AI agents.",
    type: "article",
  },
};

export default function BlogPost() {
  return (
    <article className="min-h-screen bg-black text-zinc-300 font-mono selection:bg-[#ffb000] selection:text-black py-20 px-6">
      <div className="max-w-4xl mx-auto space-y-12">
        <nav className="flex items-center gap-2 text-xs text-zinc-500 uppercase tracking-widest">
          <Link href="/" className="hover:text-[#ffb000] transition-colors">HOME</Link>
          <span>/</span>
          <Link href="/blog" className="hover:text-[#ffb000] transition-colors">BLOG</Link>
          <span>/</span>
          <span className="text-zinc-400">ZOMBIE AGENT DETECTION</span>
        </nav>

        <header className="space-y-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-center"
          >
            <span className="inline-block px-3 py-1 text-[10px] font-bold uppercase tracking-widest bg-[#ef4444]20 text-[#ef4444] rounded">
              CASE STUDY
            </span>
            <h1 className="mt-6 text-4xl md:text-6xl font-bold tracking-tight text-white leading-tight">
              Zombie Agent Detection:
              <br />
              <span className="text-[#ef4444]">How We Saved $12K/Month</span>
            </h1>
            <p className="mt-6 text-lg md:text-xl text-zinc-400 max-w-2xl mx-auto leading-relaxed">
              How acceptance-rate-based anomaly detection automatically identifies and quarantines 
              AI agents consuming budget with near-zero value delivery.
            </p>
          </motion.div>
        </header>

        {/* The Problem */}
        <section className="space-y-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="prose prose-invert max-w-none"
          >
            <h2 className="text-3xl md:text-4xl font-bold tracking-tight text-white uppercase tracking-widest mb-6">THE PROBLEM: INVISIBLE WASTE</h2>
            
            <div className="bg-[#111] border border-[#333] rounded-xl p-6">
              <p className="text-zinc-400 leading-relaxed mb-6">
                A fintech company deployed 50 autonomous agents for code review, testing, and documentation. 
                Monthly LLM spend: $8,400. Everything looked normal in aggregate dashboards.
              </p>
              <p className="text-zinc-400 leading-relaxed mb-6">
                Deep analysis revealed 3 agents (6% of fleet) were responsible for $4,200/month in spend 
                with less than 8% acceptance rates. These &apos;zombie agents&apos; were:
              </p>
              <ul className="space-y-3 text-zinc-400">
                <li className="flex items-center gap-3">
                  <span className="w-2 h-2 rounded-full bg-[#ef4444]"></span>
                  <span>Code review agent stuck in retry loops on malformed PRs</span>
                </li>
                <li className="flex items-center gap-3">
                  <span className="w-2 h-2 rounded-full bg-[#ef4444]"></span>
                  <span>Test generator producing syntactically invalid tests (rejected by CI)</span>
                </li>
                <li className="flex items-center gap-3">
                  <span className="w-2 h-2 rounded-full bg-[#ef4444]"></span>
                  <span>Doc generator hallucinating API endpoints (rejected by human reviewers)</span>
                </li>
              </ul>
            </div>

            <div className="bg-[#111] border border-[#ef4444]30 rounded-xl p-6 mb-8">
              <h3 className="text-[#ef4444] font-bold text-xl mb-3">THE COST</h3>
              <div className="grid gap-4 md:grid-cols-4">
                <div className="text-center p-4 bg-[#111] rounded-lg">
                  <p className="text-3xl font-bold text-[#ef4444] font-mono">$4,200</p>
                  <p className="text-zinc-500 text-sm">Monthly waste</p>
                </div>
                <div className="text-center p-4 bg-[#111] rounded-lg">
                  <p className="text-3xl font-bold text-[#ef4444] font-mono">50%</p>
                  <p className="text-zinc-500 text-sm">Of total agent spend</p>
                </div>
                <div className="text-center p-4 bg-[#111] rounded-lg">
                  <p className="text-3xl font-bold text-[#ef4444] font-mono">3</p>
                  <p className="text-zinc-500 text-sm">Zombie agents</p>
                </div>
                <div className="text-center p-4 bg-[#111] rounded-lg">
                  <p className="text-3xl font-bold text-[#ef4444] font-mono">6%</p>
                  <p className="text-zinc-500 text-sm">Of agent fleet</p>
                </div>
              </div>
            </div>
          </motion.div>
        </section>

        {/* The Solution */}
        <section className="space-y-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="prose prose-invert max-w-none"
          >
            <h2 className="text-3xl md:text-4xl font-bold tracking-tight text-white uppercase tracking-widest mb-6">THE SOLUTION: ACCEPTANCE-RATE THRESHOLDS</h2>
            
            <div className="bg-[#111] border border-[#333] rounded-xl p-6 mb-8">
              <h3 className="text-white font-bold text-xl mb-4">HOW ZOMBIE DETECTION WORKS</h3>
              <div className="space-y-4">
                {[
                  { step: "1", title: "TRACK ACCEPTANCE RATES", desc: "Every agent invocation returns an acceptance signal (accepted/rejected/flagged). Rolling 24h window." },
                  { step: "2", title: "CALCULATE RATES", desc: "Acceptance rate = accepted / total. Rolling window prevents burst noise." },
                  { step: "3", title: "THRESHOLD CHECK", desc: "Default: <20% over 24h = zombie candidate. Configurable per agent type." },
                  { step: "4", title: "QUARANTINE", desc: "API endpoint to pause agent. Slack alert with one-click quarantine." },
                  { step: "5", title: "REMEDIATION", desc: "Root cause analysis prompt. Auto-fix or human review workflow." },
                ].map((item, i) => (
                  <div key={i} className="flex gap-4 p-4 bg-[#0a0a0a] border border-[#222] rounded-lg">
                    <div className="w-10 h-10 rounded-xl bg-[#ef4444]20 flex items-center justify-center flex-shrink-0">
                      <span className="text-[#ef4444] font-bold text-xl">{item.step}</span>
                    </div>
                    <div>
                      <h4 className="text-white font-bold text-lg">{item.title}</h4>
                      <p className="text-zinc-400 text-sm mt-1">{item.desc}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-[#111] border border-[#333] rounded-xl p-6 mb-8">
              <h3 className="text-white font-bold text-xl mb-4">IMPLEMENTATION CODE</h3>
              <pre className="bg-[#0a0a0a] border border-[#222] rounded-lg p-4 overflow-x-auto text-sm text-zinc-300"><code>{`// Configure zombie detection per agent
await tokengoblin.zombies.configure({
  agentId: "code-review-agent",
  thresholds: {
    acceptanceRate: 0.20,    // 20% floor
    windowHours: 24,        // rolling window
    minInvocations: 50,     // minimum sample size
  },
  actions: {
    onZombie: "quarantine",      // or "alert" | "circuit-breaker"
    notify: ["slack:#ai-ops", "pagerduty"],
    autoRemediate: false,        // requires human approval
  },
});

// Check zombie status
const status = await tokengoblin.zombies.check("code-review-agent");
// { isZombie: true, acceptanceRate: 0.08, recommendation: "quarantine" }

// Quarantine with one call
await tokengoblin.zombies.quarantine("code-review-agent", "Acceptance rate 8% < 20% threshold");`}</code></pre>
            </div>
          </motion.div>
        </section>

        {/* Results */}
        <section className="space-y-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="prose prose-invert max-w-none"
          >
            <h2 className="text-3xl md:text-4xl font-bold tracking-tight text-white uppercase tracking-widest mb-6">RESULTS: $12K/MONTH SAVED</h2>
            
            <div className="grid gap-6 md:grid-cols-4 mb-8">
              <div className="bg-[#111] border border-[#22c55e]30 rounded-xl p-6 text-center">
                <p className="text-4xl font-bold text-[#22c55e] font-mono">$12,000</p>
                <p className="text-zinc-500 text-sm">Monthly savings</p>
              </div>
              <div className="text-center p-6 bg-[#111] rounded-xl">
                <p className="text-4xl font-bold text-[#22c55e] font-mono">85%</p>
                <p className="text-zinc-500 text-sm">Waste reduction</p>
              </div>
              <div className="text-center p-6 bg-[#111] rounded-xl">
                <p className="text-4xl font-bold text-[#22c55e] font-mono">47 min</p>
                <p className="text-zinc-500 text-sm">Avg detection time</p>
              </div>
              <div className="text-center p-6 bg-[#111] rounded-xl">
                <p className="text-4xl font-bold text-[#22c55e] font-mono">0</p>
                <p className="text-zinc-500 text-sm">False positives</p>
              </div>
            </div>

            <div className="bg-[#111] border border-[#333] rounded-xl p-6">
              <h3 className="text-white font-bold text-xl mb-4">REMEDIATION WORKFLOWS THAT WORKED</h3>
              <div className="space-y-4">
                <div className="bg-[#0a0a0a] border border-[#222] rounded-lg p-4">
                  <h4 className="text-[#22c55e] font-bold mb-2">RETRY LOGIC FIX</h4>
                  <p className="text-zinc-400 text-sm">Code review agent retry loops fixed with exponential backoff + max retry cap. Savings: $1,800/mo.</p>
                </div>
                <div className="bg-[#0a0a0a] border border-[#222] rounded-lg p-4">
                  <h4 className="text-[#22c55e] font-bold mb-2">PROMPT TEMPLATE FIX</h4>
                  <p className="text-zinc-400 text-sm">Test generator prompt fixed: added explicit syntax validation examples. Savings: $2,100/mo.</p>
                </div>
                <div className="bg-[#0a0a0a] border border-[#222] rounded-lg p-4">
                  <h4 className="text-[#22c55e] font-bold mb-2">CIRCUIT BREAKER</h4>
                  <p className="text-zinc-400 text-sm">Circuit breaker at 10% acceptance rate prevents future zombie outbreaks. Zero false positives in 90 days.</p>
                </div>
              </div>
            </div>
          </motion.div>
        </section>

        {/* CTA */}
        <section className="pt-12">
          <div className="bg-gradient-to-r from-[#ef4444]20 to-[#f97316]20 border border-[#ef4444]30 rounded-2xl p-12 text-center">
            <h2 className="text-3xl md:text-4xl font-bold tracking-tight text-white mb-4">
              DETECT ZOMBIE AGENTS IN YOUR FLEET
            </h2>
            <p className="text-zinc-400 text-lg mb-8 max-w-2xl mx-auto">
              Free tier includes zombie detection for up to 10 agents. Deploy in 15 minutes.
            </p>
            <div className="flex items-center justify-center gap-4">
              <Link href="/signup" className="bg-[#ef4444] text-white px-8 py-4 rounded font-bold uppercase tracking-widest hover:bg-[#dc2626] transition-colors text-lg">
                START FREE
              </Link>
              <a href="https://github.com/Hardonian/TokenGoblin" className="border border-[#333] text-zinc-300 px-8 py-4 rounded font-bold uppercase tracking-widest hover:border-[#ef4444] hover:text-[#ef4444] transition-colors">
                VIEW SOURCE
              </a>
            </div>
          </div>
        </section>
      </div>
    </article>
  );
}
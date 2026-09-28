"use client";

import { useState } from "react";
import { motion } from "framer-motion";

export function LeadCaptureWidget() {
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await fetch("/api/contact", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ type: "lead", email }),
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        throw new Error(payload?.error?.message || "Unable to submit your request.");
      }
      setSubmitted(true);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "Unable to submit your request.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="w-full bg-[#0a0a0a] border-t border-[#333] py-12 px-6 mt-16">
      <div className="max-w-[1400px] mx-auto flex flex-col md:flex-row items-center justify-between gap-8">
        <div className="flex-1">
          <h2 className="text-[#ffb000] font-black text-xl mb-2 tracking-widest uppercase">Get Your Free Token ROI Audit</h2>
          <p className="text-zinc-400 text-sm max-w-lg leading-relaxed">
            Not sure if your autonomous agents are burning cash? Drop your email and our Goblin experts will analyze your architecture for free. Stop the cost leaks today.
          </p>
        </div>
        <div className="flex-1 w-full max-w-md">
          {submitted ? (
            <motion.div 
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              className="bg-[#111] border border-[#10b981] p-4 text-center rounded text-[#10b981]"
            >
              <p className="font-bold tracking-widest uppercase text-sm">Success! A Goblin will contact you shortly.</p>
            </motion.div>
          ) : (
            <form onSubmit={handleSubmit} className="w-full">
              <div className="flex gap-2">
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="cto@startup.com"
                className="flex-1 h-12 bg-black border border-[#333] px-4 text-zinc-200 outline-none focus:border-[#ffb000] text-sm"
              />
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                type="submit"
                disabled={loading}
                className="h-12 px-6 bg-[#ffb000] hover:bg-[#ff8c00] text-black font-bold uppercase tracking-widest text-xs transition-colors disabled:opacity-50"
              >
                {loading ? "Sending…" : "Claim Audit"}
              </motion.button>
              </div>
              {error ? <p className="mt-2 text-xs text-red-400" role="alert">{error}</p> : null}
            </form>
          )}
        </div>
      </div>
    </div>
  );
}

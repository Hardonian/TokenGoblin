"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { GoblinMascot } from "./GoblinMascot";

export function SupportChat() {
  const [isOpen, setIsOpen] = useState(false);
  const [message, setMessage] = useState("");
  const [status, setStatus] = useState<"idle" | "sending" | "sent" | "error">("idle");

  async function submitMessage(event: React.FormEvent) {
    event.preventDefault();
    setStatus("sending");
    try {
      const response = await fetch("/api/contact", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ type: "support", message }),
      });
      if (!response.ok) throw new Error("delivery failed");
      setMessage("");
      setStatus("sent");
    } catch {
      setStatus("error");
    }
  }

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col items-end gap-4">
      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ opacity: 0, y: 20, scale: 0.9 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 20, scale: 0.9 }}
            transition={{ type: "spring", stiffness: 400, damping: 25 }}
            className="w-80 bg-[#111] border border-[#333] shadow-2xl overflow-hidden flex flex-col rounded-lg"
          >
            <div className="bg-[#1a1a1a] p-4 flex justify-between items-center border-b border-[#333]">
              <div className="flex items-center gap-2">
                <GoblinMascot size={24} />
                <h3 className="text-zinc-200 font-bold text-sm tracking-widest uppercase">Goblin Support</h3>
              </div>
              <button aria-label="Close Support Chat" onClick={() => setIsOpen(false)} className="text-zinc-500 hover:text-white">✕</button>
            </div>
            <div className="p-4 flex-1 h-64 overflow-y-auto bg-black text-sm text-zinc-400 font-mono flex flex-col gap-3">
              <div className="bg-[#1a1a1a] p-3 rounded-r-lg rounded-bl-lg border border-[#333] w-10/12 self-start">
                <p>Hehehe... Need help uncovering cost leaks, or just wanna hoard tokens?</p>
              </div>
              <div className="bg-[#1a1a1a] p-3 rounded-r-lg rounded-bl-lg border border-[#333] w-10/12 self-start">
                <p>Send a message below and the support team will follow up through the configured support channel.</p>
              </div>
              {status === "sent" ? <p className="text-xs text-green-400" role="status">Message delivered.</p> : null}
              {status === "error" ? <p className="text-xs text-red-400" role="alert">Delivery failed. Please try again.</p> : null}
            </div>
            <div className="p-3 bg-[#111] border-t border-[#333]">
              <form onSubmit={submitMessage} className="flex gap-2">
                <input
                  type="text"
                  value={message}
                  onChange={(event) => setMessage(event.target.value)}
                  minLength={3}
                  maxLength={2000}
                  required
                  placeholder="Ask the goblin..."
                  aria-label="Support message"
                  className="min-w-0 flex-1 bg-black border border-[#333] p-2 text-xs text-white outline-none focus:border-[#ffb000]"
                />
                <button type="submit" disabled={status === "sending"} className="border border-[#ffb000] px-3 text-xs text-[#ffb000] disabled:opacity-50">
                  Send
                </button>
              </form>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <motion.button
        whileHover={{ scale: 1.1, rotate: [0, -10, 10, -5, 5, 0] }}
        whileTap={{ scale: 0.9 }}
        transition={{ type: "spring", stiffness: 400, damping: 10 }}
        onClick={() => setIsOpen(!isOpen)}
        aria-label="Toggle Support Chat"
        className="w-14 h-14 bg-var(--color-accent-goblin) border-2 border-var(--color-accent-goblin-strong) rounded-full flex items-center justify-center shadow-[0_0_15px_rgba(16,185,129,0.3)] hover:shadow-[0_0_25px_rgba(16,185,129,0.5)] cursor-pointer"
      >
        {isOpen ? (
          <span className="text-white font-bold text-xl">✕</span>
        ) : (
          <span className="text-3xl">💬</span>
        )}
      </motion.button>
    </div>
  );
}

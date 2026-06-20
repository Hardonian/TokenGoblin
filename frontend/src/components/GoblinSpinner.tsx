"use client";

import { motion } from "framer-motion";
import { GoblinMascot } from "./GoblinMascot";
import { TokenCoin } from "./TokenCoin";

export function GoblinSpinner() {
  return (
    <div className="flex flex-col items-center justify-center p-8 gap-4">
      <div className="flex items-center gap-8 relative">
        <motion.div
          animate={{ x: [0, 40, 0] }}
          transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
        >
          <GoblinMascot size={48} />
        </motion.div>
        
        <motion.div
          animate={{ x: [0, -40, 0], opacity: [1, 0, 1] }}
          transition={{ duration: 1.5, repeat: Infinity, ease: "easeInOut" }}
        >
          <TokenCoin size={24} />
        </motion.div>
      </div>
      <p className="text-[#10b981] font-mono text-xs uppercase tracking-widest animate-pulse">
        Goblin Hoarding Tokens...
      </p>
    </div>
  );
}

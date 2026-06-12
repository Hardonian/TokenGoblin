"use client";

import { motion } from "framer-motion";

export function TokenCoin({ className = "", size = 24, delay = 0 }: { className?: string; size?: number; delay?: number }) {
  return (
    <motion.div
      className={`inline-block ${className}`}
      animate={{ y: [0, -5, 0], rotateY: [0, 180, 360] }}
      transition={{
        duration: 3,
        repeat: Infinity,
        ease: "easeInOut",
        delay,
      }}
      style={{ perspective: 1000 }}
    >
      <svg
        width={size}
        height={size}
        viewBox="0 0 100 100"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        {/* Outer Ring */}
        <circle cx="50" cy="50" r="45" fill="var(--color-accent)" stroke="var(--color-accent-strong)" strokeWidth="5" />
        {/* Inner Engraving */}
        <circle cx="50" cy="50" r="30" fill="transparent" stroke="var(--color-accent-strong)" strokeWidth="2" strokeDasharray="5 5" />
        {/* Center Symbol (T for Token) */}
        <path d="M 35 35 L 65 35 M 50 35 L 50 70" stroke="var(--color-surface-strong)" strokeWidth="8" strokeLinecap="round" />
      </svg>
    </motion.div>
  );
}

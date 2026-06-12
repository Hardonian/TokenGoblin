"use client";

import { motion } from "framer-motion";
import { useState } from "react";

export function GoblinMascot({ className = "", size = 48 }: { className?: string; size?: number }) {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <motion.div
      className={`relative inline-flex items-center justify-center cursor-pointer ${className}`}
      onHoverStart={() => setIsHovered(true)}
      onHoverEnd={() => setIsHovered(false)}
      whileHover={{ scale: 1.1, rotate: [0, -10, 10, -5, 5, 0] }}
      whileTap={{ scale: 0.9 }}
      transition={{ type: "spring", stiffness: 400, damping: 10 }}
    >
      <svg
        width={size}
        height={size}
        viewBox="0 0 100 100"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="overflow-visible"
        role="img"
        aria-label="Token Goblin Mascot"
      >
        {/* Goblin Ears */}
        <motion.path
          d="M 20 40 Q 5 20 10 10 Q 25 15 30 30"
          fill="var(--color-accent-goblin)"
          stroke="var(--color-accent-goblin-strong)"
          strokeWidth="3"
          animate={{ rotate: isHovered ? [0, -15, 0] : 0, originX: "30px", originY: "30px" }}
          transition={{ repeat: isHovered ? Infinity : 0, duration: 0.5 }}
        />
        <motion.path
          d="M 80 40 Q 95 20 90 10 Q 75 15 70 30"
          fill="var(--color-accent-goblin)"
          stroke="var(--color-accent-goblin-strong)"
          strokeWidth="3"
          animate={{ rotate: isHovered ? [0, 15, 0] : 0, originX: "70px", originY: "30px" }}
          transition={{ repeat: isHovered ? Infinity : 0, duration: 0.5 }}
        />

        {/* Goblin Head */}
        <rect
          x="25"
          y="25"
          width="50"
          height="50"
          rx="15"
          fill="var(--color-accent-goblin)"
          stroke="var(--color-accent-goblin-strong)"
          strokeWidth="4"
        />

        {/* Goblin Eyes (Left) */}
        <rect x="35" y="40" width="8" height="8" rx="2" fill="var(--color-surface-strong)" />
        <motion.rect
          x="37"
          y="42"
          width="4"
          height="4"
          rx="1"
          fill="var(--color-accent)"
          animate={{ x: isHovered ? [0, 2, -2, 0] : 0, y: isHovered ? [0, -2, 2, 0] : 0 }}
          transition={{ repeat: Infinity, duration: 1.5, ease: "easeInOut" }}
        />

        {/* Goblin Eyes (Right) */}
        <rect x="57" y="40" width="8" height="8" rx="2" fill="var(--color-surface-strong)" />
        <motion.rect
          x="59"
          y="42"
          width="4"
          height="4"
          rx="1"
          fill="var(--color-accent)"
          animate={{ x: isHovered ? [0, 2, -2, 0] : 0, y: isHovered ? [0, -2, 2, 0] : 0 }}
          transition={{ repeat: Infinity, duration: 1.5, ease: "easeInOut", delay: 0.1 }}
        />

        {/* Goblin Mouth */}
        <motion.path
          d="M 35 60 Q 50 65 65 60"
          fill="none"
          stroke="var(--color-surface-strong)"
          strokeWidth="4"
          strokeLinecap="round"
          animate={{
            d: isHovered
              ? "M 35 60 Q 50 75 65 60"
              : "M 35 60 Q 50 65 65 60",
          }}
          transition={{ type: "spring", stiffness: 300, damping: 15 }}
        />

        {/* Goblin Tooth */}
        <motion.polygon
          points="40,62 45,62 42.5,68"
          fill="var(--color-text-primary)"
          animate={{ opacity: isHovered ? 1 : 0, y: isHovered ? 0 : -5 }}
        />
        <motion.polygon
          points="55,62 60,62 57.5,68"
          fill="var(--color-text-primary)"
          animate={{ opacity: isHovered ? 1 : 0, y: isHovered ? 0 : -5 }}
        />

        {/* Fun floating token when hovered */}
        <motion.circle
          cx="50"
          cy="75"
          r="8"
          fill="var(--color-accent)"
          stroke="#ff8c00"
          strokeWidth="2"
          initial={{ opacity: 0, scale: 0, y: 10 }}
          animate={{
            opacity: isHovered ? [0, 1, 0] : 0,
            scale: isHovered ? [0, 1, 0.5] : 0,
            y: isHovered ? [10, -10, -20] : 10,
          }}
          transition={{
            duration: 0.6,
            ease: "easeOut",
            repeat: isHovered ? Infinity : 0,
            repeatDelay: 0.2,
          }}
        />
      </svg>
    </motion.div>
  );
}

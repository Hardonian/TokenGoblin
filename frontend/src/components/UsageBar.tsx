"use client";
import { motion } from "framer-motion";
import { TokenCoin } from "./TokenCoin";
export default function UsageBar({
  percent,
  label,
}: {
  percent: number;
  label?: string;
}) {
  const clamped = Math.min(100, Math.max(0, percent));
  const color =
    clamped >= 90
      ? "bg-red-500"
      : clamped >= 70
        ? "bg-[#ffb000]"
        : "bg-[#10b981]";

  return (
    <div className="w-full">
      {label && (
        <div className="mb-2 flex items-center justify-between text-xs font-bold text-zinc-300">
          <span className="flex items-center gap-2">
            <TokenCoin size={14} delay={Math.random()} />
            {label}
          </span>
          <span>{clamped.toFixed(1)}%</span>
        </div>
      )}
      <div className="h-4 w-full overflow-hidden rounded-full bg-zinc-800 border border-zinc-700">
        <motion.div
          className={`h-full rounded-full ${color}`}
          initial={{ width: 0 }}
          animate={{ width: `${clamped}%` }}
          transition={{ type: "spring", stiffness: 50, damping: 10 }}
        />
      </div>
    </div>
  );
}

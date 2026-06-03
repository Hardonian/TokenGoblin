"use client";

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
        ? "bg-amber-500"
        : "bg-[#426b51]";

  return (
    <div className="w-full">
      {label && (
        <div className="mb-1 flex justify-between text-xs text-[#52604e]">
          <span>{label}</span>
          <span>{clamped.toFixed(1)}%</span>
        </div>
      )}
      <div className="h-3 w-full overflow-hidden rounded-full bg-[#e0e4d8]">
        <div
          className={`h-full rounded-full transition-all duration-500 ${color}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}

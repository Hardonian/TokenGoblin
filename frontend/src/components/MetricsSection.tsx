import { Metric } from "@/components/shared";

interface MetricsSectionProps {
  totalTokens: number;
  totalCostUsd: number;
  outputCount: number;
  avgOutputSize: number;
}

export function MetricsSection({
  totalTokens,
  totalCostUsd,
  outputCount,
  avgOutputSize,
}: MetricsSectionProps) {
  return (
    <section className="mx-auto grid w-full max-w-7xl gap-4 px-5 py-5 md:grid-cols-4">
      <Metric label="Total tokens" value={formatInt(totalTokens)} />
      <Metric label="Estimated cost" value={`$${formatMoney(totalCostUsd)}`} />
      <Metric label="Output count" value={formatInt(outputCount)} />
      <Metric label="Avg output size" value={`${formatInt(avgOutputSize)} tokens`} />
    </section>
  );
}
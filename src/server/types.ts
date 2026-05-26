export const OUTPUT_STATUSES = [
  "accepted",
  "succeeded",
  "failed",
  "rejected",
  "pending"
] as const;

export type OutputStatus = (typeof OUTPUT_STATUSES)[number];

export type ApiStatus = "success" | "degraded" | "error";

export interface ApiIssue {
  code: string;
  message: string;
  field?: string;
}

export interface TenantContext {
  tenantId: string;
  source: "x-tenant-id";
}

export interface NormalizedTokenUsageInput {
  tenantId?: string;
  eventId?: string;
  workerId: string;
  workerName: string;
  jobId?: string;
  sessionId?: string;
  runId?: string;
  jobName?: string;
  provider: string;
  model: string;
  promptTokens: number;
  completionTokens: number;
  cachedTokens: number;
  inputTokens: number;
  outputTokens: number;
  latencyMs?: number;
  taskCategory: string;
  outputStatus: OutputStatus;
  reviewScore: number | null;
  occurredAt: string;
  externalEstimateUsd: number | null;
  externalEstimateCurrency: string | null;
  clientCostEstimateUsd?: number;
  metadata: Record<string, unknown> | null;
}

export interface CostCalculation {
  status: "priced" | "degraded";
  costEstimateUsd: number | null;
  currency: "USD";
  degradedReason: string | null;
  pricingSource: "default" | "env" | "none";
  diagnostics: ApiIssue[];
}

export interface TokenUsageEventRecord {
  tenantId: string;
  id: string;
  workerId: string;
  workerName: string;
  jobId: string | null;
  sessionId: string | null;
  runId: string | null;
  provider: string;
  model: string;
  promptTokens: number;
  completionTokens: number;
  cachedTokens: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  costEstimateUsd: number | null;
  externalEstimateUsd: number | null;
  externalEstimateCurrency: string | null;
  costCurrency: "USD";
  costIsDegraded: boolean;
  costDegradedReason: string | null;
  latencyMs: number | null;
  taskCategory: string;
  outputStatus: OutputStatus;
  reviewScore: number | null;
  occurredAt: string;
  createdAt: string;
  metadata: Record<string, unknown> | null;
}

export interface CostSnapshotRecord {
  tenantId: string;
  id: string;
  eventId: string;
  provider: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  cachedTokens: number;
  costEstimateUsd: number | null;
  currency: "USD";
  isDegraded: boolean;
  degradedReason: string | null;
  createdAt: string;
}

export type AnomalyType =
  | "spend_spike"
  | "token_spike"
  | "latency_spike"
  | "unknown_model_pricing"
  | "repeated_failed_outputs"
  | "high_cost_per_accepted_output";

export type AnomalySeverity = "info" | "warning" | "critical";

export interface AnomalySignalRecord {
  tenantId: string;
  id: string;
  eventId: string | null;
  workerId: string | null;
  type: AnomalyType;
  severity: AnomalySeverity;
  message: string;
  observedValue: number | null;
  thresholdValue: number | null;
  details: Record<string, unknown> | null;
  occurredAt: string;
  createdAt: string;
}

export interface ProductivitySummary {
  tenantId: string;
  periodStart: string | null;
  periodEnd: string | null;
  generatedAt: string;
  totalCostUsd: number;
  knownCostEventCount: number;
  unknownCostEventCount: number;
  totalEvents: number;
  outputCount: number;
  avgLatencyMs: number | null;
  anomalyCount: number;
  costPerAcceptedOutputWithReview: number | null;
  costByWorker: WorkerCostBreakdown[];
  costByCategory: CategoryCostBreakdown[];
  topCostDrivers: CostDriver[];
  degraded: ApiIssue[];
}

export interface WorkerCostBreakdown {
  workerId: string;
  workerName: string;
  eventCount: number;
  outputCount: number;
  failedOutputCount: number;
  totalTokens: number;
  totalCostUsd: number;
  unknownCostEventCount: number;
  avgLatencyMs: number | null;
  anomalyCount: number;
  costPerAcceptedOutputWithReview: number | null;
}

export interface CategoryCostBreakdown {
  taskCategory: string;
  eventCount: number;
  outputCount: number;
  totalCostUsd: number;
  avgLatencyMs: number | null;
}

export interface CostDriver {
  type: "worker" | "category" | "model";
  key: string;
  label: string;
  totalCostUsd: number;
  eventCount: number;
}

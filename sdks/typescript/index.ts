export interface TokenGoblinOptions {
  apiKey?: string;
  baseUrl?: string;
  timeoutMs?: number;
}

export class TokenGoblinClient {
  private apiKey: string;
  private baseUrl: string;
  private timeoutMs: number;

  constructor(options: TokenGoblinOptions = {}) {
    this.apiKey = options.apiKey || process.env.TOKEN_GOBLIN_API_KEY || "";
    this.baseUrl = (options.baseUrl || "http://localhost:8080").replace(/\/+$/, "");
    this.timeoutMs = options.timeoutMs ?? 10_000;

    if (!this.apiKey) {
      throw new Error("API Key must be provided or set in TOKEN_GOBLIN_API_KEY environment variable");
    }
    if (!Number.isFinite(this.timeoutMs) || this.timeoutMs <= 0) {
      throw new Error("timeoutMs must be a positive number");
    }
  }

  private async request<T>(path: string, method: string = "GET", body?: unknown): Promise<T> {
    const headers: Record<string, string> = {
      "Authorization": `Bearer ${this.apiKey}`,
      "Content-Type": "application/json"
    };

    const res = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.timeout(this.timeoutMs),
    });

    if (!res.ok) {
      throw new Error(`TokenGoblin API Error: ${res.status} ${res.statusText}`);
    }

    return (await res.json()) as T;
  }

  // V1 Endpoints
  async ingestEvent<T = unknown>(event: Record<string, unknown>): Promise<T> {
    return this.request<T>("/v1/events", "POST", event);
  }

  async ingestBatch<T = unknown>(events: Array<Record<string, unknown>>): Promise<T> {
    return this.request<T>("/v1/events/batch", "POST", events);
  }

  async getRecommendations<T = unknown>(): Promise<T> {
    return this.request<T>("/v1/dashboard/recommendations");
  }

  async getAnomalies<T = unknown>(): Promise<T> {
    return this.request<T>("/v1/dashboard/anomalies");
  }

  // V2 Endpoints - Founder Mode
  async getWasteReport<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/intelligence/waste");
  }

  async getPromptGraveyard<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/intelligence/prompt-graveyard");
  }

  async getZombieAgents<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/intelligence/zombie-agents");
  }

  async getDuplicateClusters<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/intelligence/duplicates");
  }

  async getCostLeaks<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/intelligence/cost-leaks");
  }

  async getHallucinationMap<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/intelligence/hallucination-map");
  }

  async getSpendForecast<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/forecasts/spend");
  }

  async getExecutiveScorecard<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/executive/scorecard");
  }

  async getModelComparison<T = unknown>(): Promise<T> {
    return this.request<T>("/v2/analytics/models");
  }
}

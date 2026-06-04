export interface TokenGoblinOptions {
  apiKey?: string;
  baseUrl?: string;
}

export class TokenGoblinClient {
  private apiKey: string;
  private baseUrl: string;

  constructor(options: TokenGoblinOptions = {}) {
    this.apiKey = options.apiKey || process.env.TOKEN_GOBLIN_API_KEY || "";
    this.baseUrl = options.baseUrl || "http://localhost:8080";

    if (!this.apiKey) {
      throw new Error("API Key must be provided or set in TOKEN_GOBLIN_API_KEY environment variable");
    }
  }

  private async request<T>(path: string, method: string = "GET", body?: any): Promise<T> {
    const headers: Record<string, string> = {
      "Authorization": `Bearer ${this.apiKey}`,
      "Content-Type": "application/json"
    };

    const res = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined
    });

    if (!res.ok) {
      throw new Error(`TokenGoblin API Error: ${res.status} ${res.statusText}`);
    }

    return res.json();
  }

  // V1 Endpoints
  async ingestEvent(event: any): Promise<any> {
    return this.request("/v1/events", "POST", event);
  }

  async ingestBatch(events: any[]): Promise<any> {
    return this.request("/v1/events/batch", "POST", events);
  }

  async getRecommendations(): Promise<any> {
    return this.request("/v1/dashboard/recommendations");
  }

  async getAnomalies(): Promise<any> {
    return this.request("/v1/dashboard/anomalies");
  }

  // V2 Endpoints - Founder Mode
  async getWasteReport(): Promise<any> {
    return this.request("/v2/intelligence/waste");
  }

  async getPromptGraveyard(): Promise<any> {
    return this.request("/v2/intelligence/prompt-graveyard");
  }

  async getZombieAgents(): Promise<any> {
    return this.request("/v2/intelligence/zombie-agents");
  }

  async getDuplicateClusters(): Promise<any> {
    return this.request("/v2/intelligence/duplicates");
  }

  async getCostLeaks(): Promise<any> {
    return this.request("/v2/intelligence/cost-leaks");
  }

  async getHallucinationMap(): Promise<any> {
    return this.request("/v2/intelligence/hallucination-map");
  }

  async getSpendForecast(): Promise<any> {
    return this.request("/v2/forecasts/spend");
  }

  async getExecutiveScorecard(): Promise<any> {
    return this.request("/v2/executive/scorecard");
  }

  async getModelComparison(): Promise<any> {
    return this.request("/v2/analytics/models");
  }
}

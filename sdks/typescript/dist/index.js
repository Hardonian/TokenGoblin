"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenGoblinClient = void 0;
class TokenGoblinClient {
    apiKey;
    baseUrl;
    constructor(options = {}) {
        this.apiKey = options.apiKey || process.env.TOKEN_GOBLIN_API_KEY || "";
        this.baseUrl = options.baseUrl || "http://localhost:8080";
        if (!this.apiKey) {
            throw new Error("API Key must be provided or set in TOKEN_GOBLIN_API_KEY environment variable");
        }
    }
    async request(path, method = "GET", body) {
        const headers = {
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
    async ingestEvent(event) {
        return this.request("/v1/events", "POST", event);
    }
    async ingestBatch(events) {
        return this.request("/v1/events/batch", "POST", events);
    }
    async getRecommendations() {
        return this.request("/v1/dashboard/recommendations");
    }
    async getAnomalies() {
        return this.request("/v1/dashboard/anomalies");
    }
    // V2 Endpoints - Founder Mode
    async getWasteReport() {
        return this.request("/v2/intelligence/waste");
    }
    async getPromptGraveyard() {
        return this.request("/v2/intelligence/prompt-graveyard");
    }
    async getZombieAgents() {
        return this.request("/v2/intelligence/zombie-agents");
    }
    async getDuplicateClusters() {
        return this.request("/v2/intelligence/duplicates");
    }
    async getCostLeaks() {
        return this.request("/v2/intelligence/cost-leaks");
    }
    async getHallucinationMap() {
        return this.request("/v2/intelligence/hallucination-map");
    }
    async getSpendForecast() {
        return this.request("/v2/forecasts/spend");
    }
    async getExecutiveScorecard() {
        return this.request("/v2/executive/scorecard");
    }
    async getModelComparison() {
        return this.request("/v2/analytics/models");
    }
}
exports.TokenGoblinClient = TokenGoblinClient;

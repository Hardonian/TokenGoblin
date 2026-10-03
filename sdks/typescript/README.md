# TokenGoblin TypeScript SDK

Official TypeScript/JavaScript client for [TokenGoblin](https://tokengoblin.com) — AI/LLM spend observability, token cost attribution, anomaly detection, and spend forecasting.

## Installation

```bash
npm install @tokengoblin/sdk
# or
pnpm add @tokengoblin/sdk
```

## Quick Start

```typescript
import { TokenGoblinClient } from "@tokengoblin/sdk";

const client = new TokenGoblinClient({
  apiKey: process.env.TOKEN_GOBLIN_API_KEY,
  baseUrl: "https://api.tokengoblin.com", // or http://localhost:8080
});

// Ingest a single LLM usage event
await client.ingestEvent({
  worker_id: "agent-researcher-1",
  model: "gpt-4o",
  prompt_tokens: 820,
  completion_tokens: 150,
  cost_usd: 0.0051,
  latency_ms: 640,
  status: "success",
  task: "market_research",
});

// Ingest a batch of events
await client.ingestBatch([
  {
    worker_id: "agent-coder",
    model: "claude-3-5-sonnet-20241022",
    prompt_tokens: 2400,
    completion_tokens: 800,
    cost_usd: 0.0192,
    latency_ms: 1200,
  },
]);

// Query intelligence & forecasts
const waste = await client.getWasteReport();
const zombies = await client.getZombieAgents();
const forecast = await client.getSpendForecast();

console.log("Spend Forecast:", forecast);
```

## Configuration Options

- `apiKey` (string): Your TokenGoblin API key (defaults to `TOKEN_GOBLIN_API_KEY` environment variable).
- `baseUrl` (string, optional): Base URL of the TokenGoblin API (default: `http://localhost:8080`).
- `timeoutMs` (number, optional): Request timeout in milliseconds (default: `10000`).

## License

MIT License. See [LICENSE](../../LICENSE) for details.

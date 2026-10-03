# TokenGoblin Python SDK

Official Python client for [TokenGoblin](https://tokengoblin.com) — AI/LLM spend observability, token cost attribution, anomaly detection, and spend forecasting.

## Installation

```bash
pip install token-goblin
```

## Quick Start

```python
import os
from token_goblin import TokenGoblinClient

# Initialize client with your API key
client = TokenGoblinClient(
    api_key=os.environ.get("TOKEN_GOBLIN_API_KEY"),
    base_url="https://api.tokengoblin.com"  # or http://localhost:8080
)

# Ingest an LLM usage event
client.ingest_event({
    "worker_id": "agent-researcher-1",
    "model": "gpt-4o",
    "prompt_tokens": 820,
    "completion_tokens": 150,
    "cost_usd": 0.0051,
    "latency_ms": 640,
    "status": "success",
    "task": "competitive_analysis"
})

# Ingest a batch of events
client.ingest_batch([
    {
        "worker_id": "agent-summarizer",
        "model": "claude-3-5-sonnet-20241022",
        "prompt_tokens": 1200,
        "completion_tokens": 300,
        "cost_usd": 0.0081,
        "latency_ms": 720,
    }
])

# Query intelligence endpoints
waste = client.get_waste_report()
zombies = client.get_zombie_agents()
forecast = client.get_spend_forecast()

print(f"Projected Monthly Spend: ${forecast.get('projected_cost', 0):.2f}")
```

## Context Manager

The client can be used as a context manager for deterministic resource teardown:

```python
with TokenGoblinClient(api_key="your-api-key") as client:
    client.ingest_event(...)
```

## License

MIT License. See [LICENSE](../../LICENSE) for details.

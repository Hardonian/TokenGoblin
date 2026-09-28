"""
Token Goblin Python SDK

A client library for the Token Goblin platform.
"""

import os
from typing import Any, Dict, List, Optional

import requests

class TokenGoblinClient:
    def __init__(
        self,
        api_key: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        timeout_seconds: float = 10.0,
    ):
        self.api_key = api_key or os.environ.get("TOKEN_GOBLIN_API_KEY")
        if not self.api_key:
            raise ValueError("API Key must be provided or set in TOKEN_GOBLIN_API_KEY environment variable")
        if timeout_seconds <= 0:
            raise ValueError("timeout_seconds must be positive")
        self.base_url = base_url.rstrip("/")
        self.timeout_seconds = timeout_seconds
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        })

    def close(self) -> None:
        """Release the underlying HTTP connection pool."""
        self.session.close()

    def __enter__(self) -> "TokenGoblinClient":
        return self

    def __exit__(self, exc_type: Any, exc_value: Any, traceback: Any) -> None:
        self.close()

    def ingest_event(self, event: Dict[str, Any]) -> Dict[str, Any]:
        """Ingest a single token usage event."""
        response = self.session.post(
            f"{self.base_url}/v1/events", json=event, timeout=self.timeout_seconds
        )
        response.raise_for_status()
        return response.json()

    def ingest_batch(self, events: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Ingest a batch of token usage events."""
        response = self.session.post(
            f"{self.base_url}/v1/events/batch", json=events, timeout=self.timeout_seconds
        )
        response.raise_for_status()
        return response.json()

    def get_recommendations(self) -> Dict[str, Any]:
        """Get model routing recommendations to save costs."""
        response = self.session.get(
            f"{self.base_url}/v1/dashboard/recommendations", timeout=self.timeout_seconds
        )
        response.raise_for_status()
        return response.json()

    def get_anomalies(self) -> Dict[str, Any]:
        """Get detected anomalies (spend spikes, latency issues, etc)."""
        response = self.session.get(
            f"{self.base_url}/v1/dashboard/anomalies", timeout=self.timeout_seconds
        )
        response.raise_for_status()
        return response.json()

    # V2 Endpoints - Founder Mode
    
    def get_waste_report(self) -> Dict[str, Any]:
        """Get a comprehensive waste analysis for the tenant."""
        response = self.session.get(f"{self.base_url}/v2/intelligence/waste", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()

    def get_prompt_graveyard(self) -> Dict[str, Any]:
        """Get prompts consuming cost with zero value."""
        response = self.session.get(f"{self.base_url}/v2/intelligence/prompt-graveyard", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()
        
    def get_zombie_agents(self) -> Dict[str, Any]:
        """Get agents with high activity but low value."""
        response = self.session.get(f"{self.base_url}/v2/intelligence/zombie-agents", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()
        
    def get_duplicate_clusters(self) -> Dict[str, Any]:
        """Get prompts used identically across multiple workers."""
        response = self.session.get(f"{self.base_url}/v2/intelligence/duplicates", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()
        
    def get_cost_leaks(self) -> Dict[str, Any]:
        """Get patterns of silent invisible spending."""
        response = self.session.get(f"{self.base_url}/v2/intelligence/cost-leaks", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()
        
    def get_hallucination_map(self) -> Dict[str, Any]:
        """Get failure clusters by model and category."""
        response = self.session.get(f"{self.base_url}/v2/intelligence/hallucination-map", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()
        
    def get_spend_forecast(self) -> Dict[str, Any]:
        """Get a monthly spend forecast with confidence intervals."""
        response = self.session.get(f"{self.base_url}/v2/forecasts/spend", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()
        
    def get_executive_scorecard(self) -> Dict[str, Any]:
        """Get an AI scorecard for leadership."""
        response = self.session.get(f"{self.base_url}/v2/executive/scorecard", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()
        
    def get_model_comparison(self) -> Dict[str, Any]:
        """Get side-by-side model cost/quality/latency analysis."""
        response = self.session.get(f"{self.base_url}/v2/analytics/models", timeout=self.timeout_seconds)
        response.raise_for_status()
        return response.json()

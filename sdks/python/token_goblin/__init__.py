"""
Token Goblin Python SDK

A client library for the Token Goblin platform.
"""

import os
import requests
from typing import Dict, Any, List

class TokenGoblinClient:
    def __init__(self, api_key: str = None, base_url: str = "http://localhost:8080"):
        self.api_key = api_key or os.environ.get("TOKEN_GOBLIN_API_KEY")
        if not self.api_key:
            raise ValueError("API Key must be provided or set in TOKEN_GOBLIN_API_KEY environment variable")
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        })

    def ingest_event(self, event: Dict[str, Any]) -> Dict[str, Any]:
        """Ingest a single token usage event."""
        response = self.session.post(f"{self.base_url}/v1/events", json=event)
        response.raise_for_status()
        return response.json()

    def ingest_batch(self, events: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Ingest a batch of token usage events."""
        response = self.session.post(f"{self.base_url}/v1/events/batch", json=events)
        response.raise_for_status()
        return response.json()

    def get_recommendations(self) -> Dict[str, Any]:
        """Get model routing recommendations to save costs."""
        response = self.session.get(f"{self.base_url}/v1/dashboard/recommendations")
        response.raise_for_status()
        return response.json()

    def get_anomalies(self) -> Dict[str, Any]:
        """Get detected anomalies (spend spikes, latency issues, etc)."""
        response = self.session.get(f"{self.base_url}/v1/dashboard/anomalies")
        response.raise_for_status()
        return response.json()

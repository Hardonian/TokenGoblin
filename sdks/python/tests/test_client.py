import os
import unittest
from unittest.mock import MagicMock, patch

from token_goblin import TokenGoblinClient


class TestTokenGoblinClient(unittest.TestCase):
    def test_init_missing_key_raises(self):
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(ValueError) as ctx:
                TokenGoblinClient(api_key=None)
            self.assertIn("API Key must be provided", str(ctx.exception))

    def test_init_invalid_timeout_raises(self):
        with self.assertRaises(ValueError):
            TokenGoblinClient(api_key="test-key", timeout_seconds=0)
        with self.assertRaises(ValueError):
            TokenGoblinClient(api_key="test-key", timeout_seconds=-5.0)

    def test_init_strips_trailing_slash(self):
        client = TokenGoblinClient(api_key="test-key", base_url="https://api.tokengoblin.com///")
        self.assertEqual(client.base_url, "https://api.tokengoblin.com")
        self.assertEqual(client.session.headers["Authorization"], "Bearer test-key")
        self.assertEqual(client.session.headers["Content-Type"], "application/json")
        client.close()

    def test_init_from_env(self):
        with patch.dict(os.environ, {"TOKEN_GOBLIN_API_KEY": "env-key-123"}):
            client = TokenGoblinClient()
            self.assertEqual(client.api_key, "env-key-123")
            client.close()

    def test_context_manager(self):
        with patch.object(TokenGoblinClient, "close") as mock_close:
            with TokenGoblinClient(api_key="test-key") as client:
                self.assertIsNotNone(client)
            mock_close.assert_called_once()

    @patch("requests.Session.post")
    def test_ingest_event(self, mock_post):
        mock_resp = MagicMock()
        mock_resp.json.return_value = {"status": "ok", "event_id": "evt-123"}
        mock_resp.raise_for_status.return_value = None
        mock_post.return_value = mock_resp

        client = TokenGoblinClient(api_key="test-key", base_url="http://localhost:8080")
        event = {"model": "gpt-4o", "prompt_tokens": 100, "completion_tokens": 50}
        res = client.ingest_event(event)

        mock_post.assert_called_once_with(
            "http://localhost:8080/v1/events",
            json=event,
            timeout=10.0,
        )
        self.assertEqual(res["status"], "ok")
        client.close()

    @patch("requests.Session.get")
    def test_get_spend_forecast(self, mock_get):
        mock_resp = MagicMock()
        mock_resp.json.return_value = {"projected_cost": 4250.0, "confidence_level": 0.95}
        mock_resp.raise_for_status.return_value = None
        mock_get.return_value = mock_resp

        client = TokenGoblinClient(api_key="test-key", base_url="http://localhost:8080")
        res = client.get_spend_forecast()

        mock_get.assert_called_once_with(
            "http://localhost:8080/v2/forecasts/spend",
            timeout=10.0,
        )
        self.assertEqual(res["projected_cost"], 4250.0)
        client.close()


if __name__ == "__main__":
    unittest.main()

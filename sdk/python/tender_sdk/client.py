import json
from typing import Any, Dict, Optional

import requests


class TenderClient:
    """A minimal HTTP client for interacting with the Tender node RPC layer."""

    def __init__(self, base_url: str = "http://127.0.0.1:8080") -> None:
        self.base_url = base_url.rstrip("/")

    def health(self) -> Dict[str, Any]:
        response = requests.get(f"{self.base_url}/health", timeout=5)
        response.raise_for_status()
        return response.json()

    def get_chain(self) -> Dict[str, Any]:
        response = requests.get(f"{self.base_url}/api/chain", timeout=5)
        response.raise_for_status()
        return response.json()

    def register_model(self, payload: Dict[str, Any]) -> Dict[str, Any]:
        response = requests.post(f"{self.base_url}/api/proposals", json=payload, timeout=5)
        response.raise_for_status()
        return response.json()

    def send_transaction(self, payload: Dict[str, Any]) -> Dict[str, Any]:
        response = requests.post(f"{self.base_url}/api/transfer", json=payload, timeout=5)
        response.raise_for_status()
        return response.json()

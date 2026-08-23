#!/usr/bin/env python3
"""Berth static prediction service.
Analyzes a cloned repository and returns a RuntimeProfile.
"""
import json
import os
import sys
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path


class PredictorHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/predict":
            self.send_error(404)
            return

        content_len = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_len)
        try:
            req = json.loads(body)
        except json.JSONDecodeError:
            self.send_error(400)
            return

        git_url = req.get("git_url", "")
        local_path = req.get("local_path", "")  # worker clones first, passes path
        branch = req.get("branch", "main")

        profile = self.predict(local_path)
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(profile).encode())

    def predict(self, local_path: str) -> dict:
        """Detect language from files in local_path and return profile."""
        if not local_path or not os.path.isdir(local_path):
            return self._node_fallback()

        root = Path(local_path)
        files = {f.name for f in root.iterdir() if f.is_file()}

        if "package.json" in files:
            return {
                "language": "node",
                "base_image": "node:20-alpine",
                "install_cmd": "npm install",
                "start_cmd": "npm run dev",
                "port": 3000,
                "needs_db": False,
                "confidence": 0.95,
            }

        if "requirements.txt" in files or "pyproject.toml" in files:
            return {
                "language": "python",
                "base_image": "python:3.11-alpine",
                "install_cmd": "pip install -r requirements.txt",
                "start_cmd": "python app.py",
                "port": 5000,
                "needs_db": False,
                "confidence": 0.92,
            }

        if "go.mod" in files:
            return {
                "language": "go",
                "base_image": "golang:1.23-alpine",
                "install_cmd": "go mod download",
                "start_cmd": "go run .",
                "port": 8080,
                "needs_db": False,
                "confidence": 0.94,
            }

        if "Cargo.toml" in files:
            return {
                "language": "rust",
                "base_image": "rust:1.79-alpine",
                "install_cmd": "cargo build",
                "start_cmd": "cargo run",
                "port": 8080,
                "needs_db": False,
                "confidence": 0.93,
            }

        return self._node_fallback()

    def _node_fallback(self) -> dict:
        return {
            "language": "node",
            "base_image": "node:20-alpine",
            "install_cmd": "npm install",
            "start_cmd": "npm run dev",
            "port": 3000,
            "needs_db": False,
            "confidence": 0.5,
        }

    def log_message(self, format, *args):
        # Suppress default logging; use print for structured output
        pass


def run(port: int = 50052):
    server = HTTPServer(("0.0.0.0", port), PredictorHandler)
    print(f"Berth predictor listening on :{port}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    run(int(sys.argv[1]) if len(sys.argv) > 1 else 50052)

#!/usr/bin/env python3
"""Berth prediction service with XGBoost integration.
Analyzes a cloned repository and returns a RuntimeProfile.
"""
import json
import os
import sys
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path

import pandas as pd
import xgboost as xgb
import joblib

# Global model state
LANG_MODEL = None
FEATURE_COLS = None
LABEL_ENCODER = None

def load_models():
    global LANG_MODEL, FEATURE_COLS, LABEL_ENCODER
    model_dir = Path(__file__).parent / "models"
    
    try:
        LANG_MODEL = xgb.XGBClassifier()
        LANG_MODEL.load_model(model_dir / "lang_model.json")
        FEATURE_COLS = joblib.load(model_dir / "feature_cols.joblib")
        LABEL_ENCODER = joblib.load(model_dir / "label_encoder.joblib")
        print(f"Loaded models from {model_dir}")
    except Exception as e:
        print(f"Warning: Failed to load models: {e}. Using rule-based fallback.")
        LANG_MODEL = None
        FEATURE_COLS = None
        LABEL_ENCODER = None

def rule_based_fallback(feats: dict) -> dict:
    if feats.get("has_package_json"):
        profile = get_profile_for_language("node")
    elif feats.get("has_requirements") or feats.get("has_pyproject") or feats.get("file_count_py", 0) > 0:
        profile = get_profile_for_language("python")
    elif feats.get("has_go_mod") or feats.get("file_count_go", 0) > 0:
        profile = get_profile_for_language("go")
    elif feats.get("has_cargo") or feats.get("file_count_rs", 0) > 0:
        profile = get_profile_for_language("rust")
    else:
        profile = get_profile_for_language("node")
    
    profile["confidence"] = 0.5
    return profile

def extract_features(local_path: str) -> dict:
    """Extract features from a cloned repository."""
    root = Path(local_path)
    files = {f.name for f in root.iterdir() if f.is_file()}
    all_files = list(root.rglob("*"))

    ext_counts = {}
    for f in all_files:
        if f.is_file():
            ext = f.suffix.lower()
            ext_counts[ext] = ext_counts.get(ext, 0) + 1

    features = {
        "has_package_json": int("package.json" in files),
        "has_go_mod": int("go.mod" in files),
        "has_requirements": int("requirements.txt" in files),
        "has_pyproject": int("pyproject.toml" in files),
        "has_cargo": int("Cargo.toml" in files),
        "has_dockerfile": int("Dockerfile" in files),
        "file_count_js": ext_counts.get(".js", 0) + ext_counts.get(".ts", 0),
        "file_count_go": ext_counts.get(".go", 0),
        "file_count_py": ext_counts.get(".py", 0),
        "file_count_rs": ext_counts.get(".rs", 0),
        "total_files": len(all_files),
    }
    return features

def get_profile_for_language(lang: str) -> dict:
    if lang == "node":
        return {
            "language": "node",
            "base_image": "docker.io/library/node:20-alpine",
            "install_cmd": "npm install",
            "start_cmd": "npm run dev",
            "port": 3000,
            "needs_db": False,
        }
    elif lang == "python":
        return {
            "language": "python",
            "base_image": "docker.io/library/python:3.11-alpine",
            "install_cmd": "pip install -r requirements.txt",
            "start_cmd": "python app.py",
            "port": 5000,
            "needs_db": False,
        }
    elif lang == "go":
        return {
            "language": "go",
            "base_image": "docker.io/library/golang:1.23-alpine",
            "install_cmd": "go mod download",
            "start_cmd": "go run .",
            "port": 8080,
            "needs_db": False,
        }
    elif lang == "rust":
        return {
            "language": "rust",
            "base_image": "docker.io/library/rust:1.79-alpine",
            "install_cmd": "cargo build",
            "start_cmd": "cargo run",
            "port": 8080,
            "needs_db": False,
        }
    else:
        return {
            "language": "node",
            "base_image": "docker.io/library/node:20-alpine",
            "install_cmd": "npm install",
            "start_cmd": "npm run dev",
            "port": 3000,
            "needs_db": False,
        }

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
        """Use XGBoost model to predict language."""
        if not local_path or not os.path.isdir(local_path):
            profile = get_profile_for_language("node")
            profile["confidence"] = 0.0
            return profile

        # Extract features
        feats = extract_features(local_path)
        
        # Use fallback if models aren't loaded
        if LANG_MODEL is None or FEATURE_COLS is None:
            return rule_based_fallback(feats)
        
        # Build DataFrame
        df = pd.DataFrame([feats])
        
        # Ensure correct column order
        X = df[FEATURE_COLS]
        
        # Predict
        probs = LANG_MODEL.predict_proba(X)[0]
        pred_idx = probs.argmax()
        confidence = float(probs[pred_idx])
        
        # Map back to language string
        predicted_lang = LABEL_ENCODER.inverse_transform([pred_idx])[0]
        
        profile = get_profile_for_language(predicted_lang)
        profile["confidence"] = confidence
        return profile

    def log_message(self, format, *args):
        # Suppress default logging; use print for structured output
        pass

def run(port: int = 50052):
    load_models()
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

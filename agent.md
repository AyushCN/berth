# Agent Context

This is a Phase 2 completed platform for Berth.
Currently implemented: containerd runtime integration, advanced warm pool, layer caching, NATS event bus, repository structure, fully wired API handlers.
The codebase has been refactored for cloud-scale deployment, removing all hardcoded localhost values in favor of environment variables.
Frontend UI includes File Explorer and Code Editor components, though no CRDT real-time sync exists yet.

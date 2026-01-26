---
sidebar_position: 2
---

# Architecture

VibeAuracle is built as a modular ecosystem where the Brain orchestrates multiple specialized components.

## Core Components

### The Brain (`internal/brain`)
The cognitive orchestrator. It manages the execution loop, session state, and coordinates between model providers and system tools.

### Model Providers (`internal/model`)
A pluggable system for AI engines. Supports:
- **Ollama**: Local inference.
- **OpenAI**: Cloud-based GPT models.
- **GitHub Models**: Azure-hosted models via GitHub PAT.
- **Copilot SDK**: Official GitHub Copilot integration via sidecar process.

### Prompt System (`internal/prompt`)
A multi-layered prompt builder that handles:
- **Intent Classification**: Identifying if the user wants to Ask, Plan, or Implement (CRUD).
- **Instruction Layering**: Injecting system rules, project-specific vibes, and architectural knowledge.
- **Recall Injection**: Hydrating prompts with relevant context from the persistent memory.

### Context & Memory (`internal/context`)
A dual-layer memory system:
- **Short-Term (Window)**: A rolling context of recent interactions.
- **Long-Term (DB)**: Persistent SQLite-backed storage for facts, project knowledge, and chat sessions.

### Tooling (`internal/tooling`)
A secure registry of system-aware tools (filesystem access, system stats, etc.) that the agent can invoke to perform real-world actions.

## The Cognitive Loop

1. **Perceive**: The brain receives a request and takes a system snapshot (CWD, CPU, RAM, Git SHA).
2. **Contextualize**: It pulls project-specific architectural info and recent relevant chat history.
3. **Plan**: The agent selects a strategy (Ask, Plan, CRUD) and builds the augmented prompt.
4. **Execute**: The agent runs the model and optionally executes tool calls.
5. **Reflect**: The agent observes tool outputs and repeats the loop if necessary (up to 10 turns).

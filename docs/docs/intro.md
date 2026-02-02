---
sidebar_position: 1
---

# Introduction

Welcome to **vibe auracle**, a Distributed, System-Intimate AI Engineering Ecosystem.

VibeAuracle unifies the terminal, the IDE, and the AI assistant into a single, keyboard-centric interface. It is built for engineers who value **system-intimacy**—where the AI isn't just a chatbot, but a co-pilot with deep access to your hardware, configuration, and environment.

## 🌌 The Vision
VibeAuracle is not just a CLI tool; it is a "God-tier" replacement for the fragmented developer workflow. It's built for engineers who want:
- **System-Awareness**: Real-time awareness of system resources, filesystem, and Git state.
- **Action-First**: Optimized for executing tasks directly rather than just talking about them.
- **Modular Agentic Runtimes**: Choose the best engine for your task—whether it's our artisan internal loop or the official GitHub Copilot SDK.
- **Keyboard-Centric**: Designed for speed and fluid terminal workflows.

## 🤖 Modular Agentic Runtimes
VibeAuracle is a multi-engine orchestrator. Choose the runtime that fits your task:

- **🎨 Vibe Agent (`/agent /vibe`)**: Our artisan internal loop. Highly transparent, uses custom heuristic loop-detection, and optimized for system-intimate tasks.
- **🚀 Copilot SDK Agent (`/agent /sdk`)**: Native GitHub Copilot SDK runtime. Delegates multi-step reasoning to the official GitHub agentic engine for deep tool-intimacy and secure, high-stakes engineering.
- **👤 Custom Agent (`/agent /custom`)**: User-defined agent personas. Register specialized agents with custom system prompts and restricted toolsets to create focused experts for specific workflows.

## ⚡ Core Features

- **Multi-Engine Support**: Switch between Vibe, Copilot SDK, and Custom agent personas on the fly.
- **Deep Integration**: Real-time awareness of system stats, files, and Git state.
- **Directory-Aware Sessions**: Robust PROJECT isolation using directory-hashed session IDs.
- **Deep Project Knowledge**: Persistent architectural indexing tied to Git commit history.
- **Universal Updates**: Seamless, background updates tracking Git SHAs directly.
- **Multi-Provider**: Works with Ollama, OpenAI, and GitHub Models out of the box.
- **Cleanup First**: First-class `uninstall` support to preserve privacy and system hygiene.

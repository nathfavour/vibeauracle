---
sidebar_position: 10
---

# GitHub Models Integration

*Note: This is an internal technical blueprint for the VibeAuracle engineering team.*

This document provides a technical blueprint for integrating AI capabilities using the **GitHub Models API**. While VibeAuracle supports multiple providers, this integration pattern is core to our "Universal API" strategy.

## 1. Architecture Overview

The integration follows a "Unified API" pattern. Instead of integrating multiple SDKs (OpenAI, Anthropic, Mistral), we use the **GitHub Models API**, which provides an OpenAI-compatible interface for dozens of leading models.

### Key Components
- **Model Manager**: Handles discovery and caching of available models.
- **API Client**: Manages authentication and HTTP communication.
- **Prompt Builder**: Structures application data into messages for the LLM.

## 2. Authentication

The API uses a standard GitHub Personal Access Token (PAT) as a Bearer token. No special SDK is required; standard HTTP headers are sufficient.

## 3. Model Discovery & Caching

One of the most powerful features of the GitHub Models API is the ability to dynamically discover available models. This allows VibeAuracle to automatically support new models as they are added to the platform.

### Fetching Models
The `/models` endpoint returns metadata for all available models, including their tasks (e.g., `chat-completion`).

## 4. Inference Integration

The core interaction uses a Chat Completion schema. This involves a `system` prompt to define behavior and a `user` prompt for the actual data.

## 5. Advanced Capabilities

### Multi-Model Support
By using a unified API, users can switch between models like `gpt-4o`, `Phi-3`, or `Mistral-large` without code changes.

### Context Compression
When dealing with large inputs (like code diffs), it's essential to compress data to fit within token limits and reduce costs.

### Streaming
The API supports Server-Sent Events (SSE) for streaming responses, which we use in the TUI for real-time updates.

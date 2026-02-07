<div align="center">
  <img src="./assets/vibeaura.png" width="128" alt="Vibe Auracle Logo" />

  # vibe auracle
  **Distributed, System-Intimate AI Engineering Ecosystem**

  > [!IMPORTANT]
  > **🔮 AURACLE MODE**: Press `Ctrl + Y` to enable an autonomous, human-like agentic loop that independently analyzes and improves your repository. It assumes its own personality to handle issues and patterns with zero user interference.

  <img src="./assets/shot.png" width="100%" alt="Vibe Auracle TUI Screenshot" />

  [![Stable](https://img.shields.io/badge/Stable-ec1de87-10B981?style=for-the-badge&logo=git&logoColor=white)](https://github.com/nathfavour/vibeauracle/tree/release)
  [![Beta](https://img.shields.io/badge/Beta-7ebb650-7C3AED?style=for-the-badge&logo=git&logoColor=white)](https://github.com/nathfavour/vibeauracle/tree/master)
  [![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
  [![License](https://img.shields.io/badge/License-MIT-F59E0B?style=for-the-badge)](LICENSE)
</div>

---

## 🚀 Quick Install

### 🐧 Linux / 🍎 macOS / 🤖 Android
```bash
curl -fsSL https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.sh | sh
```

### 🪟 Windows
```powershell
iex (irm https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.ps1)
```

> **Pro Tip:** One install is all you need. Keep the vibes fresh with `vibeaura update`.

---

## 🧠 Why? (The Motivation)
There are many reasons—some quite selfish—for why I’m building this. I realized I spend an immense amount of time coding, and I wanted to optimize that existence for three core reasons:

1. **The Agentic Future**: An agentic workflow is clearly the future of software development.
2. **Velocity**: Models will always be faster than humans. Throughout human history, speed has been the primary catalyst for the vast majority of innovation.
3. **Sovereignty**: I code so much that I need to control the internals of the tools I use, rather than blindly relying on closed-source black boxes.

**vibe auracle** is a lifelong project for me. I want to be able to work in my sleep. Nearly a decade ago, I told my brother that "passive productivity" was the ultimate goal. Coming from a Mathematics and Computer Science background, this is my attempt at achieving that dream.

---

## 🌌 The Vision
**vibe auracle** unifies the terminal, the IDE, and the AI assistant into a single, keyboard-centric interface. It's built for engineers who value **system-intimacy**—where the AI isn't just a chatbot, but a co-pilot with deep access to your hardware, configuration, and environment.

## 🤖 Modular Agentic Runtimes
VibeAuracle is a multi-engine orchestrator. Choose the runtime that fits your task:

- **🎨 Vibe Agent (`/agent /vibe`)**: Our artisan internal loop. Highly transparent, uses custom heuristic loop-detection, and optimized for system-intimate tasks.
- **🚀 Copilot SDK Agent (`/agent /sdk`)**: Native GitHub Copilot SDK runtime. Delegates multi-step reasoning to the official GitHub agentic engine for deep tool-intimacy and secure, high-stakes engineering.
- **👤 Custom Agent (`/agent /custom`)**: User-defined agent personas. Register specialized agents with custom system prompts and restricted toolsets to create focused experts for specific workflows.

Use the `/agent` command in the TUI to toggle between engines on the fly.

## ⚡ Core Features
- **Keyboard-Centric**: Designed for speed and fluid terminal workflows.
- **Deep Integration**: Real-time awareness of system stats, files, and Git state.
- **Universal Updates**: Seamless, background updates tracking Git SHAs directly.
- **Multi-Provider**: Works with Ollama, OpenAI, and GitHub Models out of the box.
- **Cleanup First**: First-class `uninstall` support to preserve privacy and system hygiene.

---

## 🛠️ Usage
| Command | Description |
|---------|-------------|
| `vibeaura` | Launch the interactive session |
| `vibeaura update` | Pull the latest SHA from your current branch |
| `vibeaura sys stats` | View real-time system power snapshot |
| `vibeaura models` | Discover and switch AI providers |

### 🗑️ Uninstall & Clean
We respect your space. To remove the tool but keep your data:
```bash
vibeaura uninstall
```
To wipe **everything** (binary + secrets + config):
```bash
vibeaura uninstall --clean
```

---

## 🤝 Contributing
We love community "Vibes"! Check out [CONTRIBUTING.md](./CONTRIBUTING.md) to add your own agent skills.

<div align="center">
  <sub>Built with 💜 for the future of engineering.</sub>
</div>
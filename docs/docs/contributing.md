---
sidebar_position: 9
---

# Contributing

Thank you for considering contributing to **vibe auracle**! We have a **Zero Friction Policy**. Whether you're fixing a typo, optimizing a cognitive loop, or adding a completely new "Vibe," your contribution is welcome.

## 🛠️ How to Contribute

1. **Fork the repo** and create your branch.
2. **Make your changes.** We value progress and vibes over strict bureaucracy.
3. **Open a Pull Request.**

## 🎭 Creating Your Own "Vibe"

There are two ways to extend VibeAuracle:

### 1. Markdown-based Vibes (Natural Language)
These are `.vibe.md` files that use YAML front matter for configuration and Markdown for instructions. They are perfect for zero-code extensions.
See the [Vibes Documentation](./vibes.md) for more details.

### 2. Go-native Vibes (Advanced)
For high-performance or complex logic, you can contribute a Go-native vibe.

1. **Create your Vibe directory**:
   ```bash
   mkdir -p vibes/my-cool-vibe
   cd vibes/my-cool-vibe
   go mod init github.com/nathfavour/vibeauracle/vibes/my-cool-vibe
   ```

2. **Implement the Vibe Interface**:
   Implement the `Vibe` interface from `pkg/vibe`.

3. **Register your Vibe**:
   Add your module to the root `go.work` file.

## 🧠 Agent Skills
If you want to add a specific "Skill" (a functional action the brain can take), you can contribute directly to `internal/brain` or define it within your native Vibe.

## 📜 Code of Conduct
Be excellent to each other. That's the only rule.

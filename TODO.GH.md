# 🐙 GitHub CLI Leverage Roadmap (TODO.GH.md)

Objective: Transform VibeAuracle into a "Claude Code" competitor by leveraging technical patterns and installed instances of the GitHub CLI.

## 1. Technical Leverage: Infrastructure Resilience
- [x] **Custom GitHub Transport**: Implement a specialized `http.RoundTripper` in `internal/model` to mirror `gh`'s `capiTransport`.
    - Inject `Copilot-Integration-Id: copilot-4-cli` and `X-GitHub-Api-Version` headers.
    - Add user-agent strings matching the official CLI for better API compatibility.
- [x] **Robust Backoff Strategy**: Port the `cenkalti/backoff` patterns from `gh` into the `vibe-brain` execution loop to handle rate limits and transient network failures during long-running agent tasks.

## 2. Contextual Intelligence: Project-Native Instructions
- [x] **Agent Instruction Discovery**: Update the prompt system to automatically scan for and ingest `.github/agents/*.md` or `.github/vibeaura/*.md` files.
    - This allows teams to define project-specific coding standards, architecture rules, and "vibes" that the agent respects by default.
- [x] **Repository Identity**: Use `gh repo view --json` to populate the agent's initial context with accurate repository metadata (visibility, license, primary language, topics).

## 3. Operational Leverage: Tooling Orchestration
- [x] **Advanced PR Visibility**: Expand the `gh_pr_manage` tool to fetch PR diffs and conversation history via `gh pr diff` and `gh pr view --json comments`.
    - This gives VibeAuracle the ability to "read" code reviews and fix requested changes autonomously.
- [x] **Extension Awareness**: Add a tool to detect and potentially invoke installed `gh` extensions, allowing VibeAuracle to inherit custom organizational workflows.

## 4. Hybrid Workflow: Remote Execution
- [x] **Cloud Hand-off**: Implement `gh_remote_task` to trigger `gh agent-task create`.
    - Allows the user to hand off heavy or long-running tasks from the local VibeAuracle session to GitHub's remote agent infrastructure.

package brain

import (
	"regexp"
	"strings"
)

// ShortenModelName cleans up long model identifiers for UI display.
func ShortenModelName(name string) string {
	original := name

	// 1. Remove common prefixes and suffixes
	name = strings.TrimPrefix(name, "azure-sdk-for-go/")
	name = strings.TrimSuffix(name, ":latest")

	// 2. Handle Meta Llama models
	if strings.Contains(name, "Meta-Llama") || strings.Contains(name, "Llama") {
		// Example: Meta-Llama-3.1-405B-Instruct -> Llama 3.1 405B
		// Example: meta-llama-3.1-8b-instruct -> Llama 3.1 8B
		re := regexp.MustCompile(`(?i)(?:Meta-)?Llama-?([\d\.]+)-?(\d+[BM])-?(?:Instruct|Chat)?`)
		matches := re.FindStringSubmatch(name)
		if len(matches) >= 3 {
			return "Llama " + matches[1] + " " + strings.ToUpper(matches[2])
		}
	}

	// 3. Handle OpenAI models with dates
	if strings.HasPrefix(name, "gpt-") {
		// Example: gpt-4o-2024-05-13 -> GPT-4o
		// Example: gpt-3.5-turbo-0125 -> GPT-3.5 Turbo
		re := regexp.MustCompile(`(?i)gpt-([\d\.a-z\-]+)(?:-\d{4}-\d{2}-\d{2}|-\d{4})?`)
		matches := re.FindStringSubmatch(name)
		if len(matches) >= 2 {
			res := strings.ToUpper(matches[1])
			res = strings.ReplaceAll(res, "TURBO", "Turbo")
			return "GPT-" + res
		}
	}

	// 4. Handle Phi models
	if strings.Contains(name, "Phi-") {
		re := regexp.MustCompile(`(?i)Phi-([^-]+)`)
		matches := re.FindStringSubmatch(name)
		if len(matches) >= 2 {
			return "Phi-" + matches[1]
		}
	}

	// 5. Handle Mistral/Mixtral
	if strings.Contains(name, "Mistral-") || strings.Contains(name, "Mixtral-") {
		re := regexp.MustCompile(`(?i)(Mi[sx]tral-[^-]+)`)
		matches := re.FindStringSubmatch(name)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	// 6. Generic cleanup: replace hyphens with spaces and capitalize
	if len(name) > 20 {
		return original // If it's too complex and we didn't match, keep it original
	}

	return name
}

// CleanResponse removes thinking blocks and tool calls from the response text.
func CleanResponse(raw string) string {
	// 1. Remove <thought> blocks (DeepSeek / Reasoning style)
	reThought := regexp.MustCompile(`(?s)<thought>.*?</thought>`)
	cleaned := reThought.ReplaceAllString(raw, "")

	// 2. Remove tool call blocks (fenced JSON with "tool":)
	// This ensures third-party apps don't get "broken" by internal agent JSON.
	reTool := regexp.MustCompile(`(?s)[\n\s]*` + "```" + `json\s*\{.*?"tool":\s*".*?".*?\}[\n\s]*` + "```" + `[\n\s]*`)
	cleaned = reTool.ReplaceAllString(cleaned, "\n")

	return strings.TrimSpace(cleaned)
}

// ExtractReasoning extracts the content of <thought> blocks if they exist.
func ExtractReasoning(raw string) string {
	re := regexp.MustCompile(`(?s)<thought>(.*?)</thought>`)
	matches := re.FindAllStringSubmatch(raw, -1)
	var reasoning strings.Builder
	for _, m := range matches {
		if len(m) > 1 {
			if reasoning.Len() > 0 {
				reasoning.WriteString("\n\n")
			}
			reasoning.WriteString(strings.TrimSpace(m[1]))
		}
	}
	return reasoning.String()
}

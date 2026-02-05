package brain

import (

	"context"

	"encoding/json"

	"fmt"

	"runtime"

	"time"



	"github.com/nathfavour/vibeauracle/internal/doctor"

	"github.com/nathfavour/vibeauracle/tooling"

)



// HealingState tracks the progress of a self-healing session

type HealingState struct {

	Issue       string    `json:"issue"`

	StartTime   time.Time `json:"start_time"`

	Attempts    int       `json:"attempts"`

	LastAttempt string    `json:"last_attempt"`

	Success     bool      `json:"success"`

}



// Heal initiates an autonomous self-healing loop.

// It detects failures, analyzes logs, and attempts to fix the system.

func (b *Brain) Heal(ctx context.Context, issue string) (Response, error) {

	tooling.ReportStatus("🩹", "healing", fmt.Sprintf("Initiating self-healing for: %s", issue))



	state := HealingState{

		Issue:     issue,

		StartTime: time.Now(),

	}



	maxAttempts := 5

	for i := 0; i < maxAttempts; i++ {

		state.Attempts = i + 1

		tooling.ReportStatus("🩹", "healing", fmt.Sprintf("Attempt %d/%d...", state.Attempts, maxAttempts))



		// 1. Gather Context (Logs + System State)

		logs := doctor.GetRecentLogs(20)

		logStr, _ := json.MarshalIndent(logs, "", "  ")



		snapshot, _ := b.monitor.GetSnapshot()



		// 2. Formulate Fix Strategy

		prompt := fmt.Sprintf(`SYSTEM IS EXPERIENCING A FAILURE.

Goal: Diagnose and fix the issue autonomously.



ISSUE: %s



RECENT LOGS:

%s



SYSTEM SNAPSHOT:

CWD: %s

OS: %s

ARCH: %s



Your task is to:

1. Use 'tester' to reproduce the failure and confirm current state.

2. Use 'sys_read_file' or 'grep' to investigate the code causing the failure.

3. Use 'sys_patch' or 'sys_write_file' to apply a fix.

4. Use 'tester' again to verify the fix.



Output your first tool call in JSON format.`, issue, string(logStr), snapshot.WorkingDir, runtime.GOOS, runtime.GOARCH)

		// We use a specialized "Healer" persona by overriding the prompt
		resp, err := b.Process(ctx, Request{
			ID:      fmt.Sprintf("heal_%d_%d", time.Now().Unix(), i),
			Content: prompt,
			Intent:  "implement", // Force execution mode
		})

		if err != nil {
			tooling.ReportStatus("❌", "healing", fmt.Sprintf("Healing attempt failed: %v", err))
			continue
		}

		// 3. Check for Success
		// In a real implementation, we might check if the 'tester' tool in the last turn returned "success".
		// For now, we'll look at the agent's final word or run a verification test.
		tooling.ReportStatus("🧪", "verify", "Verifying healing results...")
		verifyResult, _ := b.tools.Get("tester")
		if verifyResult != nil {
			res, err := verifyResult.Execute(ctx, json.RawMessage(`{}`))
			if err == nil && res.Status == "success" {
				state.Success = true
				tooling.ReportStatus("✨", "healed", "System successfully self-healed!")
				return Response{
					Content: fmt.Sprintf("Self-healing successful!\n\nSummary: %s", resp.Content),
					Metadata: map[string]interface{}{
						"healing_state": state,
					},
				}, nil
			}
		}

		state.LastAttempt = resp.Content
	}

	return Response{
		Content: "Self-healing failed after maximum attempts. Manual intervention required.",
		Metadata: map[string]interface{}{
			"healing_state": state,
		},
	}, nil
}

// TriggerHealingIfNecessary checks system health and triggers healing if catastrophic.
func (b *Brain) TriggerHealingIfNecessary(ctx context.Context) {
	health := doctor.AnalyzeHealth()
	if health == doctor.HealthCatastrophic {
		logs := doctor.GetRecentLogs(5)
		var lastError string
		if len(logs) > 0 {
			lastError = logs[len(logs)-1].Message
		}
		
		go func() {
			_, _ = b.Heal(ctx, "Catastrophic system failure detected: "+lastError)
		}()
	}
}

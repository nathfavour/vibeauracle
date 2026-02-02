package tooling

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// RiskLevel defines the severity of a shell command risk.
type RiskLevel string

const (
	RiskSafe     RiskLevel = "safe"
	RiskLow      RiskLevel = "low"      // Might read sensitive info
	RiskMedium   RiskLevel = "medium"   // Modifies system state but recoverable
	RiskHigh     RiskLevel = "high"     // Potential data loss or system compromise
	RiskCritical RiskLevel = "critical" // Highly destructive or obfuscated
)

// ShellRisk contains the results of a command analysis.
type ShellRisk struct {
	Level   RiskLevel
	Reasons []string
}

// ShellGuard analyzes shell commands for dangerous operations.
type ShellGuard struct {
	// Commands that are always risky
	riskyCommands map[string]RiskLevel
	// Targets that are dangerous to touch
	dangerousTargets []string
	// Flag combinations that increase risk (e.g. -rf)
	riskyFlags map[string]RiskLevel
}

func NewShellGuard() *ShellGuard {
	return &ShellGuard{
		riskyCommands: map[string]RiskLevel{
			"rm":       RiskHigh,
			"dd":       RiskCritical,
			"mkfs":     RiskCritical,
			"format":   RiskCritical,
			"shred":    RiskHigh,
			"chmod":    RiskMedium,
			"chown":    RiskMedium,
			"curl":     RiskMedium, // Downloading and piping is risky
			"wget":     RiskMedium,
			"scp":      RiskMedium,
			"nc":       RiskHigh,
			"netcat":   RiskHigh,
			"nmap":     RiskMedium,
			"iptables": RiskHigh,
			"ufw":      RiskHigh,
			"passwd":   RiskHigh,
			"useradd":  RiskHigh,
			"deluser":  RiskHigh,
			"visudo":   RiskCritical,
		},
		dangerousTargets: []string{
			"/", "/etc", "/dev", "/var", "/boot", "/proc", "/sys", "/root",
			"/home", "/bin", "/sbin", "/usr/bin", "/usr/sbin",
		},
		riskyFlags: map[string]RiskLevel{
			"-rf": RiskHigh,
			"-f":  RiskMedium,
		},
	}
}

// Analyze parses a shell command and returns a risk assessment.
func (sg *ShellGuard) Analyze(command string) ShellRisk {
	p := syntax.NewParser()
	f, err := p.Parse(strings.NewReader(command), "")
	if err != nil {
		return ShellRisk{Level: RiskHigh, Reasons: []string{"Failed to parse command (potential obfuscation)"}}
	}

	risk := ShellRisk{Level: RiskSafe, Reasons: []string{}}

	// Walk the AST
	syntax.Walk(f, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CallExpr:
			sg.analyzeCall(n, &risk)
		case *syntax.BinaryCmd:
			// Detect pipes to sh/bash
			sg.analyzeBinaryCmd(n, &risk)
		case *syntax.Redirect:
			sg.analyzeRedirect(n, &risk)
		}
		return true
	})

	return risk
}

func (sg *ShellGuard) analyzeCall(ce *syntax.CallExpr, risk *ShellRisk) {
	if len(ce.Args) == 0 {
		return
	}

	// First argument is the command
	cmdName := sg.nodeToString(ce.Args[0])
	
	// Check risky commands
	if level, ok := sg.riskyCommands[cmdName]; ok {
		sg.elevateRisk(risk, level, fmt.Sprintf("Uses risky command: %s", cmdName))
	}

	// Check arguments for dangerous targets or flags
	fullCmd := ""
	for _, arg := range ce.Args {
		argStr := sg.nodeToString(arg)
		fullCmd += argStr + " "

		// Check for dangerous targets
		for _, target := range sg.dangerousTargets {
			if argStr == target || strings.HasPrefix(argStr, target+"/") {
				sg.elevateRisk(risk, RiskHigh, fmt.Sprintf("Targets sensitive directory: %s", argStr))
			}
		}

		// Check for risky flags
		for flag, level := range sg.riskyFlags {
			if strings.Contains(argStr, flag) {
				sg.elevateRisk(risk, level, fmt.Sprintf("Uses risky flag: %s", flag))
			}
		}
	}

	// Detect "curl ... | sh" patterns (this might also be caught in analyzeBinaryCmd)
	if (cmdName == "curl" || cmdName == "wget") && strings.Contains(fullCmd, "|") {
		sg.elevateRisk(risk, RiskCritical, "Detected network download piped to another command")
	}
}

func (sg *ShellGuard) analyzeBinaryCmd(be *syntax.BinaryCmd, risk *ShellRisk) {
	if be.Op == syntax.Pipe {
		// Detect piping into a shell
		rightSide := sg.nodeToString(be.Y)
		if strings.Contains(rightSide, "sh") || strings.Contains(rightSide, "bash") || strings.Contains(rightSide, "zsh") || strings.Contains(rightSide, "python") {
			sg.elevateRisk(risk, RiskCritical, "Detected pipe into a shell or interpreter")
		}
	}
}

func (sg *ShellGuard) analyzeRedirect(r *syntax.Redirect, risk *ShellRisk) {
	if r.Op == syntax.RdrOut || r.Op == syntax.AppOut {
		target := sg.nodeToString(r.Word)
		for _, dt := range sg.dangerousTargets {
			if target == dt || strings.HasPrefix(target, dt+"/") {
				sg.elevateRisk(risk, RiskHigh, fmt.Sprintf("Redirects output to sensitive directory: %s", target))
			}
		}
	}
}

func (sg *ShellGuard) nodeToString(node syntax.Node) string {
	var sb strings.Builder
	p := syntax.NewPrinter()
	p.Print(&sb, node)
	return sb.String()
}

func (sg *ShellGuard) elevateRisk(risk *ShellRisk, level RiskLevel, reason string) {
	risk.Reasons = append(risk.Reasons, reason)
	
	levels := map[RiskLevel]int{
		RiskSafe:     0,
		RiskLow:      1,
		RiskMedium:   2,
		RiskHigh:     3,
		RiskCritical: 4,
	}

	if levels[level] > levels[risk.Level] {
		risk.Level = level
	}
}
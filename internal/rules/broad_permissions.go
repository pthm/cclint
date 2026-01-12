package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pthm-cable/cclint/internal/parser"
)

// BroadPermissionsRule checks for overly broad permission configurations
type BroadPermissionsRule struct{}

func (r *BroadPermissionsRule) Name() string {
	return "broad-permissions"
}

func (r *BroadPermissionsRule) Description() string {
	return "Checks for overly broad or risky permission configurations"
}

func (r *BroadPermissionsRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryConfig,
		},
	}
}

func (r *BroadPermissionsRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	// Check settings.json for broad permissions
	settingsPath := filepath.Join(ctx.RootPath, ".claude", "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		issues = append(issues, r.checkSettings(settingsPath, data)...)
	}

	// Also check settings.local.json
	localSettingsPath := filepath.Join(ctx.RootPath, ".claude", "settings.local.json")
	if data, err := os.ReadFile(localSettingsPath); err == nil {
		issues = append(issues, r.checkSettings(localSettingsPath, data)...)
	}

	return issues, nil
}

func (r *BroadPermissionsRule) checkSettings(path string, data []byte) []Issue {
	var issues []Issue

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return issues
	}

	// Check for dangerous glob patterns in allowedTools
	if allowed, ok := settings["allowedTools"].([]interface{}); ok {
		for _, tool := range allowed {
			if toolStr, ok := tool.(string); ok {
				if r.isDangerousPattern(toolStr) {
					issues = append(issues, Issue{
						Rule:     r.Name() + "/dangerous-pattern",
						Severity: Warning,
						Message:  fmt.Sprintf("Overly broad tool permission: %s", toolStr),
						File:     path,
						Line:     1,
					})
				}
			}
		}
	}

	// Check for dangerous Bash permissions
	if bash, ok := settings["bash"].(map[string]interface{}); ok {
		if allowed, ok := bash["allow"].([]interface{}); ok {
			for _, pattern := range allowed {
				if patternStr, ok := pattern.(string); ok {
					if r.isDangerousBashPattern(patternStr) {
						issues = append(issues, Issue{
							Rule:     r.Name() + "/dangerous-bash-pattern",
							Severity: Warning,
							Message:  fmt.Sprintf("Overly broad bash permission: %s", patternStr),
							File:     path,
							Line:     1,
						})
					}
				}
			}
		}
	}

	return issues
}

func (r *BroadPermissionsRule) isDangerousPattern(pattern string) bool {
	dangerous := []string{
		"*",
		"**",
		"Bash(*)",
		"Edit(*)",
		"Write(*)",
	}

	for _, d := range dangerous {
		if pattern == d || strings.HasPrefix(pattern, d) {
			return true
		}
	}
	return false
}

func (r *BroadPermissionsRule) isDangerousBashPattern(pattern string) bool {
	dangerous := []string{
		"*",
		"rm -rf *",
		"sudo *",
		"curl | sh",
		"curl | bash",
	}

	for _, d := range dangerous {
		if pattern == d || strings.Contains(pattern, d) {
			return true
		}
	}
	return false
}

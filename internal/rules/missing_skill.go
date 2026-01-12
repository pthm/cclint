package rules

import (
	"fmt"

	"github.com/pthm/cclint/internal/analyzer"
)

// MissingSkillRule checks that skills declared in frontmatter exist in the project
type MissingSkillRule struct{}

func (r *MissingSkillRule) Name() string {
	return "missing-skill"
}

func (r *MissingSkillRule) Description() string {
	return "Checks that skills declared in frontmatter exist in .claude/skills/"
}

func (r *MissingSkillRule) Config() RuleConfig {
	return RuleConfig{} // Applies to all file types
}

func (r *MissingSkillRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	scopes, err := ctx.Scopes()
	if err != nil {
		return nil, err
	}

	// Build set of available skill names from discovered skill scopes
	availableSkills := make(map[string]bool)
	for _, scope := range scopes {
		if scope.Type == analyzer.ScopeTypeMain {
			// Skills are children of main scope
			for _, child := range scope.Children {
				if child.Type == analyzer.ScopeTypeSkill {
					availableSkills[child.Name] = true
				}
			}
		}
	}

	// Check each subagent's declared skills
	for _, scope := range scopes {
		if scope.Type != analyzer.ScopeTypeSubagent {
			continue
		}

		for _, skill := range scope.DeclaredSkills {
			if !availableSkills[skill] {
				issues = append(issues, Issue{
					Rule:     r.Name(),
					Severity: Warning,
					Message:  fmt.Sprintf("Skill '%s' not found in .claude/skills/", skill),
					File:     scope.Entrypoint,
					Line:     1, // Frontmatter is at the top
					Context:  fmt.Sprintf("Declared in frontmatter of subagent '%s'", scope.Name),
					Fix: &Fix{
						Description: fmt.Sprintf("Create skill at .claude/skills/%s.md or .claude/skills/%s/SKILL.md", skill, skill),
					},
				})
			}
		}
	}

	return issues, nil
}

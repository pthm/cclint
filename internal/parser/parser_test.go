package parser

import (
	"testing"
)

func TestGetFileCategory(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected FileCategory
	}{
		// Config files
		{
			name:     "settings.json",
			path:     "/project/.claude/settings.json",
			expected: FileCategoryConfig,
		},
		{
			name:     "settings.local.json",
			path:     "/project/.claude/settings.local.json",
			expected: FileCategoryConfig,
		},
		{
			name:     "mcp.json",
			path:     "/project/mcp.json",
			expected: FileCategoryConfig,
		},
		{
			name:     ".mcp.json",
			path:     "/project/.mcp.json",
			expected: FileCategoryConfig,
		},

		// Instructions files
		{
			name:     "CLAUDE.md at root",
			path:     "/project/CLAUDE.md",
			expected: FileCategoryInstructions,
		},
		{
			name:     "CLAUDE.md in .claude",
			path:     "/project/.claude/CLAUDE.md",
			expected: FileCategoryInstructions,
		},
		{
			name:     "CLAUDE lowercase",
			path:     "/project/claude.md",
			expected: FileCategoryInstructions,
		},

		// Commands files
		{
			name:     "command in .claude/commands",
			path:     "/project/.claude/commands/commit.md",
			expected: FileCategoryCommands,
		},
		{
			name:     "nested command",
			path:     "/project/.claude/commands/git/push.md",
			expected: FileCategoryCommands,
		},

		// Documentation files
		{
			name:     "README.md",
			path:     "/project/README.md",
			expected: FileCategoryDocumentation,
		},
		{
			name:     "readme.md lowercase",
			path:     "/project/readme.md",
			expected: FileCategoryDocumentation,
		},
		{
			name:     "file in docs directory",
			path:     "/project/docs/guide.md",
			expected: FileCategoryDocumentation,
		},
		{
			name:     "file in nested docs",
			path:     "/project/docs/api/endpoints.md",
			expected: FileCategoryDocumentation,
		},

		// Unknown files
		{
			name:     "random markdown file",
			path:     "/project/notes.md",
			expected: FileCategoryUnknown,
		},
		{
			name:     "source code",
			path:     "/project/src/main.go",
			expected: FileCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFileCategory(tt.path)
			if got != tt.expected {
				t.Errorf("GetFileCategory(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestFileCategoryString(t *testing.T) {
	tests := []struct {
		category FileCategory
		expected string
	}{
		{FileCategoryConfig, "config"},
		{FileCategoryInstructions, "instructions"},
		{FileCategoryCommands, "commands"},
		{FileCategoryDocumentation, "documentation"},
		{FileCategoryUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.category.String(); got != tt.expected {
				t.Errorf("FileCategory.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

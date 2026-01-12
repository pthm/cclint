package rules

import (
	"fmt"
	"strings"
)

// CircularRefsRule checks for circular references in the configuration tree
type CircularRefsRule struct{}

func (r *CircularRefsRule) Name() string {
	return "circular-refs"
}

func (r *CircularRefsRule) Description() string {
	return "Checks for circular references between configuration files"
}

func (r *CircularRefsRule) Config() RuleConfig {
	return RuleConfig{} // Applies to all file types
}

func (r *CircularRefsRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	// Build adjacency list
	graph := make(map[string][]string)
	for _, node := range ctx.AllFiles() {
		for _, child := range node.Children {
			graph[node.Path] = append(graph[node.Path], child.Path)
		}
	}

	// Find cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var detectCycle func(node string) bool
	detectCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if detectCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}

				if cycleStart >= 0 {
					cycle := append(path[cycleStart:], neighbor)
					issues = append(issues, Issue{
						Rule:     r.Name(),
						Severity: Warning,
						Message:  fmt.Sprintf("Circular reference detected: %s", strings.Join(cycle, " -> ")),
						File:     neighbor,
						Line:     1,
					})
				}
				return true
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	for node := range graph {
		if !visited[node] {
			detectCycle(node)
		}
	}

	return issues, nil
}

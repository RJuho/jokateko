package validator

import (
	"slices"
	"strings"
)

const (
	colorWhite = 0 // unvisited
	colorGray  = 1 // currently visiting (in stack)
	colorBlack = 2 // fully visited
)

// DetectCycles analyzes a directed task dependency graph (taskID -> list of dependencyIDs)
// and returns diagnostics for any circular dependencies found.
func DetectCycles(deps map[string][]string, taskFiles map[string]string) []Diagnostic {
	var diagnostics []Diagnostic
	reportedCycles := make(map[string]bool)

	// Sort task IDs for deterministic traversal
	nodes := make([]string, 0, len(deps))
	for node := range deps {
		nodes = append(nodes, node)
	}
	slices.Sort(nodes)

	color := make(map[string]int, len(deps))
	var path []string

	var dfs func(u string)
	dfs = func(u string) {
		color[u] = colorGray
		path = append(path, u)

		for _, v := range deps[u] {
			if _, exists := deps[v]; !exists {
				// Dependency not in graph (handled by TSK-007)
				continue
			}

			if color[v] == colorGray {
				// Found a cycle! Reconstruct path from v to end of path, plus v
				cycleStartIdx := -1
				for i, p := range path {
					if p == v {
						cycleStartIdx = i
						break
					}
				}

				if cycleStartIdx != -1 {
					cycleNodes := make([]string, 0, len(path)-cycleStartIdx+1)
					cycleNodes = append(cycleNodes, path[cycleStartIdx:]...)
					cycleNodes = append(cycleNodes, v)

					canonicalKey := canonicalCycleKey(cycleNodes)
					if !reportedCycles[canonicalKey] {
						reportedCycles[canonicalKey] = true

						filePath := taskFiles[u]
						if filePath == "" {
							filePath = u
						}

						cycleStr := strings.Join(cycleNodes, " -> ")
						diagnostics = append(diagnostics, Diagnostic{
							RuleID:   "DAG-001",
							Severity: SeverityError,
							File:     filePath,
							Message:  "Circular task dependency detected:",
							Context: []string{
								cycleStr,
							},
							Fix: "Remove the circular reference from 'dependencies' in one of these tasks.",
						})
					}
				}
			} else if color[v] == colorWhite {
				dfs(v)
			}
		}

		path = path[:len(path)-1]
		color[u] = colorBlack
	}

	for _, node := range nodes {
		if color[node] == colorWhite {
			dfs(node)
		}
	}

	return diagnostics
}

// canonicalCycleKey produces a rotated representation of the cycle starting from
// the lexicographically smallest element to deduplicate cycles encountered from different entry points.
func canonicalCycleKey(cycle []string) string {
	if len(cycle) <= 1 {
		return strings.Join(cycle, "->")
	}

	// The last element is a duplicate of the first (closing the loop)
	nodes := cycle[:len(cycle)-1]
	minIdx := 0
	for i := 1; i < len(nodes); i++ {
		if nodes[i] < nodes[minIdx] {
			minIdx = i
		}
	}

	rotated := make([]string, len(nodes))
	for i := range nodes {
		rotated[i] = nodes[(minIdx+i)%len(nodes)]
	}

	return strings.Join(rotated, "->")
}

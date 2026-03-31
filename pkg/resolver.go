package pkg

import (
	"fmt"
	"strings"
)

// resolveGraph validates the dependency graph and returns providers in topological order.
// Singletons come first (in dependency order), then prototypes.
func resolveGraph(providers []Provider) ([]Provider, error) {
	// Build provider map: canonical return type → provider index
	typeToIdx := make(map[string]int)
	for i, p := range providers {
		key := p.ReturnType.FullName()
		if prev, exists := typeToIdx[key]; exists {
			return nil, fmt.Errorf(
				"duplicate provider for type %s:\n  %s.%s (%s)\n  %s.%s (%s)",
				key,
				providers[prev].ImportPath, providers[prev].FuncName, providers[prev].File,
				p.ImportPath, p.FuncName, p.File,
			)
		}
		typeToIdx[key] = i
	}

	// Validate all dependencies can be satisfied
	for _, p := range providers {
		for _, dep := range p.Params {
			depKey := dep.FullName()
			if _, ok := typeToIdx[depKey]; !ok {
				return nil, fmt.Errorf(
					"unsatisfied dependency: %s.%s (%s) requires %s, but no provider found",
					p.ImportPath, p.FuncName, p.File, depKey,
				)
			}
		}
	}

	// Topological sort using Kahn's algorithm
	n := len(providers)
	adj := make([][]int, n) // adj[i] = providers that depend on i
	inDeg := make([]int, n)

	for i, p := range providers {
		for _, dep := range p.Params {
			depIdx := typeToIdx[dep.FullName()]
			adj[depIdx] = append(adj[depIdx], i)
			inDeg[i]++
		}
	}

	// Start with providers that have no dependencies
	var queue []int
	for i := 0; i < n; i++ {
		if inDeg[i] == 0 {
			queue = append(queue, i)
		}
	}

	var sorted []Provider
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		sorted = append(sorted, providers[idx])

		for _, next := range adj[idx] {
			inDeg[next]--
			if inDeg[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(sorted) != n {
		// Find cycle for error message
		cycle := findCycle(providers, typeToIdx)
		return nil, fmt.Errorf("circular dependency detected: %s", cycle)
	}

	return sorted, nil
}

// findCycle returns a human-readable description of a dependency cycle.
func findCycle(providers []Provider, typeToIdx map[string]int) string {
	n := len(providers)
	visited := make([]int, n) // 0=unvisited, 1=in-stack, 2=done
	parent := make([]int, n)
	for i := range parent {
		parent[i] = -1
	}

	var cyclePath []string

	var dfs func(idx int) bool
	dfs = func(idx int) bool {
		visited[idx] = 1
		p := providers[idx]
		for _, dep := range p.Params {
			depIdx := typeToIdx[dep.FullName()]
			if visited[depIdx] == 1 {
				// Found cycle — trace back
				cyclePath = append(cyclePath, providers[depIdx].ImportPath+"."+providers[depIdx].FuncName)
				for cur := idx; cur != depIdx; cur = parent[cur] {
					cyclePath = append(cyclePath, providers[cur].ImportPath+"."+providers[cur].FuncName)
				}
				cyclePath = append(cyclePath, providers[depIdx].ImportPath+"."+providers[depIdx].FuncName)
				// Reverse
				for i, j := 0, len(cyclePath)-1; i < j; i, j = i+1, j-1 {
					cyclePath[i], cyclePath[j] = cyclePath[j], cyclePath[i]
				}
				return true
			}
			if visited[depIdx] == 0 {
				parent[depIdx] = idx
				if dfs(depIdx) {
					return true
				}
			}
		}
		visited[idx] = 2
		return false
	}

	for i := 0; i < n; i++ {
		if visited[i] == 0 {
			if dfs(i) {
				return strings.Join(cyclePath, " → ")
			}
		}
	}

	return "unknown cycle"
}

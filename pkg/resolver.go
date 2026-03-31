package pkg

import (
	"fmt"
	"strings"
)

// resolveGraph validates the dependency graph and returns providers in topological order.
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
		for _, param := range p.Params {
			depKey := param.Type.FullName()
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
		for _, param := range p.Params {
			depIdx := typeToIdx[param.Type.FullName()]
			adj[depIdx] = append(adj[depIdx], i)
			inDeg[i]++
		}
	}

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
		cycle := findCycle(providers, typeToIdx)
		return nil, fmt.Errorf("circular dependency detected: %s", cycle)
	}

	return sorted, nil
}

// resolveApps builds a separate dependency graph for each @app provider.
func resolveApps(apps []Provider, regular []Provider) ([]AppGroup, error) {
	// Build type → provider map from @inject providers
	typeMap := make(map[string]Provider)
	for _, p := range regular {
		key := p.ReturnType.FullName()
		if _, exists := typeMap[key]; exists {
			return nil, fmt.Errorf("duplicate provider for type %s", key)
		}
		typeMap[key] = p
	}

	var groups []AppGroup
	for _, app := range apps {
		deps, err := traceDeps(app, typeMap)
		if err != nil {
			return nil, fmt.Errorf("app %s.%s: %w", app.ImportPath, app.FuncName, err)
		}

		all := append(deps, app)
		sorted, err := resolveGraph(all)
		if err != nil {
			return nil, fmt.Errorf("app %s.%s: %w", app.ImportPath, app.FuncName, err)
		}

		groups = append(groups, AppGroup{
			App:       app,
			Providers: sorted,
			Name:      appContainerName(app),
		})
	}

	return groups, nil
}

// traceDeps recursively collects all providers required by root.
func traceDeps(root Provider, typeMap map[string]Provider) ([]Provider, error) {
	visited := make(map[string]bool)
	var result []Provider

	var visit func(p Provider) error
	visit = func(p Provider) error {
		for _, param := range p.Params {
			key := param.Type.FullName()
			if visited[key] {
				continue
			}
			visited[key] = true
			dep, ok := typeMap[key]
			if !ok {
				return fmt.Errorf("unsatisfied dependency: requires %s, but no provider found", key)
			}
			if err := visit(dep); err != nil {
				return err
			}
			result = append(result, dep)
		}
		return nil
	}

	if err := visit(root); err != nil {
		return nil, err
	}
	return result, nil
}

// appContainerName returns the container name prefix from @Application("name").
func appContainerName(app Provider) string {
	return upperFirst(app.AppName)
}

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
		for _, param := range p.Params {
			depIdx := typeToIdx[param.Type.FullName()]
			if visited[depIdx] == 1 {
				cyclePath = append(cyclePath, providers[depIdx].ImportPath+"."+providers[depIdx].FuncName)
				for cur := idx; cur != depIdx; cur = parent[cur] {
					cyclePath = append(cyclePath, providers[cur].ImportPath+"."+providers[cur].FuncName)
				}
				cyclePath = append(cyclePath, providers[depIdx].ImportPath+"."+providers[depIdx].FuncName)
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

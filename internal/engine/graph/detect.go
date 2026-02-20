// # internal/graph/detect.go
package graph

import "sort"

func (g *Graph) DetectCycles() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var cycles [][]string
	visited := make(map[string]bool)
	onStack := make(map[string]bool)

	for modName := range g.modules {
		if !visited[modName] {
			g.findCycles(modName, visited, onStack, []string{}, &cycles)
		}
	}

	return cycles
}

func (g *Graph) findCycles(curr string, visited, onStack map[string]bool, path []string, cycles *[][]string) {
	visited[curr] = true
	onStack[curr] = true
	path = append(path, curr)

	for next := range g.imports[curr] {
		if onStack[next] {
			// Found a cycle
			cycleStart := -1
			for i, mod := range path {
				if mod == next {
					cycleStart = i
					break
				}
			}
			if cycleStart != -1 {
				cycle := make([]string, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				*cycles = append(*cycles, cycle)
			}
		} else if !visited[next] {
			g.findCycles(next, visited, onStack, path, cycles)
		}
	}

	onStack[curr] = false
}

func (g *Graph) FindImportChain(from, to string) ([]string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.modules[from]; !ok {
		return nil, false
	}
	if _, ok := g.modules[to]; !ok {
		return nil, false
	}
	if from == to {
		return []string{from}, true
	}

	queue := []string{from}
	visited := map[string]bool{from: true}
	prev := make(map[string]string)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		neighbors := make([]string, 0, len(g.imports[curr]))
		for next := range g.imports[curr] {
			if _, ok := g.modules[next]; !ok {
				continue
			}
			neighbors = append(neighbors, next)
		}
		sort.Strings(neighbors)

		for _, next := range neighbors {
			if visited[next] {
				continue
			}
			visited[next] = true
			prev[next] = curr

			if next == to {
				path := []string{to}
				for node := to; node != from; {
					p, ok := prev[node]
					if !ok {
						return nil, false
					}
					path = append(path, p)
					node = p
				}
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return path, true
			}

			queue = append(queue, next)
		}
	}

	return nil, false
}

func (g *Graph) InvalidateTransitive(changedFile string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	moduleName, ok := g.fileToModule[changedFile]
	if !ok {
		return nil
	}

	toRecheck := []string{changedFile}
	seen := map[string]bool{changedFile: true}
	modSeen := map[string]bool{moduleName: true}

	queue := []string{moduleName}
	for len(queue) > 0 {
		mod := queue[0]
		queue = queue[1:]

		for importer := range g.importedBy[mod] {
			if modSeen[importer] {
				continue
			}
			modSeen[importer] = true

			if importerMod, ok := g.modules[importer]; ok {
				for _, f := range importerMod.Files {
					if !seen[f] {
						seen[f] = true
						toRecheck = append(toRecheck, f)
					}
				}
				queue = append(queue, importer)
			}
		}
	}

	return toRecheck
}

// # internal/graph/detect.go
package graph

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

func (g *Graph) InvalidateTransitive(changedFile string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	file := g.files[changedFile]
	if file == nil {
		return nil
	}

	toRecheck := []string{changedFile}
	seen := map[string]bool{changedFile: true}
	modSeen := map[string]bool{file.Module: true}

	queue := []string{file.Module}
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

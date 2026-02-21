// # internal/graph/detect.go
package graph

import "sort"

func (g *Graph) DetectCycles() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var cycles [][]string
	visited := make(map[string]bool)
	onStack := make(map[string]bool)

	// To keep detection deterministic, sort modules
	moduleNames := make([]string, 0, len(g.modules))
	for name := range g.modules {
		moduleNames = append(moduleNames, name)
	}
	sort.Strings(moduleNames)

	for _, startMod := range moduleNames {
		if visited[startMod] {
			continue
		}

		type frame struct {
			mod       string
			neighbors []string
			nextIdx   int
		}

		stack := []*frame{
			{
				mod:       startMod,
				neighbors: g.getSortedNeighbors(startMod),
				nextIdx:   0,
			},
		}
		visited[startMod] = true
		onStack[startMod] = true
		path := []string{startMod}

		for len(stack) > 0 {
			top := stack[len(stack)-1]

			if top.nextIdx < len(top.neighbors) {
				next := top.neighbors[top.nextIdx]
				top.nextIdx++

				if onStack[next] {
					// Found a cycle
					cycleStart := -1
					for i, m := range path {
						if m == next {
							cycleStart = i
							break
						}
					}
					if cycleStart != -1 {
						cycle := make([]string, len(path)-cycleStart)
						copy(cycle, path[cycleStart:])
						cycles = append(cycles, cycle)
					}
				} else if !visited[next] {
					visited[next] = true
					onStack[next] = true
					path = append(path, next)
					stack = append(stack, &frame{
						mod:       next,
						neighbors: g.getSortedNeighbors(next),
						nextIdx:   0,
					})
				}
			} else {
				// Finished with this node
				onStack[top.mod] = false
				path = path[:len(path)-1]
				stack = stack[:len(stack)-1]
			}
		}
	}

	return cycles
}

func (g *Graph) getSortedNeighbors(mod string) []string {
	neighbors := make([]string, 0, len(g.imports[mod]))
	for next := range g.imports[mod] {
		// Only consider neighbors that are in the internal module set
		if _, ok := g.modules[next]; ok {
			neighbors = append(neighbors, next)
		}
	}
	sort.Strings(neighbors)
	return neighbors
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

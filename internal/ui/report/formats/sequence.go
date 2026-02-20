package formats

// internal/ui/report/formats/sequence.go

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"fmt"
	"sort"
	"strings"
)

// sequenceHop tracks a single caller→callee message in the sequence trace.
type sequenceHop struct {
	From   string
	To     string
	Symbol string
}

// TraceCallSequence generates a Mermaid sequenceDiagram showing how modules
// call each other's exported symbols, starting from entryModule and walking
// the reference graph up to maxDepth hops.
//
// It uses the graph's References (cross-module symbol references) to build
// the sequence. When a module references a symbol that belongs to another
// module, a message arrow is emitted from caller to callee.
//
// Returns an error only if entryModule is not found in the graph.
func TraceCallSequence(g *graph.Graph, entryModule string, maxDepth int) (string, error) {
	// Validate entry module.
	if _, ok := g.GetModule(entryModule); !ok {
		return "", fmt.Errorf("sequence: entry module %q not found", entryModule)
	}

	if maxDepth <= 0 {
		maxDepth = 5
	}

	// Build a lookup: symbol name → owning module.
	// We iterate all files and record (exported) definitions.
	symbolOwner := make(map[string]string) // symbol -> module
	for _, file := range g.GetAllFiles() {
		for _, def := range file.Definitions {
			if def.Exported {
				symbolOwner[def.Name] = file.Module
				if def.FullName != "" && def.FullName != def.Name {
					symbolOwner[def.FullName] = file.Module
				}
			}
		}
	}

	// Walk the reference graph BFS-style, respecting maxDepth.
	type bfsItem struct {
		module string
		depth  int
	}

	visited := make(map[string]bool)
	queue := []bfsItem{{module: entryModule, depth: 0}}
	participants := []string{entryModule}
	participantSeen := map[string]bool{entryModule: true}
	hops := []sequenceHop{}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.module] {
			continue
		}
		visited[item.module] = true

		if item.depth >= maxDepth {
			continue
		}

		// Find all files in this module and inspect their references.
		refs := collectReferences(g, item.module)
		for _, ref := range refs {
			owner, ok := symbolOwner[ref.Name]
			if !ok || owner == item.module {
				continue
			}

			// Emit a hop from this module to the owner of the referenced symbol.
			hops = append(hops, sequenceHop{
				From:   item.module,
				To:     owner,
				Symbol: ref.Name,
			})

			if !participantSeen[owner] {
				participantSeen[owner] = true
				participants = append(participants, owner)
			}
			if !visited[owner] {
				queue = append(queue, bfsItem{module: owner, depth: item.depth + 1})
			}
		}
	}

	// De-duplicate hops (same caller→callee→symbol).
	hops = deduplicateHops(hops)

	var b strings.Builder
	b.WriteString("sequenceDiagram\n")
	b.WriteString("  autonumber\n")
	sort.Strings(participants)

	// Preserve entry module as first participant.
	sortedParticipants := make([]string, 0, len(participants))
	sortedParticipants = append(sortedParticipants, entryModule)
	for _, p := range participants {
		if p != entryModule {
			sortedParticipants = append(sortedParticipants, p)
		}
	}

	for _, p := range sortedParticipants {
		// Use the last path segment as the display alias for readability.
		alias := p
		if idx := strings.LastIndex(p, "/"); idx >= 0 {
			alias = p[idx+1:]
		}
		b.WriteString(fmt.Sprintf("  participant %s as %s\n", sanitizeID(p), escapeLabel(alias)))
	}

	b.WriteString("\n")
	for _, hop := range hops {
		fromID := sanitizeID(hop.From)
		toID := sanitizeID(hop.To)
		b.WriteString(fmt.Sprintf("  %s->>%s: %s()\n", fromID, toID, escapeLabel(hop.Symbol)))
	}

	return b.String(), nil
}

// collectReferences returns all parser.Reference entries from files belonging
// to the given module. Results are de-duplicated by name and sorted for
// deterministic output.
func collectReferences(g *graph.Graph, module string) []parser.Reference {
	refs := make([]parser.Reference, 0)
	seen := make(map[string]bool)
	for _, file := range g.GetAllFiles() {
		if file.Module != module {
			continue
		}
		for _, ref := range file.References {
			if seen[ref.Name] {
				continue
			}
			seen[ref.Name] = true
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Name < refs[j].Name
	})
	return refs
}

// deduplicateHops removes exact duplicate sequenceHop entries.
func deduplicateHops(hops []sequenceHop) []sequenceHop {
	seen := make(map[string]bool, len(hops))
	out := make([]sequenceHop, 0, len(hops))
	for _, h := range hops {
		key := h.From + "|" + h.To + "|" + h.Symbol
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, h)
	}
	return out
}

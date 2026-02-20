package formats

// internal/ui/report/formats/html_interactive.go

import (
	"circular/internal/engine/graph"
	"circular/internal/shared/util"
	"encoding/json"
	"fmt"
	"strings"
)

// htmlTreemapNode represents a single module node in the D3 treemap JSON hierarchy.
type htmlTreemapNode struct {
	Name       string            `json:"name"`
	Size       int               `json:"size,omitempty"`       // leaf node size (file count)
	Complexity int               `json:"complexity,omitempty"` // hotspot score for colour
	Importance float64           `json:"importance,omitempty"` // importance score
	Children   []htmlTreemapNode `json:"children,omitempty"`   // non-leaf: sub-modules or cluster
}

// GenerateInteractiveReport produces a self-contained HTML page embedding a
// D3 v7 zoomable treemap. Each tile represents a module:
//   - tile area    ∝ number of source files in the module
//   - tile colour  = blue (low) → red (high) based on the complexity hotspot score
//
// The report requires no server — it embeds all data as JSON and loads D3 from CDN.
func GenerateInteractiveReport(g *graph.Graph, metrics map[string]graph.ModuleMetrics, hotspots []graph.ComplexityHotspot) (string, error) {
	modules := g.Modules()
	moduleNames := util.SortedStringKeys(modules)

	// Build hotspot lookup: module -> max complexity score.
	hotspotScore := make(map[string]int, len(hotspots))
	for _, h := range hotspots {
		if cur, ok := hotspotScore[h.Module]; !ok || h.Score > cur {
			hotspotScore[h.Module] = h.Score
		}
	}

	// Group modules by their top-level path component (cluster).
	clusterMap := make(map[string][]htmlTreemapNode)
	for _, name := range moduleNames {
		mod := modules[name]
		fileCount := 0
		if mod != nil {
			fileCount = len(mod.Files)
		}
		if fileCount == 0 {
			fileCount = 1 // always give a tile some area
		}
		imp := 0.0
		if m, ok := metrics[name]; ok {
			imp = m.ImportanceScore
		}

		parts := strings.SplitN(name, "/", 2)
		cluster := parts[0]

		clusterMap[cluster] = append(clusterMap[cluster], htmlTreemapNode{
			Name:       name,
			Size:       fileCount,
			Complexity: hotspotScore[name],
			Importance: imp,
		})
	}

	// Build root children (one per cluster).
	rootChildren := make([]htmlTreemapNode, 0, len(clusterMap))
	clusterNames := util.SortedStringKeys(clusterMap)
	for _, cl := range clusterNames {
		rootChildren = append(rootChildren, htmlTreemapNode{
			Name:     cl,
			Children: clusterMap[cl],
		})
	}

	root := htmlTreemapNode{
		Name:     "root",
		Children: rootChildren,
	}

	dataJSON, err := json.Marshal(root)
	if err != nil {
		return "", fmt.Errorf("html_interactive: marshal treemap data: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Circular — Interactive Dependency Treemap</title>
<script src="https://cdn.jsdelivr.net/npm/d3@7/dist/d3.min.js"></script>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Segoe UI', system-ui, sans-serif; background: #1a1a2e; color: #eee; }
  h1 { padding: 12px 20px; font-size: 1.1rem; background: #16213e; color: #a9c4f5; }
  #legend { padding: 6px 20px; background: #16213e; font-size: 0.78rem; color: #aaa; border-top: 1px solid #2a3a5e; }
  #chart { width: 100vw; height: calc(100vh - 60px); }
  .node rect { stroke: #1a1a2e; stroke-width: 1px; rx: 3; transition: opacity 0.15s; }
  .node rect:hover { opacity: 0.85; cursor: pointer; }
  .node text { pointer-events: none; fill: #fff; font-size: 11px; text-shadow: 0 1px 2px #000; }
  #tooltip {
    position: fixed; padding: 8px 12px; background: rgba(22,33,62,0.95);
    border: 1px solid #4a6fa5; border-radius: 6px; font-size: 0.8rem;
    pointer-events: none; opacity: 0; transition: opacity 0.15s; max-width: 320px;
  }
  #tooltip dt { color: #a9c4f5; font-weight: 600; }
  #tooltip dd { color: #ddd; margin-left: 8px; }
</style>
</head>
<body>
<h1>Circular — Interactive Dependency Treemap</h1>
<div id="legend">
  Tile size = source file count &nbsp;|&nbsp; Colour: blue (low complexity) → red (high complexity) &nbsp;|&nbsp; Hover for details
</div>
<div id="chart"></div>
<div id="tooltip"></div>
<script>
const data = `)
	sb.Write(dataJSON)
	sb.WriteString(`;

const chart = document.getElementById('chart');
const tooltip = document.getElementById('tooltip');
const W = chart.clientWidth || window.innerWidth;
const H = chart.clientHeight || (window.innerHeight - 60);

const color = d3.scaleSequential()
  .domain([0, 30])
  .interpolator(d3.interpolateRdYlBu)
  .clamp(true);
// Reverse so red = high complexity.
const getColor = d => color(30 - (d.data.complexity || 0));

const root = d3.treemap()
  .size([W, H])
  .paddingOuter(6)
  .paddingTop(22)
  .paddingInner(2)
  .round(true)
  (d3.hierarchy(data).sum(d => d.size || 0).sort((a, b) => b.value - a.value));

const svg = d3.select('#chart').append('svg')
  .attr('width', W).attr('height', H);

const node = svg.selectAll('g')
  .data(root.leaves())
  .join('g')
    .attr('class', 'node')
    .attr('transform', d => 'translate(' + d.x0 + ',' + d.y0 + ')');

node.append('rect')
  .attr('width', d => Math.max(0, d.x1 - d.x0))
  .attr('height', d => Math.max(0, d.y1 - d.y0))
  .attr('fill', getColor);

node.append('text')
  .attr('x', 5).attr('y', 14)
  .text(d => {
    const w = d.x1 - d.x0;
    const name = d.data.name.split('/').pop();
    return w > 40 ? name : '';
  });

// Tooltip
node.on('mousemove', (event, d) => {
  tooltip.style.opacity = '1';
  tooltip.style.left = (event.clientX + 14) + 'px';
  tooltip.style.top  = (event.clientY - 10) + 'px';
  tooltip.innerHTML =
    '<dl>' +
    '<dt>' + d.data.name + '</dt>' +
    '<dd>Files: ' + (d.data.size || 0) + '</dd>' +
    '<dd>Complexity: ' + (d.data.complexity || 0) + '</dd>' +
    '<dd>Importance: ' + (d.data.importance ? d.data.importance.toFixed(1) : '0') + '</dd>' +
    '</dl>';
}).on('mouseleave', () => { tooltip.style.opacity = '0'; });
</script>
</body>
</html>
`)

	return sb.String(), nil
}

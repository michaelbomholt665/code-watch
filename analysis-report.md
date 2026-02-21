---
title: Code Analysis Report
project: code-watch
generated_at: 2026-02-21T21:11:08Z
version: 1.0.0
---

# Analysis Report

## Table of Contents
- [Executive Summary](#executive-summary)
- [Circular Imports](#circular-imports)
- [Architecture Violations](#architecture-violations)
- [Complexity Hotspots](#complexity-hotspots)
- [Probable Bridge References](#probable-bridge-references)
- [Unresolved References](#unresolved-references)
- [Unused Imports](#unused-imports)

## Executive Summary
| Metric | Value |
| --- | --- |
| Total Modules | 39 |
| Total Files | 148 |
| Circular Imports | 0 |
| Architecture Violations | 0 |
| Complexity Hotspots | 5 |
| Probable Bridge References | 0 |
| Unresolved References | 1174 |
| Unused Imports | 100 |

## Circular Imports
No circular imports detected.

## Architecture Violations
No architecture violations detected.

## Complexity Hotspots
| Module | Definition | File | Score | Branches | Params | Nesting | LOC |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `circular/internal/ui/cli` | `Run` | `internal/ui/cli/runtime.go` | 145 | 51 | 1 | 9 | 244 |
| `circular/internal/core/config` | `applyDefaults` | `internal/core/config/loader.go` | 131 | 53 | 1 | 3 | 187 |
| `circular/internal/core/app` | `GenerateOutputs` | `internal/core/app/output.go` | 116 | 36 | 8 | 9 | 189 |
| `circular/internal/mcp/validate` | `ParseToolArgs` | `internal/mcp/validate/args.go` | 105 | 37 | 3 | 5 | 186 |
| `circular/internal/core/config` | `validateArchitecture` | `internal/core/config/validator.go` | 97 | 31 | 1 | 11 | 120 |

## Probable Bridge References
No probable bridge references detected.

## Unresolved References
<details>
<summary>Unresolved reference details</summary>

| Reference | Location |
| --- | --- |
| `g.GetAllFiles` | `internal/ui/report/formats/sequence.go:42:23` |
| `b.WriteString` | `internal/ui/report/formats/sequence.go:107:2` |
| `b.WriteString` | `internal/ui/report/formats/sequence.go:108:2` |
| `b.WriteString` | `internal/ui/report/formats/sequence.go:126:3` |
| `b.WriteString` | `internal/ui/report/formats/sequence.go:129:2` |
| `b.WriteString` | `internal/ui/report/formats/sequence.go:133:3` |
| `b.String` | `internal/ui/report/formats/sequence.go:136:9` |
| `g.GetAllFiles` | `internal/ui/report/formats/sequence.go:145:23` |
| `a.resolveOutputRoot` | `internal/core/app/output_targets.go:21:15` |
| `helpers.ResolveOutputPath` | `internal/core/app/output_targets.go:33:13` |
| `helpers.ResolveOutputPath` | `internal/core/app/output_targets.go:34:13` |
| `helpers.ResolveDiagramPath` | `internal/core/app/output_targets.go:35:13` |
| `helpers.ResolveDiagramPath` | `internal/core/app/output_targets.go:36:13` |
| `helpers.ResolveOutputPath` | `internal/core/app/output_targets.go:37:13` |
| `config.ResolvePaths` | `internal/core/app/output_targets.go:48:16` |
| `g.Modules` | `internal/ui/report/formats/diagram_modes.go:47:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:48:17` |
| `g.GetImports` | `internal/ui/report/formats/diagram_modes.go:76:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:77:23` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:81:14` |
| `g.GetAllFiles` | `internal/ui/report/formats/diagram_modes.go:92:11` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:152:10` |
| `g.Modules` | `internal/ui/report/formats/diagram_modes.go:186:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:187:17` |
| `g.GetAllFiles` | `internal/ui/report/formats/diagram_modes.go:196:38` |
| `g.GetImports` | `internal/ui/report/formats/diagram_modes.go:198:38` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:204:12` |
| `g.GetImports` | `internal/ui/report/formats/diagram_modes.go:212:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:241:29` |
| `util.SortedStringKeys` | `internal/ui/report/formats/diagram_modes.go:250:22` |
| `r.checkModule` | `internal/engine/resolver/unresolved.go:15:5` |
| `r.graph.HasDefinitions` | `internal/engine/resolver/unresolved.go:50:8` |
| `r.checkModule` | `internal/engine/resolver/unresolved.go:51:9` |
| `r.graph.HasDefinitions` | `internal/engine/resolver/unresolved.go:67:9` |
| `r.checkModule` | `internal/engine/resolver/unresolved.go:68:10` |
| `observability.Tracer.Start` | `internal/engine/resolver/unresolved.go:88:15` |
| `span.End` | `internal/engine/resolver/unresolved.go:89:8` |
| `r.graph.GetAllFiles` | `internal/engine/resolver/unresolved.go:93:11` |
| `r.findUnresolvedInFile` | `internal/engine/resolver/unresolved.go:96:35` |
| `observability.Tracer.Start` | `internal/engine/resolver/unresolved.go:103:15` |
| `span.End` | `internal/engine/resolver/unresolved.go:104:8` |
| `r.graph.GetAllFiles` | `internal/engine/resolver/unresolved.go:108:11` |
| `r.findProbableBridgeReferencesInFile` | `internal/engine/resolver/unresolved.go:111:31` |
| `observability.Tracer.Start` | `internal/engine/resolver/unresolved.go:118:15` |
| `span.End` | `internal/engine/resolver/unresolved.go:119:8` |
| `r.graph.GetFile` | `internal/engine/resolver/unresolved.go:130:15` |
| `r.findUnresolvedInFile` | `internal/engine/resolver/unresolved.go:134:35` |
| `observability.Tracer.Start` | `internal/engine/resolver/unresolved.go:141:15` |
| `span.End` | `internal/engine/resolver/unresolved.go:142:8` |
| `r.graph.GetFile` | `internal/engine/resolver/unresolved.go:153:15` |
| `r.findProbableBridgeReferencesInFile` | `internal/engine/resolver/unresolved.go:157:31` |
| `r.resolveReferenceResult` | `internal/engine/resolver/unresolved.go:166:13` |
| `r.resolveReferenceResult` | `internal/engine/resolver/unresolved.go:183:13` |
| `watcher.Close` | `internal/core/config/watcher.go:40:3` |
| `watcher.Close` | `internal/core/config/watcher.go:47:9` |
| `g.mu.RLock` | `internal/engine/graph/detect.go:7:2` |
| `g.mu.RUnlock` | `internal/engine/graph/detect.go:8:8` |
| `g.getSortedNeighbors` | `internal/engine/graph/detect.go:35:16` |
| `g.getSortedNeighbors` | `internal/engine/graph/detect.go:70:18` |
| `g.mu.RLock` | `internal/engine/graph/detect.go:100:2` |
| `g.mu.RUnlock` | `internal/engine/graph/detect.go:101:8` |
| `g.mu.RLock` | `internal/engine/graph/detect.go:161:2` |
| `g.mu.RUnlock` | `internal/engine/graph/detect.go:162:8` |
| `w.pendingMu.Lock` | `internal/core/watcher/watcher.go:135:2` |
| `w.pendingMu.Unlock` | `internal/core/watcher/watcher.go:136:8` |
| `w.watchRecursive` | `internal/core/watcher/watcher.go:142:13` |
| `info.IsDir` | `internal/core/watcher/watcher.go:157:6` |
| `w.shouldExcludeDir` | `internal/core/watcher/watcher.go:158:7` |
| `info.IsDir` | `internal/core/watcher/watcher.go:179:22` |
| `w.shouldExcludeDir` | `internal/core/watcher/watcher.go:180:10` |
| `w.watchRecursive` | `internal/core/watcher/watcher.go:181:17` |
| `w.shouldExcludeFile` | `internal/core/watcher/watcher.go:191:7` |
| `w.scheduleChange` | `internal/core/watcher/watcher.go:199:5` |
| `w.pendingMu.Lock` | `internal/core/watcher/watcher.go:212:2` |
| `w.pendingMu.Unlock` | `internal/core/watcher/watcher.go:213:8` |
| `w.timer.Stop` | `internal/core/watcher/watcher.go:218:3` |
| `w.pendingMu.Lock` | `internal/core/watcher/watcher.go:227:2` |
| `w.pendingMu.Unlock` | `internal/core/watcher/watcher.go:233:2` |
| `info.IsDir` | `internal/core/watcher/watcher.go:248:20` |
| `g.Match` | `internal/core/watcher/watcher.go:276:6` |
| `g.Match` | `internal/core/watcher/watcher.go:304:6` |
| `w.timer.Stop` | `internal/core/watcher/watcher.go:313:3` |
| `info.IsDir` | `internal/core/watcher/watcher.go:320:35` |
| `w.shouldExcludeFile` | `internal/core/watcher/watcher.go:323:6` |
| `w.scheduleChange` | `internal/core/watcher/watcher.go:326:3` |
| `m.mu.Lock` | `internal/mcp/transport/mock_transport.go:36:2` |
| `m.mu.Unlock` | `internal/mcp/transport/mock_transport.go:38:3` |
| `m.mu.Unlock` | `internal/mcp/transport/mock_transport.go:43:2` |
| `m.mu.Lock` | `internal/mcp/transport/mock_transport.go:57:2` |
| `m.mu.Unlock` | `internal/mcp/transport/mock_transport.go:58:8` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:46:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:47:8` |
| `c.order.MoveToFront` | `internal/engine/graph/lru.go:54:2` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:61:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:62:8` |
| `c.order.MoveToFront` | `internal/engine/graph/lru.go:66:3` |
| `c.order.Len` | `internal/engine/graph/lru.go:72:5` |
| `c.evictLeastRecentLocked` | `internal/engine/graph/lru.go:73:3` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:84:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:85:8` |
| `c.order.Remove` | `internal/engine/graph/lru.go:91:2` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:97:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:98:8` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:110:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:111:8` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:122:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:123:8` |
| `c.order.Len` | `internal/engine/graph/lru.go:124:9` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:134:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:135:8` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:143:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:144:8` |
| `c.order.Len` | `internal/engine/graph/lru.go:151:6` |
| `c.evictLeastRecentLocked` | `internal/engine/graph/lru.go:152:3` |
| `c.order.Remove` | `internal/engine/graph/lru.go:163:2` |
| `c.mu.Lock` | `internal/engine/graph/lru.go:170:2` |
| `c.mu.Unlock` | `internal/engine/graph/lru.go:171:8` |
| `c.order.Len` | `internal/engine/graph/lru.go:180:12` |
| `c.evictLeastRecentLocked` | `internal/engine/graph/lru.go:183:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:50:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:51:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:52:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:53:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:54:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:55:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:57:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:59:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:60:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:61:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:62:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:63:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:64:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:65:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:66:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:68:4` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:70:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:73:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:74:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:75:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:76:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:77:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:78:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:79:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:80:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:81:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:82:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:83:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:93:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:94:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:95:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:96:3` |
| `b.String` | `internal/ui/report/formats/markdown.go:99:9` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:103:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:105:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:122:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:133:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:135:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:148:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:159:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:161:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:178:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:189:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:191:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:213:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:223:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:234:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:236:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:249:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:260:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:262:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:279:3` |
| `m.writeTableWithCollapse` | `internal/ui/report/formats/markdown.go:289:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:308:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:309:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:310:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:311:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:314:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:317:3` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:319:2` |
| `b.WriteString` | `internal/ui/report/formats/markdown.go:321:3` |
| `engine.Walk` | `internal/engine/parser/profile_extractors.go:53:2` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:59:32` |
| `node.ChildCount` | `internal/engine/parser/profile_extractors.go:61:25` |
| `node.Child` | `internal/engine/parser/profile_extractors.go:62:13` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:63:7` |
| `n.Kind` | `internal/engine/parser/profile_extractors.go:81:10` |
| `n.ChildCount` | `internal/engine/parser/profile_extractors.go:100:25` |
| `n.Child` | `internal/engine/parser/profile_extractors.go:101:9` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:105:12` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:123:37` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:128:12` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:148:37` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:168:37` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:173:12` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:203:8` |
| `engine.Walk` | `internal/engine/parser/profile_extractors.go:255:2` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:261:37` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:280:37` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:299:37` |
| `engine.Walk` | `internal/engine/parser/profile_extractors.go:343:2` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:373:2` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:378:2` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:383:2` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:388:2` |
| `node.Kind` | `internal/engine/parser/profile_extractors.go:394:5` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:397:2` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:402:37` |
| `node.ChildCount` | `internal/engine/parser/profile_extractors.go:404:25` |
| `node.Child` | `internal/engine/parser/profile_extractors.go:405:13` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:406:7` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:406:39` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:440:37` |
| `engine.Walk` | `internal/engine/parser/profile_extractors.go:484:2` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:523:2` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:528:2` |
| `e.addNamedDef` | `internal/engine/parser/profile_extractors.go:533:2` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:538:37` |
| `node.ChildCount` | `internal/engine/parser/profile_extractors.go:540:25` |
| `node.Child` | `internal/engine/parser/profile_extractors.go:541:13` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:542:7` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:542:39` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:571:36` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:623:15` |
| `node.ChildByFieldName` | `internal/engine/parser/profile_extractors.go:661:15` |
| `engine.Walk` | `internal/engine/parser/profile_extractors.go:692:2` |
| `node.ChildCount` | `internal/engine/parser/profile_extractors.go:699:24` |
| `node.Child` | `internal/engine/parser/profile_extractors.go:700:12` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:701:6` |
| `node.ChildCount` | `internal/engine/parser/profile_extractors.go:710:24` |
| `node.Child` | `internal/engine/parser/profile_extractors.go:711:11` |
| `engine.Walk` | `internal/engine/parser/profile_extractors.go:768:2` |
| `node.ChildCount` | `internal/engine/parser/profile_extractors.go:776:24` |
| `node.Child` | `internal/engine/parser/profile_extractors.go:777:12` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:778:6` |
| `child.Kind` | `internal/engine/parser/profile_extractors.go:778:40` |
| `e.ExtractRaw` | `internal/engine/parser/profile_extractors.go:812:9` |
| `e.ExtractRaw` | `internal/engine/parser/profile_extractors.go:849:9` |
| `observability.Tracer.Start` | `internal/core/app/service.go:43:15` |
| `span.End` | `internal/core/app/service.go:44:8` |
| `s.app.Graph.FileCount` | `internal/core/app/service.go:80:18` |
| `s.app.Graph.ModuleCount` | `internal/core/app/service.go:85:17` |
| `s.app.Graph.DetectCycles` | `internal/core/app/service.go:127:12` |
| `s.app.Graph.ComputeModuleMetrics` | `internal/core/app/service.go:182:13` |
| `s.app.Graph.DetectCycles` | `internal/core/app/service.go:183:12` |
| `s.app.AnalyzeHallucinations` | `internal/core/app/service.go:184:16` |
| `s.app.AnalyzeUnusedImports` | `internal/core/app/service.go:185:12` |
| `s.app.ArchitectureViolations` | `internal/core/app/service.go:188:16` |
| `s.app.Graph.TopComplexity` | `internal/core/app/service.go:194:14` |
| `s.app.Graph.ModuleCount` | `internal/core/app/service.go:202:22` |
| `s.app.Graph.FileCount` | `internal/core/app/service.go:203:22` |
| `s.app.Graph.DetectCycles` | `internal/core/app/service.go:256:12` |
| `s.app.Graph.ComputeModuleMetrics` | `internal/core/app/service.go:262:13` |
| `s.app.ArchitectureViolations` | `internal/core/app/service.go:270:16` |
| `s.app.Graph.TopComplexity` | `internal/core/app/service.go:276:14` |
| `s.app.AnalyzeHallucinations` | `internal/core/app/service.go:277:20` |
| `s.app.AnalyzeUnusedImports` | `internal/core/app/service.go:278:19` |
| `s.app.Graph.FileCount` | `internal/core/app/service.go:281:19` |
| `s.app.Graph.ModuleCount` | `internal/core/app/service.go:282:19` |
| `d.detectWithRanges` | `internal/engine/secrets/detector.go:97:9` |
| `d.detectWithRanges` | `internal/engine/secrets/detector.go:101:9` |
| `index.lineCol` | `internal/engine/secrets/detector.go:177:17` |
| `index.lineCol` | `internal/engine/secrets/detector.go:217:15` |
| `index.lineCol` | `internal/engine/secrets/detector.go:261:16` |
| `bridge.matchesSource` | `internal/engine/resolver/bridge.go:106:7` |
| `bridge.matchesReference` | `internal/engine/resolver/bridge.go:109:6` |
| `g.mu.RLock` | `internal/engine/graph/metrics.go:25:2` |
| `g.mu.RUnlock` | `internal/engine/graph/metrics.go:26:8` |
| `r.mu.RLock` | `internal/mcp/registry/registry.go:44:2` |
| `r.mu.RUnlock` | `internal/mcp/registry/registry.go:45:8` |
| `r.mu.RLock` | `internal/mcp/registry/registry.go:52:2` |
| `r.mu.RUnlock` | `internal/mcp/registry/registry.go:53:8` |
| `config.ResolvePaths` | `internal/mcp/runtime/project_context.go:31:16` |
| `config.ResolveRelative` | `internal/mcp/runtime/project_context.go:38:21` |
| `config.ResolveRelative` | `internal/mcp/runtime/project_context.go:40:28` |
| `util.WriteFileWithDirs` | `internal/mcp/runtime/project_context.go:84:12` |
| `util.WriteFileWithDirs` | `internal/mcp/runtime/project_context.go:109:12` |
| `util.WriteFileWithDirs` | `internal/mcp/runtime/project_context.go:126:12` |
| `m.moduleList.Index` | `internal/ui/cli/ui_panels.go:30:9` |
| `s.db.Exec` | `internal/mcp/tools/overlays/handler.go:98:14` |
| `s.db.Exec` | `internal/mcp/tools/overlays/handler.go:195:12` |
| `a.codeParser.IsSupportedPath` | `internal/core/app/analyzer.go:27:7` |
| `a.codeParser.IsTestFile` | `internal/core/app/analyzer.go:31:25` |
| `a.unresolvedMu.Lock` | `internal/core/app/analyzer.go:46:4` |
| `a.unresolvedMu.Unlock` | `internal/core/app/analyzer.go:48:4` |
| `a.unusedMu.Lock` | `internal/core/app/analyzer.go:49:4` |
| `a.unusedMu.Unlock` | `internal/core/app/analyzer.go:51:4` |
| `a.Graph.DetectCycles` | `internal/core/app/analyzer.go:60:12` |
| `a.Graph.ModuleCount` | `internal/core/app/analyzer.go:73:29` |
| `a.Graph.ModuleCount` | `internal/core/app/analyzer.go:77:19` |
| `a.Graph.FileCount` | `internal/core/app/analyzer.go:78:19` |
| `a.SecretCount` | `internal/core/app/analyzer.go:79:19` |
| `resolver.NewResolver` | `internal/core/app/analyzer.go:89:10` |
| `resolver.NewResolver` | `internal/core/app/analyzer.go:100:10` |
| `res.WithBridgeResolutionConfig` | `internal/core/app/analyzer.go:101:3` |
| `a.resolverBridgeConfig` | `internal/core/app/analyzer.go:101:34` |
| `res.WithExplicitBridges` | `internal/core/app/analyzer.go:102:3` |
| `a.loadResolverBridges` | `internal/core/app/analyzer.go:102:27` |
| `resolver.NewResolver` | `internal/core/app/analyzer.go:107:10` |
| `res.WithBridgeResolutionConfig` | `internal/core/app/analyzer.go:108:3` |
| `a.resolverBridgeConfig` | `internal/core/app/analyzer.go:108:34` |
| `res.WithExplicitBridges` | `internal/core/app/analyzer.go:109:3` |
| `a.loadResolverBridges` | `internal/core/app/analyzer.go:109:27` |
| `res.WithBridgeResolutionConfig` | `internal/core/app/analyzer.go:113:2` |
| `a.resolverBridgeConfig` | `internal/core/app/analyzer.go:113:33` |
| `res.WithExplicitBridges` | `internal/core/app/analyzer.go:114:2` |
| `a.loadResolverBridges` | `internal/core/app/analyzer.go:114:26` |
| `helpers.UniqueScanRoots` | `internal/core/app/analyzer.go:143:46` |
| `a.newResolver` | `internal/core/app/analyzer.go:162:9` |
| `func() { _ = res.Close() }` | `internal/core/app/analyzer.go:163:8` |
| `res.Close` | `internal/core/app/analyzer.go:163:21` |
| `a.newResolver` | `internal/core/app/analyzer.go:170:9` |
| `func() { _ = res.Close() }` | `internal/core/app/analyzer.go:171:8` |
| `res.Close` | `internal/core/app/analyzer.go:171:21` |
| `a.cachedUnresolved` | `internal/core/app/analyzer.go:177:10` |
| `a.newResolver` | `internal/core/app/analyzer.go:185:9` |
| `func() { _ = res.Close() }` | `internal/core/app/analyzer.go:186:8` |
| `res.Close` | `internal/core/app/analyzer.go:186:21` |
| `a.unresolvedMu.Lock` | `internal/core/app/analyzer.go:189:2` |
| `a.Graph.GetFile` | `internal/core/app/analyzer.go:191:15` |
| `a.unresolvedMu.Unlock` | `internal/core/app/analyzer.go:201:2` |
| `a.cachedUnresolved` | `internal/core/app/analyzer.go:203:9` |
| `a.newResolver` | `internal/core/app/analyzer.go:207:9` |
| `func() { _ = res.Close() }` | `internal/core/app/analyzer.go:208:8` |
| `res.Close` | `internal/core/app/analyzer.go:208:21` |
| `a.currentGraphPaths` | `internal/core/app/analyzer.go:213:11` |
| `a.newResolver` | `internal/core/app/analyzer.go:214:9` |
| `func() { _ = res.Close() }` | `internal/core/app/analyzer.go:215:8` |
| `res.Close` | `internal/core/app/analyzer.go:215:21` |
| `res.FindUnusedImports` | `internal/core/app/analyzer.go:216:12` |
| `a.cachedUnused` | `internal/core/app/analyzer.go:223:10` |
| `a.newResolver` | `internal/core/app/analyzer.go:231:9` |
| `func() { _ = res.Close() }` | `internal/core/app/analyzer.go:232:8` |
| `res.Close` | `internal/core/app/analyzer.go:232:21` |
| `res.FindUnusedImports` | `internal/core/app/analyzer.go:233:13` |
| `a.unusedMu.Lock` | `internal/core/app/analyzer.go:235:2` |
| `a.Graph.GetFile` | `internal/core/app/analyzer.go:237:15` |
| `a.unusedMu.Unlock` | `internal/core/app/analyzer.go:247:2` |
| `a.cachedUnused` | `internal/core/app/analyzer.go:249:9` |
| `m.Save` | `internal/ui/cli/grammars.go:109:12` |
| `m.Save` | `internal/ui/cli/grammars.go:144:12` |
| `util.NormalizePatternPath` | `internal/engine/graph/architecture.go:71:15` |
| `g.mu.RLock` | `internal/engine/graph/architecture.go:104:2` |
| `g.mu.RUnlock` | `internal/engine/graph/architecture.go:105:8` |
| `util.SortedStringKeys` | `internal/engine/graph/architecture.go:112:14` |
| `util.SortedStringKeys` | `internal/engine/graph/architecture.go:126:13` |
| `util.NormalizePatternPath` | `internal/engine/graph/architecture.go:164:16` |
| `util.NormalizePatternPath` | `internal/engine/graph/architecture.go:166:13` |
| `util.HasPathPrefix` | `internal/engine/graph/architecture.go:203:5` |
| `util.HasPathPrefix` | `internal/engine/graph/architecture.go:206:29` |
| `a.Graph.GetAllFiles` | `internal/core/app/caches.go:7:20` |
| `a.unresolvedMu.Lock` | `internal/core/app/caches.go:13:2` |
| `a.unresolvedMu.Unlock` | `internal/core/app/caches.go:15:2` |
| `a.Graph.GetAllFiles` | `internal/core/app/caches.go:31:20` |
| `a.unusedMu.Lock` | `internal/core/app/caches.go:37:2` |
| `a.unusedMu.Unlock` | `internal/core/app/caches.go:39:2` |
| `a.Graph.GetAllFiles` | `internal/core/app/caches.go:54:11` |
| `a.Graph.DetectCycles` | `internal/core/app/output.go:24:19` |
| `a.Graph.ModuleCount` | `internal/core/app/output.go:26:19` |
| `a.Graph.FileCount` | `internal/core/app/output.go:27:19` |
| `a.SecretCount` | `internal/core/app/output.go:28:19` |
| `helpers.ArchitectureModelFromConfig` | `internal/core/app/output.go:51:15` |
| `helpers.WriteArtifact` | `internal/core/app/output.go:71:13` |
| `helpers.WriteArtifact` | `internal/core/app/output.go:115:13` |
| `a.Config.Output.MermaidEnabled` | `internal/core/app/output.go:120:42` |
| `a.Config.Output.PlantUMLEnabled` | `internal/core/app/output.go:121:44` |
| `a.Config.Output.Report.IncludeMermaidEnabled` | `internal/core/app/output.go:122:31` |
| `a.Config.Output.MermaidEnabled` | `internal/core/app/output.go:122:81` |
| `a.Config.Output.MermaidEnabled` | `internal/core/app/output.go:128:7` |
| `a.Config.Output.PlantUMLEnabled` | `internal/core/app/output.go:132:7` |
| `report.NewMermaidGenerator` | `internal/core/app/output.go:139:17` |
| `mermaidGen.SetModuleMetrics` | `internal/core/app/output.go:140:3` |
| `mermaidGen.SetComplexityHotspots` | `internal/core/app/output.go:141:3` |
| `mode.Suffix` | `internal/core/app/output.go:145:59` |
| `helpers.DiagramOutputPath` | `internal/core/app/output.go:149:16` |
| `helpers.WriteArtifact` | `internal/core/app/output.go:150:15` |
| `mode.Suffix` | `internal/core/app/output.go:164:60` |
| `helpers.DiagramOutputPath` | `internal/core/app/output.go:168:16` |
| `helpers.WriteArtifact` | `internal/core/app/output.go:169:15` |
| `a.resolveOutputRoot` | `internal/core/app/output.go:195:16` |
| `report.NewMarkdownGenerator().Generate` | `internal/core/app/output.go:200:14` |
| `report.NewMarkdownGenerator` | `internal/core/app/output.go:200:14` |
| `a.Graph.ModuleCount` | `internal/core/app/output.go:201:21` |
| `a.Graph.FileCount` | `internal/core/app/output.go:202:21` |
| `a.Config.Output.Report.IncludeMermaidEnabled` | `internal/core/app/output.go:217:25` |
| `helpers.WriteArtifact` | `internal/core/app/output.go:223:13` |
| `sample.WriteRune` | `internal/core/config/helpers/validators.go:69:4` |
| `sample.WriteRune` | `internal/core/config/helpers/validators.go:75:4` |
| `sample.WriteRune` | `internal/core/config/helpers/validators.go:77:4` |
| `db.Exec` | `internal/engine/graph/schema.go:12:12` |
| `msg.String` | `internal/ui/cli/ui_actions.go:14:9` |
| `m.issueList.Update` | `internal/ui/cli/ui_actions.go:31:22` |
| `msg.String` | `internal/ui/cli/ui_actions.go:35:9` |
| `m.moduleList.Update` | `internal/ui/cli/ui_actions.go:70:22` |
| `m.moduleList.Index` | `internal/ui/cli/ui_actions.go:78:9` |
| `err.Error` | `internal/ui/cli/ui_actions.go:84:24` |
| `successStyle.Render` | `internal/ui/cli/runtime.go:58:50` |
| `err.Error` | `internal/ui/cli/runtime.go:69:27` |
| `err.Error` | `internal/ui/cli/runtime.go:77:27` |
| `analysis.RunScan` | `internal/ui/cli/runtime.go:161:15` |
| `config.NewWatcher` | `internal/ui/cli/runtime.go:241:19` |
| `analysis.UpdateConfig` | `internal/ui/cli/runtime.go:243:13` |
| `configWatcher.Start` | `internal/ui/cli/runtime.go:247:12` |
| `configWatcher.Stop` | `internal/ui/cli/runtime.go:250:9` |
| `analysis.WatchService` | `internal/ui/cli/runtime.go:253:11` |
| `err.Error` | `internal/ui/cli/runtime.go:283:28` |
| `err.Error` | `internal/ui/cli/runtime.go:293:28` |
| `analysis.QueryService` | `internal/ui/cli/runtime.go:312:9` |
| `err.Error` | `internal/ui/cli/runtime.go:323:28` |
| `err.Error` | `internal/ui/cli/runtime.go:339:28` |
| `err.Error` | `internal/ui/cli/runtime.go:344:28` |
| `err.Error` | `internal/ui/cli/runtime.go:356:28` |
| `err.Error` | `internal/ui/cli/runtime.go:361:28` |
| `err.Error` | `internal/ui/cli/runtime.go:380:28` |
| `config.Load` | `internal/ui/cli/runtime.go:399:15` |
| `config.Load` | `internal/ui/cli/runtime.go:413:19` |
| `config.ResolveRelative` | `internal/ui/cli/runtime.go:648:19` |
| `config.ResolveRelative` | `internal/ui/cli/runtime.go:656:34` |
| `config.ResolveRelative` | `internal/ui/cli/runtime.go:658:41` |
| `analysis.RunScan` | `internal/ui/cli/runtime.go:705:15` |
| `config.NewWatcher` | `internal/ui/cli/runtime.go:725:19` |
| `analysis.UpdateConfig` | `internal/ui/cli/runtime.go:727:13` |
| `configWatcher.Start` | `internal/ui/cli/runtime.go:731:12` |
| `configWatcher.Stop` | `internal/ui/cli/runtime.go:734:9` |
| `s.graph.Modules` | `internal/data/query/service.go:36:13` |
| `s.graph.GetImports` | `internal/data/query/service.go:37:13` |
| `s.graph.GetImports` | `internal/data/query/service.go:84:13` |
| `s.graph.GetImports` | `internal/data/query/service.go:205:13` |
| `s.graph.Modules` | `internal/data/query/service.go:207:13` |
| `w.cfg.batchSize` | `internal/engine/graph/writer.go:101:35` |
| `w.cfg.flushInterval` | `internal/engine/graph/writer.go:102:27` |
| `w.writeBatch` | `internal/engine/graph/writer.go:109:10` |
| `w.cfg.batchSize` | `internal/engine/graph/writer.go:118:21` |
| `w.cfg.flushInterval` | `internal/engine/graph/writer.go:121:18` |
| `tx.Rollback` | `internal/engine/graph/writer.go:160:8` |
| `tx.Commit` | `internal/engine/graph/writer.go:164:12` |
| `w.writeBatch` | `internal/engine/graph/writer.go:178:11` |
| `b.WriteString` | `internal/ui/report/surgical.go:131:2` |
| `b.WriteString` | `internal/ui/report/surgical.go:132:2` |
| `b.WriteString` | `internal/ui/report/surgical.go:133:2` |
| `contracts.OperationID` | `internal/mcp/openapi/converter.go:36:10` |
| `a.detector.Detect` | `internal/engine/secrets/adapter.go:27:9` |
| `a.detector.Detect` | `internal/engine/secrets/adapter.go:32:10` |
| `util.SortedStringKeys` | `internal/engine/parser/parser.go:145:9` |
| `util.SortedStringKeys` | `internal/engine/parser/parser.go:149:9` |
| `parsed.UTC` | `internal/mcp/tools/query/handler.go:53:10` |
| `parsed.UTC` | `internal/mcp/tools/query/handler.go:56:10` |
| `r.mu.Lock` | `internal/shared/util/limiter_registry.go:39:2` |
| `r.mu.Unlock` | `internal/shared/util/limiter_registry.go:40:8` |
| `r.mu.Lock` | `internal/shared/util/limiter_registry.go:63:2` |
| `r.mu.Unlock` | `internal/shared/util/limiter_registry.go:64:8` |
| `tx.Exec` | `internal/data/history/schema.go:139:16` |
| `tx.Rollback` | `internal/data/history/schema.go:140:8` |
| `tx.Exec` | `internal/data/history/schema.go:143:16` |
| `tx.Rollback` | `internal/data/history/schema.go:144:8` |
| `s.app.Graph.FileCount` | `internal/core/app/health.go:35:73` |
| `s.app.Graph.ModuleCount` | `internal/core/app/health.go:35:98` |
| `gen.GenerateArchitecture` | `internal/core/app/helpers/diagrams.go:99:10` |
| `gen.GenerateComponent` | `internal/core/app/helpers/diagrams.go:101:10` |
| `gen.GenerateFlow` | `internal/core/app/helpers/diagrams.go:103:10` |
| `gen.Generate` | `internal/core/app/helpers/diagrams.go:105:10` |
| `gen.GenerateArchitecture` | `internal/core/app/helpers/diagrams.go:119:10` |
| `gen.GenerateComponent` | `internal/core/app/helpers/diagrams.go:121:10` |
| `gen.GenerateFlow` | `internal/core/app/helpers/diagrams.go:123:10` |
| `gen.Generate` | `internal/core/app/helpers/diagrams.go:125:10` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:46:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:47:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:48:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:49:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:50:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:51:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:52:2` |
| `util.SortedStringKeys` | `internal/ui/report/formats/plantuml.go:56:17` |
| `util.SortedStringKeys` | `internal/ui/report/formats/plantuml.go:70:19` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:89:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:91:5` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:93:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:97:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:101:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:106:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:109:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:113:2` |
| `util.SortedStringKeys` | `internal/ui/report/formats/plantuml.go:114:23` |
| `util.SortedStringKeys` | `internal/ui/report/formats/plantuml.go:115:14` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:131:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:140:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:144:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:145:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:146:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:147:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:148:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:149:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:150:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:151:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:153:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:156:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:159:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:161:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:162:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:164:2` |
| `b.String` | `internal/ui/report/formats/plantuml.go:165:9` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:174:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:175:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:176:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:177:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:178:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:179:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:180:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:185:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:188:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:196:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:199:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:200:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:201:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:202:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:203:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:204:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:205:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:206:2` |
| `b.String` | `internal/ui/report/formats/plantuml.go:207:9` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:215:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:216:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:217:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:218:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:219:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:220:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:221:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:230:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:232:5` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:234:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:237:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:241:4` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:249:5` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:250:5` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:255:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:277:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:280:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:281:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:282:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:283:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:284:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:286:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:287:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:289:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:290:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:292:2` |
| `b.String` | `internal/ui/report/formats/plantuml.go:293:9` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:309:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:310:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:311:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:312:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:313:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:314:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:315:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:322:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:325:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:327:3` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:330:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:331:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:332:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:333:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:334:2` |
| `b.WriteString` | `internal/ui/report/formats/plantuml.go:335:2` |
| `b.String` | `internal/ui/report/formats/plantuml.go:336:9` |
| `helpers.CompileGlobs` | `internal/core/app/app.go:103:29` |
| `helpers.CompileGlobs` | `internal/core/app/app.go:107:30` |
| `helpers.CompileGlobs` | `internal/core/app/app.go:154:28` |
| `helpers.CompileGlobs` | `internal/core/app/app.go:158:29` |
| `graph.NewLayerRuleEngine` | `internal/core/app/app.go:187:23` |
| `helpers.ArchitectureModelFromConfig` | `internal/core/app/app.go:187:48` |
| `bridge.matchesSource` | `internal/engine/resolver/bridge_scoring.go:41:7` |
| `bridge.matchesReference` | `internal/engine/resolver/bridge_scoring.go:44:6` |
| `r.symbolTable.Lookup` | `internal/engine/resolver/bridge_scoring.go:125:16` |
| `r.symbolTable.Lookup` | `internal/engine/resolver/bridge_scoring.go:129:17` |
| `r.symbolTable.LookupService` | `internal/engine/resolver/bridge_scoring.go:134:24` |
| `r.symbolTable.LookupService` | `internal/engine/resolver/bridge_scoring.go:138:25` |
| `r.isStdlibSymbol` | `internal/engine/resolver/bridge_scoring.go:190:9` |
| `b.WriteString` | `internal/core/app/impact_report.go:12:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:13:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:14:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:16:3` |
| `b.WriteString` | `internal/core/app/impact_report.go:18:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:20:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:22:3` |
| `b.WriteString` | `internal/core/app/impact_report.go:24:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:26:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:28:3` |
| `b.WriteString` | `internal/core/app/impact_report.go:30:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:32:2` |
| `b.WriteString` | `internal/core/app/impact_report.go:34:3` |
| `b.String` | `internal/core/app/impact_report.go:37:9` |
| `g.mu.RLock` | `internal/engine/graph/symbol_table.go:41:2` |
| `g.mu.RUnlock` | `internal/engine/graph/symbol_table.go:42:8` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:50:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:51:2` |
| `m.graph.Modules` | `internal/ui/report/formats/mermaid.go:53:13` |
| `m.graph.GetImports` | `internal/ui/report/formats/mermaid.go:54:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/mermaid.go:55:17` |
| `util.SortedStringKeys` | `internal/ui/report/formats/mermaid.go:69:19` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:89:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:91:5` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:93:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:98:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:102:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:108:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:111:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:115:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:117:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:118:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:119:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:120:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:123:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:125:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:127:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:128:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:129:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:135:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:136:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:137:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:138:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:150:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:151:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:152:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:153:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:157:2` |
| `util.SortedStringKeys` | `internal/ui/report/formats/mermaid.go:162:23` |
| `util.SortedStringKeys` | `internal/ui/report/formats/mermaid.go:163:14` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:178:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:188:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:195:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:198:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:201:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:204:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:206:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:207:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:208:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:209:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:210:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:211:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:212:2` |
| `b.String` | `internal/ui/report/formats/mermaid.go:214:9` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:223:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:224:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:229:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:231:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:232:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:233:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:234:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:235:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:244:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:248:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:249:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:252:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:253:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:254:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:255:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:256:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:257:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:258:2` |
| `b.String` | `internal/ui/report/formats/mermaid.go:259:9` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:266:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:267:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:277:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:279:5` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:281:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:284:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:288:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:306:5` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:307:5` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:312:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:313:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:315:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:316:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:317:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:320:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:321:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:322:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:323:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:326:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:351:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:355:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:356:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:359:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:360:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:361:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:362:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:364:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:366:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:367:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:369:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:371:3` |
| `b.String` | `internal/ui/report/formats/mermaid.go:373:9` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:389:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:390:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:392:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:394:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:396:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:399:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:400:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:401:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:402:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:403:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:411:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:412:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:413:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:414:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:417:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:418:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:419:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:420:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:421:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:422:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:423:2` |
| `b.String` | `internal/ui/report/formats/mermaid.go:424:9` |
| `m.graph.Modules` | `internal/ui/report/formats/mermaid.go:434:13` |
| `m.graph.GetImports` | `internal/ui/report/formats/mermaid.go:435:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/mermaid.go:436:17` |
| `util.SortedStringKeys` | `internal/ui/report/formats/mermaid.go:480:23` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:512:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:513:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:518:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:524:4` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:526:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:528:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:532:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:533:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:534:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:535:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:546:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:554:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:555:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:558:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:559:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:560:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:561:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:563:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:565:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:566:2` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:568:3` |
| `b.WriteString` | `internal/ui/report/formats/mermaid.go:570:3` |
| `b.String` | `internal/ui/report/formats/mermaid.go:573:9` |
| `util.NormalizePatternPath` | `internal/ui/report/formats/mermaid.go:591:17` |
| `util.NormalizePatternPath` | `internal/ui/report/formats/mermaid.go:593:17` |
| `util.NormalizePatternPath` | `internal/ui/report/formats/mermaid.go:599:16` |
| `util.HasPathPrefix` | `internal/ui/report/formats/mermaid.go:638:5` |
| `util.HasPathPrefix` | `internal/ui/report/formats/mermaid.go:641:29` |
| `a.Graph.GetModule` | `internal/core/app/reporting.go:17:14` |
| `a.Graph.GetModule` | `internal/core/app/reporting.go:20:14` |
| `b.WriteString` | `internal/core/app/reporting.go:30:2` |
| `b.WriteString` | `internal/core/app/reporting.go:32:3` |
| `b.WriteString` | `internal/core/app/reporting.go:33:3` |
| `b.WriteString` | `internal/core/app/reporting.go:35:4` |
| `b.String` | `internal/core/app/reporting.go:39:27` |
| `a.Graph.GetAllFiles` | `internal/core/app/reporting.go:56:23` |
| `a.Graph.GetAllFiles` | `internal/core/app/reporting.go:64:23` |
| `observability.Tracer.Start` | `internal/engine/resolver/unused_imports.go:11:13` |
| `span.End` | `internal/engine/resolver/unused_imports.go:12:8` |
| `r.graph.GetFile` | `internal/engine/resolver/unused_imports.go:23:15` |
| `sp.SetLanguage` | `internal/engine/parser/pool.go:44:4` |
| `sp.SetLanguage` | `internal/engine/parser/pool.go:56:2` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:24:2` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:29:4` |
| `buf.String` | `internal/ui/report/formats/tsv.go:34:9` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:40:2` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:42:3` |
| `buf.String` | `internal/ui/report/formats/tsv.go:54:9` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:60:2` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:62:3` |
| `buf.String` | `internal/ui/report/formats/tsv.go:74:9` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:80:2` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:82:3` |
| `buf.String` | `internal/ui/report/formats/tsv.go:93:9` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:99:2` |
| `buf.WriteString` | `internal/ui/report/formats/tsv.go:101:3` |
| `buf.String` | `internal/ui/report/formats/tsv.go:113:9` |
| `contracts.OperationID` | `internal/mcp/openapi/filters.go:26:11` |
| `b.WriteRune` | `internal/ui/report/formats/utils.go:40:4` |
| `b.WriteRune` | `internal/ui/report/formats/utils.go:43:3` |
| `b.String` | `internal/ui/report/formats/utils.go:45:9` |
| `g.Modules` | `internal/ui/report/formats/utils.go:94:13` |
| `g.GetImports` | `internal/ui/report/formats/utils.go:95:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/utils.go:96:17` |
| `util.SortedStringKeys` | `internal/ui/report/formats/utils.go:100:23` |
| `util.SortedStringKeys` | `internal/ui/report/formats/utils.go:105:14` |
| `graph.NewLayerRuleEngine` | `internal/core/app/presentation_service.go:24:15` |
| `helpers.ArchitectureModelFromConfig` | `internal/core/app/presentation_service.go:24:40` |
| `report.NewMermaidGenerator` | `internal/core/app/presentation_service.go:44:17` |
| `mermaidGen.SetModuleMetrics` | `internal/core/app/presentation_service.go:45:3` |
| `mermaidGen.SetComplexityHotspots` | `internal/core/app/presentation_service.go:46:3` |
| `helpers.ArchitectureModelFromConfig` | `internal/core/app/presentation_service.go:47:65` |
| `report.NewMarkdownGenerator().Generate` | `internal/core/app/presentation_service.go:57:13` |
| `report.NewMarkdownGenerator` | `internal/core/app/presentation_service.go:57:13` |
| `helpers.WriteArtifact` | `internal/core/app/presentation_service.go:97:13` |
| `helpers.MetricLeaders` | `internal/core/app/presentation_service.go:170:15` |
| `helpers.MetricLeaders` | `internal/core/app/presentation_service.go:171:15` |
| `helpers.MetricLeaders` | `internal/core/app/presentation_service.go:172:16` |
| `config.ResolvePaths` | `internal/core/app/symbol_store.go:22:27` |
| `a.currentGraphPaths` | `internal/core/app/symbol_store.go:59:36` |
| `f.Close` | `internal/engine/parser/manifest.go:47:8` |
| `f.Close` | `internal/engine/parser/manifest.go:79:8` |
| `s.mu.Lock` | `internal/mcp/transport/stdio.go:47:2` |
| `s.mu.Unlock` | `internal/mcp/transport/stdio.go:49:3` |
| `s.mu.Unlock` | `internal/mcp/transport/stdio.go:54:2` |
| `s.mu.Lock` | `internal/mcp/transport/stdio.go:57:3` |
| `s.mu.Unlock` | `internal/mcp/transport/stdio.go:59:3` |
| `s.mu.Lock` | `internal/mcp/transport/stdio.go:63:2` |
| `s.mu.Unlock` | `internal/mcp/transport/stdio.go:65:2` |
| `encoder.Encode` | `internal/mcp/transport/stdio.go:144:15` |
| `writer.Flush` | `internal/mcp/transport/stdio.go:147:15` |
| `encoder.Encode` | `internal/mcp/transport/stdio.go:178:13` |
| `writer.Flush` | `internal/mcp/transport/stdio.go:181:13` |
| `schema.BuildToolDefinitions` | `internal/mcp/transport/stdio.go:247:15` |
| `encoder.Encode` | `internal/mcp/transport/stdio.go:295:12` |
| `writer.Flush` | `internal/mcp/transport/stdio.go:298:12` |
| `node.Kind` | `internal/engine/parser/universal.go:180:11` |
| `node.ChildCount` | `internal/engine/parser/universal.go:185:26` |
| `node.Child` | `internal/engine/parser/universal.go:186:11` |
| `ch.Kind` | `internal/engine/parser/universal.go:187:21` |
| `node.ChildCount` | `internal/engine/parser/universal.go:211:24` |
| `node.Child` | `internal/engine/parser/universal.go:212:9` |
| `ch.Kind` | `internal/engine/parser/universal.go:216:10` |
| `ch.Kind` | `internal/engine/parser/universal.go:239:10` |
| `node.StartPosition` | `internal/engine/parser/universal.go:259:14` |
| `node.ChildCount` | `internal/engine/parser/universal.go:261:24` |
| `node.Child` | `internal/engine/parser/universal.go:262:9` |
| `ch.Kind` | `internal/engine/parser/universal.go:266:10` |
| `node.StartPosition` | `internal/engine/parser/universal.go:289:14` |
| `node.ChildCount` | `internal/engine/parser/universal.go:292:24` |
| `node.Child` | `internal/engine/parser/universal.go:293:9` |
| `ch.Kind` | `internal/engine/parser/universal.go:297:10` |
| `node.ChildCount` | `internal/engine/parser/universal.go:320:24` |
| `node.Child` | `internal/engine/parser/universal.go:321:9` |
| `ch.Kind` | `internal/engine/parser/universal.go:325:10` |
| `node.Kind` | `internal/engine/parser/universal.go:344:10` |
| `node.StartPosition` | `internal/engine/parser/universal.go:358:17` |
| `node.StartPosition` | `internal/engine/parser/universal.go:359:17` |
| `node.ChildCount` | `internal/engine/parser/universal.go:395:24` |
| `node.Child` | `internal/engine/parser/universal.go:396:17` |
| `node.ChildByFieldName` | `internal/engine/parser/universal.go:409:11` |
| `node.ChildByFieldName` | `internal/engine/parser/universal.go:416:13` |
| `node.ChildByFieldName` | `internal/engine/parser/universal.go:421:12` |
| `node.ChildCount` | `internal/engine/parser/universal.go:428:24` |
| `node.Child` | `internal/engine/parser/universal.go:429:12` |
| `child.Kind` | `internal/engine/parser/universal.go:433:11` |
| `node.Kind` | `internal/engine/parser/universal.go:444:10` |
| `node.ChildCount` | `internal/engine/parser/universal.go:445:5` |
| `node.StartByte` | `internal/engine/parser/universal.go:459:11` |
| `node.EndByte` | `internal/engine/parser/universal.go:460:9` |
| `node.StartPosition` | `internal/engine/parser/universal.go:630:15` |
| `node.ChildByFieldName` | `internal/engine/parser/universal.go:642:15` |
| `node.ChildByFieldName` | `internal/engine/parser/universal.go:644:15` |
| `node.ChildByFieldName` | `internal/engine/parser/universal.go:647:15` |
| `n.Kind` | `internal/engine/parser/universal.go:658:10` |
| `n.ChildCount` | `internal/engine/parser/universal.go:666:25` |
| `n.Child` | `internal/engine/parser/universal.go:667:9` |
| `ch.Kind` | `internal/engine/parser/universal.go:678:11` |
| `node.ChildCount` | `internal/engine/parser/universal.go:692:24` |
| `node.Child` | `internal/engine/parser/universal.go:693:9` |
| `ch.Kind` | `internal/engine/parser/universal.go:697:10` |
| `n.Kind` | `internal/engine/parser/universal.go:713:11` |
| `n.ChildCount` | `internal/engine/parser/universal.go:731:25` |
| `n.Child` | `internal/engine/parser/universal.go:732:9` |
| `util.NewLimiterRegistry` | `internal/mcp/transport/sse.go:50:22` |
| `util.NewLimiterRegistry` | `internal/mcp/transport/sse.go:51:25` |
| `mux.HandleFunc` | `internal/mcp/transport/sse.go:61:2` |
| `mux.HandleFunc` | `internal/mcp/transport/sse.go:62:2` |
| `util.GetClientIP` | `internal/mcp/transport/sse.go:96:9` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:98:4` |
| `w.Header` | `internal/mcp/transport/sse.go:98:4` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:104:2` |
| `w.Header` | `internal/mcp/transport/sse.go:104:2` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:105:2` |
| `w.Header` | `internal/mcp/transport/sse.go:105:2` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:106:2` |
| `w.Header` | `internal/mcp/transport/sse.go:106:2` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:107:2` |
| `w.Header` | `internal/mcp/transport/sse.go:107:2` |
| `s.sessionsMu.Lock` | `internal/mcp/transport/sse.go:122:2` |
| `s.sessionsMu.Unlock` | `internal/mcp/transport/sse.go:124:2` |
| `s.sessionsMu.Lock` | `internal/mcp/transport/sse.go:127:3` |
| `s.sessionsMu.Unlock` | `internal/mcp/transport/sse.go:129:3` |
| `flusher.Flush` | `internal/mcp/transport/sse.go:134:2` |
| `flusher.Flush` | `internal/mcp/transport/sse.go:144:4` |
| `flusher.Flush` | `internal/mcp/transport/sse.go:150:4` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:157:2` |
| `w.Header` | `internal/mcp/transport/sse.go:157:2` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:158:2` |
| `w.Header` | `internal/mcp/transport/sse.go:158:2` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:159:2` |
| `w.Header` | `internal/mcp/transport/sse.go:159:2` |
| `w.WriteHeader` | `internal/mcp/transport/sse.go:162:3` |
| `util.GetClientIP` | `internal/mcp/transport/sse.go:187:9` |
| `w.Header().Set` | `internal/mcp/transport/sse.go:189:4` |
| `w.Header` | `internal/mcp/transport/sse.go:189:4` |
| `w.WriteHeader` | `internal/mcp/transport/sse.go:209:2` |
| `s.handler` | `internal/mcp/transport/sse.go:222:18` |
| `schema.BuildToolDefinitions` | `internal/mcp/transport/sse.go:271:15` |
| `s.handler` | `internal/mcp/transport/sse.go:287:18` |
| `g.mu.RLock` | `internal/engine/graph/impact.go:32:2` |
| `g.mu.RUnlock` | `internal/engine/graph/impact.go:33:8` |
| `g.analyzeImpactForModule` | `internal/engine/graph/impact.go:44:11` |
| `g.analyzeImpactForModule` | `internal/engine/graph/impact.go:49:9` |
| `engine.Walk` | `internal/engine/parser/dynamic_extractor.go:43:2` |
| `node.ChildCount` | `internal/engine/parser/dynamic_extractor.go:72:24` |
| `node.Child` | `internal/engine/parser/dynamic_extractor.go:73:12` |
| `child.Kind` | `internal/engine/parser/dynamic_extractor.go:74:6` |
| `child.Kind` | `internal/engine/parser/dynamic_extractor.go:74:38` |
| `r.symbolTable.Lookup` | `internal/engine/resolver/probabilistic.go:15:16` |
| `r.symbolTable.Lookup` | `internal/engine/resolver/probabilistic.go:19:17` |
| `r.symbolTable.LookupService` | `internal/engine/resolver/probabilistic.go:24:24` |
| `r.symbolTable.LookupService` | `internal/engine/resolver/probabilistic.go:28:25` |
| `observability.Tracer.Start` | `internal/core/app/scanner.go:21:15` |
| `span.End` | `internal/core/app/scanner.go:22:8` |
| `helpers.UniqueScanRoots` | `internal/core/app/scanner.go:24:16` |
| `resolver.NewGoResolver` | `internal/core/app/scanner.go:27:8` |
| `r.FindGoMod` | `internal/core/app/scanner.go:28:13` |
| `r.GetModuleRoot` | `internal/core/app/scanner.go:29:36` |
| `helpers.UniqueScanRoots` | `internal/core/app/scanner.go:34:15` |
| `a.processFileWithUpserter` | `internal/core/app/scanner.go:52:13` |
| `g.Match` | `internal/core/app/scanner.go:98:9` |
| `a.codeParser.IsSupportedPath` | `internal/core/app/scanner.go:105:8` |
| `a.codeParser.IsTestFile` | `internal/core/app/scanner.go:110:26` |
| `g.Match` | `internal/core/app/scanner.go:115:8` |
| `a.processFileWithUpserter` | `internal/core/app/scanner.go:132:9` |
| `a.Graph.GetFile` | `internal/core/app/scanner.go:153:21` |
| `g.Match` | `internal/core/app/scanner.go:222:6` |
| `g.Match` | `internal/core/app/scanner.go:231:7` |
| `db.Close` | `internal/data/history/store.go:65:7` |
| `db.Close` | `internal/data/history/store.go:69:7` |
| `s.mu.Lock` | `internal/data/history/store.go:84:2` |
| `s.mu.Unlock` | `internal/data/history/store.go:85:8` |
| `s.withRetry` | `internal/data/history/store.go:128:9` |
| `s.mu.Lock` | `internal/data/history/store.go:153:2` |
| `s.mu.Unlock` | `internal/data/history/store.go:154:8` |
| `s.withRetry` | `internal/data/history/store.go:178:9` |
| `err.Error` | `internal/data/history/store.go:259:25` |
| `err.Error` | `internal/data/history/store.go:274:25` |
| `m.issueList.Update` | `internal/ui/cli/ui.go:160:22` |
| `m.moduleList.Update` | `internal/ui/cli/ui.go:162:23` |
| `successStyle.Render` | `internal/ui/cli/ui.go:173:13` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:47:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:48:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:49:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:50:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:51:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:52:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:53:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:54:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:89:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:90:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:91:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:92:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:93:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:117:4` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:120:5` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:122:5` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:126:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:129:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:130:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:132:3` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:134:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:144:5` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:146:5` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:148:5` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:154:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:155:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:156:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:157:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:158:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:159:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:160:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:161:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:162:2` |
| `buf.WriteString` | `internal/ui/report/formats/dot.go:164:2` |
| `buf.String` | `internal/ui/report/formats/dot.go:166:9` |
| `info.IsDir` | `internal/engine/parser/loader.go:46:57` |
| `info.IsDir` | `internal/engine/parser/loader.go:52:56` |
| `util.SortedStringKeys` | `internal/engine/parser/loader.go:75:25` |
| `registry.New` | `internal/mcp/runtime/server.go:86:9` |
| `s.projectMu.RLock` | `internal/mcp/runtime/server.go:120:2` |
| `s.projectMu.RUnlock` | `internal/mcp/runtime/server.go:121:8` |
| `s.mu.Lock` | `internal/mcp/runtime/server.go:130:2` |
| `s.mu.Unlock` | `internal/mcp/runtime/server.go:132:3` |
| `s.mu.Unlock` | `internal/mcp/runtime/server.go:137:2` |
| `s.mu.Lock` | `internal/mcp/runtime/server.go:147:2` |
| `s.mu.Unlock` | `internal/mcp/runtime/server.go:149:2` |
| `s.mu.Lock` | `internal/mcp/runtime/server.go:155:2` |
| `s.mu.Unlock` | `internal/mcp/runtime/server.go:156:8` |
| `s.history.Close` | `internal/mcp/runtime/server.go:160:11` |
| `s.history.Close` | `internal/mcp/runtime/server.go:166:13` |
| `s.projectMu.RLock` | `internal/mcp/runtime/server.go:182:2` |
| `s.projectMu.RUnlock` | `internal/mcp/runtime/server.go:184:2` |
| `s.projectMu.RLock` | `internal/mcp/runtime/server.go:193:2` |
| `s.projectMu.RUnlock` | `internal/mcp/runtime/server.go:195:2` |
| `s.projectMu.RLock` | `internal/mcp/runtime/server.go:208:2` |
| `s.projectMu.RUnlock` | `internal/mcp/runtime/server.go:210:2` |
| `config.ResolvePaths` | `internal/mcp/runtime/server.go:232:16` |
| `config.ResolveRelative` | `internal/mcp/runtime/server.go:234:26` |
| `s.watchMu.Lock` | `internal/mcp/runtime/server.go:254:2` |
| `s.watchMu.Unlock` | `internal/mcp/runtime/server.go:256:3` |
| `s.watchMu.Unlock` | `internal/mcp/runtime/server.go:263:2` |
| `s.watchMu.Lock` | `internal/mcp/runtime/server.go:268:4` |
| `s.watchMu.Unlock` | `internal/mcp/runtime/server.go:270:4` |
| `s.registry.HandlerFor` | `internal/mcp/runtime/server.go:279:14` |
| `s.registry.HandlerFor` | `internal/mcp/runtime/server.go:304:17` |
| `s.projectMu.RLock` | `internal/mcp/runtime/server.go:324:2` |
| `s.projectMu.RUnlock` | `internal/mcp/runtime/server.go:326:2` |
| `s.busyMu.Lock` | `internal/mcp/runtime/server.go:344:3` |
| `s.busyMu.Unlock` | `internal/mcp/runtime/server.go:346:4` |
| `s.busyMu.Unlock` | `internal/mcp/runtime/server.go:353:3` |
| `s.busyMu.Lock` | `internal/mcp/runtime/server.go:355:4` |
| `s.busyMu.Unlock` | `internal/mcp/runtime/server.go:357:4` |
| `buf.WriteString` | `internal/ui/report/trends.go:13:2` |
| `buf.WriteString` | `internal/ui/report/trends.go:15:3` |
| `g.Modules` | `internal/ui/report/formats/html_interactive.go:29:13` |
| `util.SortedStringKeys` | `internal/ui/report/formats/html_interactive.go:30:17` |
| `util.SortedStringKeys` | `internal/ui/report/formats/html_interactive.go:69:18` |
| `sb.WriteString` | `internal/ui/report/formats/html_interactive.go:88:2` |
| `sb.WriteString` | `internal/ui/report/formats/html_interactive.go:123:2` |
| `resolver.NewGoResolver` | `internal/core/app/gomod.go:42:7` |
| `r.FindGoMod` | `internal/core/app/gomod.go:43:12` |
| `r.GetModuleRoot` | `internal/core/app/gomod.go:52:15` |
| `g.mu.Lock` | `internal/engine/graph/graph.go:77:2` |
| `g.mu.Unlock` | `internal/engine/graph/graph.go:78:8` |
| `g.mu.Lock` | `internal/engine/graph/graph.go:83:2` |
| `g.mu.Unlock` | `internal/engine/graph/graph.go:84:8` |
| `g.mu.Lock` | `internal/engine/graph/graph.go:93:2` |
| `g.mu.Unlock` | `internal/engine/graph/graph.go:94:8` |
| `g.fileCache.Get` | `internal/engine/graph/graph.go:98:18` |
| `g.removeFileLocked` | `internal/engine/graph/graph.go:99:3` |
| `g.fileCache.Put` | `internal/engine/graph/graph.go:102:2` |
| `g.imports` | `internal/engine/graph/graph.go:161:3` |
| `g.importedBy` | `internal/engine/graph/graph.go:166:3` |
| `observability.GraphNodes.Set` | `internal/engine/graph/graph.go:169:2` |
| `observability.GraphEdges.Set` | `internal/engine/graph/graph.go:174:2` |
| `g.mu.Lock` | `internal/engine/graph/graph.go:178:2` |
| `g.mu.Unlock` | `internal/engine/graph/graph.go:179:8` |
| `g.removeFileLocked` | `internal/engine/graph/graph.go:180:2` |
| `g.fileCache.Get` | `internal/engine/graph/graph.go:215:17` |
| `g.imports` | `internal/engine/graph/graph.go:230:7` |
| `g.importedBy` | `internal/engine/graph/graph.go:234:7` |
| `observability.GraphNodes.Set` | `internal/engine/graph/graph.go:253:2` |
| `observability.GraphEdges.Set` | `internal/engine/graph/graph.go:258:2` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:262:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:263:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:272:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:273:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:283:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:284:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:289:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:290:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:303:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:304:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:310:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:311:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:316:2` |
| `g.fileCache.Get` | `internal/engine/graph/graph.go:317:11` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:319:2` |
| `g.mu.Lock` | `internal/engine/graph/graph.go:329:4` |
| `g.fileCache.Put` | `internal/engine/graph/graph.go:330:4` |
| `g.mu.Unlock` | `internal/engine/graph/graph.go:331:4` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:340:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:341:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:353:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:354:8` |
| `g.mu.RLock` | `internal/engine/graph/graph.go:367:2` |
| `g.mu.RUnlock` | `internal/engine/graph/graph.go:368:8` |
| `g.mu.Lock` | `internal/engine/graph/graph.go:469:2` |
| `g.mu.Unlock` | `internal/engine/graph/graph.go:470:8` |
| `g.mu.Lock` | `internal/engine/graph/graph.go:477:2` |
| `g.mu.Unlock` | `internal/engine/graph/graph.go:478:8` |
| `analysis.QueryService` | `internal/ui/cli/run_ui.go:16:13` |
| `analysis.WatchService` | `internal/ui/cli/run_ui.go:17:11` |
| `a.mu.Lock` | `internal/mcp/adapters/adapter.go:37:2` |
| `a.mu.Unlock` | `internal/mcp/adapters/adapter.go:38:8` |
| `a.mu.RLock` | `internal/mcp/adapters/adapter.go:43:2` |
| `a.mu.RUnlock` | `internal/mcp/adapters/adapter.go:44:8` |
| `a.mu.Lock` | `internal/mcp/adapters/adapter.go:53:2` |
| `a.mu.Unlock` | `internal/mcp/adapters/adapter.go:54:8` |
| `a.analysis.RunScan` | `internal/mcp/adapters/adapter.go:60:17` |
| `a.mu.Lock` | `internal/mcp/adapters/adapter.go:87:2` |
| `a.mu.Unlock` | `internal/mcp/adapters/adapter.go:88:8` |
| `a.analysis.RunScan` | `internal/mcp/adapters/adapter.go:93:21` |
| `a.analysis.ListFiles` | `internal/mcp/adapters/adapter.go:98:16` |
| `a.mu.RLock` | `internal/mcp/adapters/adapter.go:116:2` |
| `a.mu.RUnlock` | `internal/mcp/adapters/adapter.go:117:8` |
| `a.analysis.ListFiles` | `internal/mcp/adapters/adapter.go:122:16` |
| `a.mu.RLock` | `internal/mcp/adapters/adapter.go:138:2` |
| `a.mu.RUnlock` | `internal/mcp/adapters/adapter.go:139:8` |
| `a.queryService` | `internal/mcp/adapters/adapter.go:159:9` |
| `a.queryService` | `internal/mcp/adapters/adapter.go:187:9` |
| `a.queryService` | `internal/mcp/adapters/adapter.go:223:9` |
| `a.queryService` | `internal/mcp/adapters/adapter.go:244:9` |
| `a.mu.Lock` | `internal/mcp/adapters/adapter.go:276:2` |
| `a.mu.Unlock` | `internal/mcp/adapters/adapter.go:277:8` |
| `a.mu.Lock` | `internal/mcp/adapters/adapter.go:295:2` |
| `a.mu.Unlock` | `internal/mcp/adapters/adapter.go:296:8` |
| `domainErrors.IsCode` | `internal/mcp/adapters/adapter.go:341:7` |
| `domainErrors.IsCode` | `internal/mcp/adapters/adapter.go:343:7` |
| `domainErrors.IsCode` | `internal/mcp/adapters/adapter.go:345:7` |
| `domainErrors.IsCode` | `internal/mcp/adapters/adapter.go:347:7` |
| `config.ResolvePaths` | `internal/mcp/runtime/bootstrap.go:25:16` |
| `config.ResolveRelative` | `internal/mcp/runtime/bootstrap.go:29:25` |
| `historyStore.Close` | `internal/mcp/runtime/bootstrap.go:43:8` |
| `registry.New` | `internal/mcp/runtime/bootstrap.go:48:9` |
| `historyStore.Close` | `internal/mcp/runtime/bootstrap.go:53:8` |
| `historyStore.Close` | `internal/mcp/runtime/bootstrap.go:60:7` |
| `g.BuildUniversalSymbolTable` | `internal/engine/resolver/resolver.go:88:17` |
| `g.BuildUniversalSymbolTable` | `internal/engine/resolver/resolver.go:111:17` |
| `r.resolveReferenceResult` | `internal/engine/resolver/resolver.go:132:12` |
| `observability.Tracer.Start` | `internal/engine/resolver/resolver.go:137:13` |
| `span.End` | `internal/engine/resolver/resolver.go:141:8` |
| `r.isStdlibSymbol` | `internal/engine/resolver/resolver.go:163:5` |
| `r.isStdlibSymbol` | `internal/engine/resolver/resolver.go:275:9` |
| `db.Close` | `internal/engine/graph/symbol_store.go:50:7` |
| `db.Close` | `internal/engine/graph/symbol_store.go:55:7` |
| `s.db.Begin` | `internal/engine/graph/symbol_store.go:76:13` |
| `s.db.Begin` | `internal/engine/graph/symbol_store.go:131:13` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:137:16` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:138:8` |
| `tx.Commit` | `internal/engine/graph/symbol_store.go:141:13` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:148:7` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:153:8` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:157:8` |
| `tx.Commit` | `internal/engine/graph/symbol_store.go:162:12` |
| `s.db.Begin` | `internal/engine/graph/symbol_store.go:172:13` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:177:7` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:181:7` |
| `tx.Commit` | `internal/engine/graph/symbol_store.go:184:12` |
| `s.db.Begin` | `internal/engine/graph/symbol_store.go:194:13` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:199:7` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:202:15` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:203:7` |
| `tx.Commit` | `internal/engine/graph/symbol_store.go:206:12` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:236:11` |
| `s.db.Begin` | `internal/engine/graph/symbol_store.go:247:13` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:252:16` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:253:8` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:256:16` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:257:8` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:262:8` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:266:8` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:269:16` |
| `tx.Rollback` | `internal/engine/graph/symbol_store.go:270:8` |
| `tx.Commit` | `internal/engine/graph/symbol_store.go:274:12` |
| `s.lookupRows` | `internal/engine/graph/symbol_store.go:288:9` |
| `s.lookupRows` | `internal/engine/graph/symbol_store.go:319:9` |
| `db.Exec` | `internal/engine/graph/symbol_store.go:429:13` |
| `db.Exec` | `internal/engine/graph/symbol_store.go:484:13` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:510:15` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:517:15` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:524:15` |
| `tx.Prepare` | `internal/engine/graph/symbol_store.go:527:15` |
| `stmt.Close` | `internal/engine/graph/symbol_store.go:531:8` |
| `stmt.Exec` | `internal/engine/graph/symbol_store.go:533:16` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:549:15` |
| `tx.Exec` | `internal/engine/graph/symbol_store.go:556:15` |
| `tx.Prepare` | `internal/engine/graph/symbol_store.go:722:15` |
| `stmt.Close` | `internal/engine/graph/symbol_store.go:749:8` |
| `stmt.Exec` | `internal/engine/graph/symbol_store.go:752:16` |
| `node.Kind` | `internal/engine/parser/engine.go:38:31` |
| `node.ChildCount` | `internal/engine/parser/engine.go:43:25` |
| `node.Child` | `internal/engine/parser/engine.go:44:16` |
| `node.StartByte` | `internal/engine/parser/engine.go:53:25` |
| `node.EndByte` | `internal/engine/parser/engine.go:53:42` |
| `node.StartPosition` | `internal/engine/parser/engine.go:59:15` |
| `node.StartPosition` | `internal/engine/parser/engine.go:60:15` |
| `node.ChildCount` | `internal/engine/parser/engine.go:68:24` |
| `node.Child` | `internal/engine/parser/engine.go:69:12` |
| `child.Kind` | `internal/engine/parser/engine.go:70:6` |
| `c.Text` | `internal/engine/parser/engine.go:71:11` |
| `node.Kind` | `internal/engine/parser/engine.go:81:5` |
| `c.Text` | `internal/engine/parser/engine.go:82:53` |
| `node.ChildCount` | `internal/engine/parser/engine.go:85:24` |
| `node.Child` | `internal/engine/parser/engine.go:86:28` |

</details>

## Unused Imports
<details>
<summary>Unused import details</summary>

| Language | Module | Alias | Item | Confidence | Location |
| --- | --- | --- | --- | --- | --- |
| `go` | `circular/internal/core/app` | `` | `` | `medium` | `internal/ui/cli/server_observability.go:9:0` |
| `go` | `circular/internal/shared/version` | `` | `` | `medium` | `internal/core/config/loader.go:4:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/ui/cli/run_ui.go:4:0` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/ui/cli/run_ui.go:5:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/mcp/adapters/adapter.go:5:0` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/mcp/adapters/adapter.go:6:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/mcp/adapters/adapter.go:7:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/adapters/adapter.go:9:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/resolver/resolver.go:6:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/app/helpers/metrics.go:4:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/secrets/git_scanner.go:7:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/ui/report/formats/tsv.go:6:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/core/ports/ports.go:4:0` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/core/ports/ports.go:5:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/ports/ports.go:7:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/core/ports/ports.go:8:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/ui/report/formats/utils.go:4:0` |
| `go` | `circular/internal/shared/version` | `` | `` | `medium` | `internal/core/app/presentation_service.go:7:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/schema/tools.go:3:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/core/app/symbol_store.go:6:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/mcp/transport/stdio.go:5:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/transport/stdio.go:6:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/mcp/transport/sse.go:4:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/transport/sse.go:5:0` |
| `go` | `circular/internal/engine/parser/registry` | `` | `` | `medium` | `internal/engine/parser/dynamic_extractor.go:4:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/engine/resolver/probabilistic.go:4:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/resolver/probabilistic.go:5:0` |
| `go` | `circular/internal/mcp/adapters` | `` | `` | `medium` | `internal/mcp/tools/scan/handler.go:4:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/tools/scan/handler.go:5:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/ui/cli/ui.go:4:0` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/ui/cli/ui.go:5:0` |
| `go` | `github.com/prometheus/client_golang/prometheus` | `` | `` | `medium` | `internal/shared/observability/metrics.go:4:0` |
| `go` | `circular/internal/mcp/adapters` | `` | `` | `medium` | `internal/mcp/tools/graph/handler.go:4:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/tools/graph/handler.go:5:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/mcp/runtime/server.go:5:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/runtime/server.go:7:0` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/ui/report/trends.go:4:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/ui/report/formats/html_interactive.go:6:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/ui/report/formats_bridge.go:4:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/graph/graph.go:5:0` |
| `go` | `runtime` | `` | `` | `medium` | `internal/engine/parser/builder.go:8:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/ui/cli/runtime_factory.go:5:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/ui/cli/runtime_factory.go:6:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/mcp/runtime/allowlist.go:4:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/runtime/allowlist.go:5:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/core/app/app.go:5:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/core/app/app.go:7:0` |
| `go` | `circular/internal/core/watcher` | `` | `` | `medium` | `internal/core/app/app.go:8:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/app/impact_report.go:4:0` |
| `go` | `circular/internal/shared/version` | `` | `` | `medium` | `internal/ui/cli/cli.go:4:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/graph/symbol_table.go:4:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/core/app/reporting.go:4:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/app/reporting.go:6:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/core/app/reporting.go:7:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/ui/report/formats/sequence.go:6:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/ui/report/formats/sequence.go:7:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/ui/report/formats/diagram_modes.go:4:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/core/app/helpers/utils.go:4:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/app/helpers/utils.go:5:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/ui/report/formats/markdown.go:4:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/core/app/service.go:4:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/core/app/service.go:6:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/app/service.go:9:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/core/app/service.go:10:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/secrets/detector.go:5:0` |
| `go` | `circular/internal/mcp/adapters` | `` | `` | `medium` | `internal/mcp/tools/secrets/handler.go:4:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/tools/secrets/handler.go:5:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/resolver/bridge.go:4:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/ui/report/formats/sarif.go:5:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/ui/report/formats/sarif.go:6:0` |
| `go` | `circular/internal/shared/version` | `` | `` | `medium` | `internal/ui/report/formats/sarif.go:7:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/graph/metrics.go:4:0` |
| `go` | `circular/internal/engine/parser/registry` | `` | `` | `medium` | `internal/engine/parser/grammar/verify.go:4:0` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/ui/cli/ui_panels.go:4:0` |
| `go` | `bytes` | `` | `` | `medium` | `internal/data/history/git.go:4:0` |
| `go` | `database/sql` | `` | `` | `medium` | `internal/mcp/tools/overlays/handler.go:7:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/ui/cli/grammars.go:4:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/app/output.go:5:0` |
| `go` | `circular/internal/shared/version` | `` | `` | `medium` | `internal/core/app/output.go:7:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/tools/report/handler.go:4:0` |
| `go` | `database/sql` | `` | `` | `medium` | `internal/engine/graph/schema.go:5:0` |
| `go` | `net/http` | `` | `` | `medium` | `internal/mcp/openapi/loader.go:7:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/ui/cli/runtime.go:6:0` |
| `go` | `syscall` | `` | `` | `medium` | `internal/ui/cli/runtime.go:23:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/graph/writer.go:5:0` |
| `go` | `github.com/getkin/kin-openapi/openapi3` | `` | `` | `medium` | `internal/mcp/openapi/converter.go:10:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/engine/secrets/adapter.go:4:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/engine/secrets/adapter.go:5:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/secrets/adapter.go:6:0` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/data/query/models.go:3:0` |
| `go` | `circular/internal/mcp/adapters` | `` | `` | `medium` | `internal/mcp/tools/query/handler.go:4:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/tools/query/handler.go:5:0` |
| `go` | `database/sql` | `` | `` | `medium` | `internal/data/history/schema.go:4:0` |
| `go` | `circular/internal/core/ports` | `` | `` | `medium` | `internal/core/app/helpers/secrets.go:4:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/core/app/helpers/secrets.go:5:0` |
| `go` | `net/http` | `` | `` | `medium` | `internal/shared/util/net.go:5:0` |
| `go` | `circular/internal/engine/parser` | `` | `` | `medium` | `internal/engine/resolver/heuristics.go:5:0` |
| `go` | `circular/internal/mcp/contracts` | `` | `` | `medium` | `internal/mcp/tools/system/handler.go:4:0` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/core/app/helpers/diagrams.go:4:0` |
| `go` | `circular/internal/engine/graph` | `` | `` | `medium` | `internal/core/app/helpers/diagrams.go:5:0` |

</details>


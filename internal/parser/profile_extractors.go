package parser

import (
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func DefaultExtractorForLanguage(lang string) (Extractor, bool) {
	switch lang {
	case "go":
		return &GoExtractor{}, true
	case "python":
		return &PythonExtractor{}, true
	case "javascript":
		return newJavaScriptProfileExtractor("javascript"), true
	case "typescript":
		return newTypeScriptProfileExtractor("typescript"), true
	case "tsx":
		return newTypeScriptProfileExtractor("tsx"), true
	case "java":
		return &javaProfileExtractor{}, true
	case "rust":
		return &rustProfileExtractor{}, true
	case "html":
		return &htmlProfileExtractor{}, true
	case "css":
		return &cssProfileExtractor{}, true
	case "gomod":
		return &goModProfileExtractor{}, true
	case "gosum":
		return &goSumProfileExtractor{}, true
	default:
		return nil, false
	}
}

type jsProfileExtractor struct {
	language string
}

func newJavaScriptProfileExtractor(language string) *jsProfileExtractor {
	return &jsProfileExtractor{language: language}
}

func (e *jsProfileExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: e.language,
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"import_statement":               e.extractImport,
		"function_declaration":           e.extractFunction,
		"generator_function_declaration": e.extractFunction,
		"class_declaration":              e.extractClass,
		"method_definition":              e.extractMethod,
		"lexical_declaration":            e.extractVarDecl,
		"variable_declaration":           e.extractVarDecl,
		"formal_parameters":              e.extractParams,
		"call_expression":                e.extractCall,
		"new_expression":                 e.extractCall,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *jsProfileExtractor) extractImport(ctx *ExtractionContext, node *sitter.Node) {
	module := trimQuoted(ctx.Text(node.ChildByFieldName("source")))
	if module == "" {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "string" {
				module = trimQuoted(ctx.Text(child))
				break
			}
		}
	}
	if module == "" {
		return
	}

	seen := make(map[string]bool)
	items := make([]string, 0)
	alias := ""
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		switch n.Kind() {
		case "import_specifier":
			raw := strings.TrimSpace(ctx.Text(n))
			if raw == "" {
				return
			}
			parts := splitAndTrim(raw, "as")
			if len(parts) > 0 {
				items = appendUnique(items, seen, parts[0])
				if len(parts) > 1 {
					alias = parts[len(parts)-1]
				}
			}
			return
		case "namespace_import", "identifier":
			if n.Parent() != nil && n.Parent().Kind() == "import_clause" && alias == "" {
				alias = ctx.Text(n)
			}
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}

	clause := node.ChildByFieldName("clause")
	if clause != nil {
		walk(clause)
	} else {
		walk(node)
	}

	ctx.File.Imports = append(ctx.File.Imports, Import{
		Module:    module,
		RawImport: module,
		Alias:     strings.TrimSpace(alias),
		Items:     items,
		Location:  ctx.Location(node),
	})
}

func (e *jsProfileExtractor) extractFunction(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		return
	}

	params := node.ChildByFieldName("parameters")
	if params != nil {
		ctx.AppendLocalIdentifiers(params)
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     KindFunction,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

func (e *jsProfileExtractor) extractClass(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		return
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     KindClass,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

func (e *jsProfileExtractor) extractMethod(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		return
	}

	params := node.ChildByFieldName("parameters")
	if params != nil {
		ctx.AppendLocalIdentifiers(params)
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     KindMethod,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

func (e *jsProfileExtractor) extractVarDecl(ctx *ExtractionContext, node *sitter.Node) {
	ctx.AppendLocalIdentifiers(node)
}

func (e *jsProfileExtractor) extractParams(ctx *ExtractionContext, node *sitter.Node) {
	ctx.AppendLocalIdentifiers(node)
}

func (e *jsProfileExtractor) extractCall(ctx *ExtractionContext, node *sitter.Node) {
	fn := node.ChildByFieldName("function")
	if fn == nil {
		return
	}
	name := normalizeRefName(ctx.Text(fn))
	if name == "" {
		return
	}

	ctx.File.References = append(ctx.File.References, Reference{
		Name:     name,
		Location: ctx.Location(fn),
	})
}

type tsProfileExtractor struct {
	language string
	js       *jsProfileExtractor
}

func newTypeScriptProfileExtractor(language string) *tsProfileExtractor {
	return &tsProfileExtractor{
		language: language,
		js:       newJavaScriptProfileExtractor(language),
	}
}

func (e *tsProfileExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: e.language,
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"import_statement":               e.js.extractImport,
		"function_declaration":           e.js.extractFunction,
		"generator_function_declaration": e.js.extractFunction,
		"class_declaration":              e.js.extractClass,
		"method_definition":              e.js.extractMethod,
		"lexical_declaration":            e.js.extractVarDecl,
		"variable_declaration":           e.js.extractVarDecl,
		"formal_parameters":              e.js.extractParams,
		"call_expression":                e.js.extractCall,
		"new_expression":                 e.js.extractCall,
		"interface_declaration":          e.extractInterface,
		"type_alias_declaration":         e.extractTypeAlias,
		"enum_declaration":               e.extractEnum,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *tsProfileExtractor) extractInterface(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		return
	}
	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     KindInterface,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

func (e *tsProfileExtractor) extractTypeAlias(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		return
	}
	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     KindType,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

func (e *tsProfileExtractor) extractEnum(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		return
	}
	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     KindConstant,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

type javaProfileExtractor struct{}

func (e *javaProfileExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: "java",
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"package_declaration":        e.extractPackage,
		"import_declaration":         e.extractImport,
		"class_declaration":          e.extractClass,
		"interface_declaration":      e.extractInterface,
		"enum_declaration":           e.extractEnum,
		"record_declaration":         e.extractType,
		"method_declaration":         e.extractMethod,
		"constructor_declaration":    e.extractMethod,
		"formal_parameter":           e.extractParameter,
		"spread_parameter":           e.extractParameter,
		"local_variable_declaration": e.extractLocal,
		"catch_formal_parameter":     e.extractParameter,
		"method_invocation":          e.extractReference,
		"object_creation_expression": e.extractReference,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *javaProfileExtractor) extractPackage(ctx *ExtractionContext, node *sitter.Node) {
	ctx.File.PackageName = strings.TrimSpace(strings.TrimPrefix(ctx.Text(node), "package"))
	ctx.File.PackageName = strings.TrimSuffix(ctx.File.PackageName, ";")
}

func (e *javaProfileExtractor) extractImport(ctx *ExtractionContext, node *sitter.Node) {
	raw := strings.TrimSpace(ctx.Text(node))
	raw = strings.TrimPrefix(raw, "import")
	raw = strings.TrimSpace(strings.TrimSuffix(raw, ";"))
	if raw == "" {
		return
	}

	module := strings.TrimPrefix(raw, "static ")
	module = strings.TrimSpace(module)
	ctx.File.Imports = append(ctx.File.Imports, Import{
		Module:    module,
		RawImport: raw,
		Location:  ctx.Location(node),
	})
}

func (e *javaProfileExtractor) extractClass(ctx *ExtractionContext, node *sitter.Node) {
	e.addNamedDef(ctx, node, KindClass)
}

func (e *javaProfileExtractor) extractInterface(ctx *ExtractionContext, node *sitter.Node) {
	e.addNamedDef(ctx, node, KindInterface)
}

func (e *javaProfileExtractor) extractEnum(ctx *ExtractionContext, node *sitter.Node) {
	e.addNamedDef(ctx, node, KindConstant)
}

func (e *javaProfileExtractor) extractType(ctx *ExtractionContext, node *sitter.Node) {
	e.addNamedDef(ctx, node, KindType)
}

func (e *javaProfileExtractor) extractMethod(ctx *ExtractionContext, node *sitter.Node) {
	kind := KindMethod
	if node.Kind() == "constructor_declaration" {
		kind = KindFunction
	}
	e.addNamedDef(ctx, node, kind)
}

func (e *javaProfileExtractor) addNamedDef(ctx *ExtractionContext, node *sitter.Node, kind DefinitionKind) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "identifier" || child.Kind() == "type_identifier" {
				name = strings.TrimSpace(ctx.Text(child))
				break
			}
		}
	}
	if name == "" {
		return
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     kind,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

func (e *javaProfileExtractor) extractParameter(ctx *ExtractionContext, node *sitter.Node) {
	ctx.AppendLocalIdentifiers(node)
}

func (e *javaProfileExtractor) extractLocal(ctx *ExtractionContext, node *sitter.Node) {
	ctx.AppendLocalIdentifiers(node)
}

func (e *javaProfileExtractor) extractReference(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		name = normalizeRefName(ctx.Text(node))
	}
	if name == "" {
		return
	}
	ctx.File.References = append(ctx.File.References, Reference{
		Name:     name,
		Location: ctx.Location(node),
	})
}

type rustProfileExtractor struct{}

func (e *rustProfileExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: "rust",
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"use_declaration":  e.extractUse,
		"function_item":    e.extractFunction,
		"struct_item":      e.extractType,
		"enum_item":        e.extractType,
		"trait_item":       e.extractType,
		"impl_item":        e.extractType,
		"type_item":        e.extractType,
		"const_item":       e.extractConst,
		"let_declaration":  e.extractLocal,
		"parameters":       e.extractLocal,
		"call_expression":  e.extractCall,
		"macro_invocation": e.extractCall,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *rustProfileExtractor) extractUse(ctx *ExtractionContext, node *sitter.Node) {
	raw := strings.TrimSpace(ctx.Text(node))
	raw = strings.TrimPrefix(raw, "use")
	raw = strings.TrimSpace(strings.TrimSuffix(raw, ";"))
	if raw == "" {
		return
	}

	for _, item := range splitAndTrim(raw, ",") {
		entry := strings.Trim(item, "{}")
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		module := entry
		alias := ""
		if strings.Contains(entry, " as ") {
			parts := splitAndTrim(entry, " as ")
			if len(parts) >= 2 {
				module = parts[0]
				alias = parts[len(parts)-1]
			}
		}
		ctx.File.Imports = append(ctx.File.Imports, Import{
			Module:    module,
			RawImport: entry,
			Alias:     alias,
			Location:  ctx.Location(node),
		})
	}
}

func (e *rustProfileExtractor) extractFunction(ctx *ExtractionContext, node *sitter.Node) {
	e.addNamedDef(ctx, node, KindFunction)
}

func (e *rustProfileExtractor) extractType(ctx *ExtractionContext, node *sitter.Node) {
	e.addNamedDef(ctx, node, KindType)
}

func (e *rustProfileExtractor) extractConst(ctx *ExtractionContext, node *sitter.Node) {
	e.addNamedDef(ctx, node, KindConstant)
}

func (e *rustProfileExtractor) addNamedDef(ctx *ExtractionContext, node *sitter.Node, kind DefinitionKind) {
	name := strings.TrimSpace(ctx.Text(node.ChildByFieldName("name")))
	if name == "" {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "identifier" || child.Kind() == "type_identifier" {
				name = strings.TrimSpace(ctx.Text(child))
				break
			}
		}
	}
	if name == "" {
		return
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: name,
		Kind:     kind,
		Exported: isExportedName(name),
		Location: ctx.Location(node),
	})
}

func (e *rustProfileExtractor) extractLocal(ctx *ExtractionContext, node *sitter.Node) {
	ctx.AppendLocalIdentifiers(node)
}

func (e *rustProfileExtractor) extractCall(ctx *ExtractionContext, node *sitter.Node) {
	name := normalizeRefName(ctx.Text(node.ChildByFieldName("function")))
	if name == "" {
		name = normalizeRefName(ctx.Text(node))
	}
	if name == "" {
		return
	}
	ctx.File.References = append(ctx.File.References, Reference{
		Name:     name,
		Location: ctx.Location(node),
	})
}

type htmlProfileExtractor struct{}

func (e *htmlProfileExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: "html",
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"start_tag":        e.extractTag,
		"self_closing_tag": e.extractTag,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *htmlProfileExtractor) extractTag(ctx *ExtractionContext, node *sitter.Node) {
	tag := ""
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "tag_name" {
			tag = strings.ToLower(strings.TrimSpace(ctx.Text(child)))
			break
		}
	}
	if tag == "" {
		return
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		attr := node.Child(i)
		if attr.Kind() != "attribute" {
			continue
		}

		name := ""
		value := ""
		for j := uint(0); j < attr.ChildCount(); j++ {
			part := attr.Child(j)
			switch part.Kind() {
			case "attribute_name":
				name = strings.ToLower(strings.TrimSpace(ctx.Text(part)))
			case "quoted_attribute_value", "attribute_value":
				value = trimQuoted(ctx.Text(part))
			}
		}
		if name == "" {
			continue
		}

		if tag == "script" && name == "src" && value != "" {
			ctx.File.Imports = append(ctx.File.Imports, Import{Module: value, RawImport: value, Location: ctx.Location(attr)})
		}
		if tag == "link" && name == "href" && value != "" {
			ctx.File.Imports = append(ctx.File.Imports, Import{Module: value, RawImport: value, Location: ctx.Location(attr)})
		}
		if name == "id" && value != "" {
			ctx.File.Definitions = append(ctx.File.Definitions, Definition{Name: value, FullName: value, Kind: KindVariable, Exported: false, Location: ctx.Location(attr)})
		}
		if name == "class" {
			for _, className := range strings.Fields(value) {
				if className == "" {
					continue
				}
				ctx.File.References = append(ctx.File.References, Reference{Name: className, Location: ctx.Location(attr)})
			}
		}
	}
}

type cssProfileExtractor struct{}

func (e *cssProfileExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: "css",
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"import_statement": e.extractImport,
		"class_selector":   e.extractDefinition,
		"id_selector":      e.extractDefinition,
		"type_selector":    e.extractDefinition,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *cssProfileExtractor) extractImport(ctx *ExtractionContext, node *sitter.Node) {
	raw := ctx.Text(node)
	module := ""
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "string_value" || child.Kind() == "string" {
			module = trimQuoted(ctx.Text(child))
			break
		}
	}
	if module == "" {
		if start := strings.Index(raw, "\""); start >= 0 {
			if end := strings.LastIndex(raw, "\""); end > start {
				module = raw[start+1 : end]
			}
		}
	}
	if module == "" {
		return
	}

	ctx.File.Imports = append(ctx.File.Imports, Import{Module: module, RawImport: raw, Location: ctx.Location(node)})
}

func (e *cssProfileExtractor) extractDefinition(ctx *ExtractionContext, node *sitter.Node) {
	name := strings.TrimSpace(ctx.Text(node))
	name = strings.TrimPrefix(name, ".")
	name = strings.TrimPrefix(name, "#")
	if name == "" {
		return
	}
	ctx.File.Definitions = append(ctx.File.Definitions, Definition{Name: name, FullName: name, Kind: KindVariable, Location: ctx.Location(node)})
}

type goModProfileExtractor struct{}

func (e *goModProfileExtractor) Extract(_ *sitter.Node, source []byte, filePath string) (*File, error) {
	return e.ExtractRaw(source, filePath)
}

func (e *goModProfileExtractor) ExtractRaw(source []byte, filePath string) (*File, error) {
	file := &File{
		Path:        filePath,
		Language:    "gomod",
		PackageName: "gomod",
		ParsedAt:    time.Now(),
	}

	lines := strings.Split(string(source), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.HasPrefix(trimmed, "module ") {
			mod := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
			file.Module = mod
			file.Definitions = append(file.Definitions, Definition{Name: mod, FullName: mod, Kind: KindVariable, Exported: true, Location: Location{File: filePath, Line: 1, Column: 1}})
			continue
		}
		if strings.HasPrefix(trimmed, "require ") {
			fields := strings.Fields(strings.TrimPrefix(trimmed, "require "))
			if len(fields) > 0 {
				file.Imports = append(file.Imports, Import{Module: fields[0], RawImport: trimmed, Location: Location{File: filePath, Line: 1, Column: 1}})
			}
		}
	}

	return file, nil
}

type goSumProfileExtractor struct{}

func (e *goSumProfileExtractor) Extract(_ *sitter.Node, source []byte, filePath string) (*File, error) {
	return e.ExtractRaw(source, filePath)
}

func (e *goSumProfileExtractor) ExtractRaw(source []byte, filePath string) (*File, error) {
	file := &File{
		Path:        filePath,
		Language:    "gosum",
		PackageName: "gosum",
		ParsedAt:    time.Now(),
	}

	seen := make(map[string]bool)
	lines := strings.Split(string(source), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		mod := fields[0]
		if strings.HasSuffix(mod, "/go.mod") {
			mod = strings.TrimSuffix(mod, "/go.mod")
		}
		if mod == "" || seen[mod] {
			continue
		}
		seen[mod] = true
		file.Imports = append(file.Imports, Import{Module: mod, RawImport: line, Location: Location{File: filePath, Line: 1, Column: 1}})
	}

	return file, nil
}

package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/redneckbeard/thanos/facades"
	"github.com/redneckbeard/thanos/types"
)

// ParseProgram parses a Ruby file and all its require_relative dependencies
// into a single Root, then runs Analyze(). This is the multi-file entry point.
func ParseProgram(entryPath string) (*Root, error) {
	absPath, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path %s: %w", entryPath, err)
	}

	types.ClassRegistry.Initialize()

	// Load built-in facades that ship with thanos
	allFacades, err := facades.LoadBuiltins()
	if err != nil {
		return nil, fmt.Errorf("failed to load built-in facades: %w", err)
	}

	// Overlay project-local facades (can override builtins)
	if projectFacades := findFacadesConfig(filepath.Dir(absPath)); projectFacades != nil {
		for k, v := range projectFacades {
			allFacades[k] = v
		}
	}

	// Register all facades
	var allNamespaces []types.FacadeNamespace
	if len(allFacades) > 0 {
		for requireName, lib := range allFacades {
			namespaces := types.RegisterFacade(requireName, lib)
			allNamespaces = append(allNamespaces, namespaces...)
		}
		types.ClassRegistry.Initialize()
	}

	root := NewRoot()
	root.facades = allFacades

	// Create Module objects for scoped facade namespaces (e.g., Digest::SHA256)
	// so that ScopeAccessNode can resolve them via the scope chain.
	registerFacadeNamespaces(root, allNamespaces)
	loaded := map[string]bool{}

	if err := loadFile(absPath, root, loaded); err != nil {
		return root, err
	}

	if err := root.Analyze(); err != nil {
		return root, err
	}
	if err := root.ParseError(); err != nil {
		return root, err
	}
	return root, nil
}

// findFacadesConfig walks up from dir looking for .thanos/facades.json.
func findFacadesConfig(dir string) types.FacadeConfig {
	for {
		path := filepath.Join(dir, ".thanos", "facades.json")
		if config, err := types.LoadFacades(path); err == nil {
			return config
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}

// loadFile reads and parses a single Ruby file into the shared Root, then
// recursively loads any require_relative dependencies. Already-loaded files
// (tracked by absolute path) are skipped.
func loadFile(absPath string, root *Root, loaded map[string]bool) error {
	if loaded[absPath] {
		return nil
	}
	loaded[absPath] = true

	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", absPath, err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", absPath, err)
	}

	parser := yyNewParser()
	l := NewLexerWithRoot(b, root)
	parser.Parse(l)

	if pErr := root.ParseError(); pErr != nil {
		return pErr
	}

	// Scan statements for require/require_relative calls, resolve and load them,
	// then strip them from the Root.
	dir := filepath.Dir(absPath)
	var remaining []Node
	for _, stmt := range root.Statements {
		if call, ok := stmt.(*MethodCall); ok && call.Receiver == nil {
			switch call.MethodName {
			case "require_relative":
				if len(call.Args) == 1 {
					if rel := extractStringArg(call.Args[0]); rel != "" {
						if !strings.HasSuffix(rel, ".rb") {
							rel += ".rb"
						}
						depPath := filepath.Join(dir, rel)
						depAbs, err := filepath.Abs(depPath)
						if err != nil {
							return fmt.Errorf("could not resolve require_relative %q: %w", rel, err)
						}
						if err := loadFile(depAbs, root, loaded); err != nil {
							return err
						}
						continue // strip
					}
				}
			case "require":
				if len(call.Args) == 1 {
					if name := extractStringArg(call.Args[0]); name != "" {
						if root.facades != nil {
							if lib, hasBind := root.facades[name]; hasBind {
								// Warn if the facade is marked incomplete
								if lib.Coverage != "" {
									fmt.Fprintf(os.Stderr, "warning: require '%s': facade coverage is %s — some methods may not be available\n", name, lib.Coverage)
								}
								// Inject scope entries for this require (e.g., CSV::Row, CSV::Table)
								if inject, ok := requireScopeInjectors[name]; ok {
									inject(root)
								}
								continue // strip
							}
						}
						// No facade found — this is an error
						return fmt.Errorf("line %d: require '%s' has no thanos facade. See doc/facades.md for how to create one", call.LineNo(), name)
					}
				}
			}
		}
		remaining = append(remaining, stmt)
	}
	root.Statements = remaining

	return nil
}

// extractStringArg pulls the string value from a StringNode with no interpolations.
func extractStringArg(n Node) string {
	if node, ok := n.(*StringNode); ok {
		if len(node.BodySegments) == 1 && len(node.Interps) == 0 {
			return node.BodySegments[0]
		}
	}
	return ""
}

// registerFacadeNamespaces creates parser.Module objects for scoped facade
// names (e.g., "Digest::SHA256") and registers them in the Root's scope chain
// so that ScopeAccessNode can resolve them.
func registerFacadeNamespaces(root *Root, namespaces []types.FacadeNamespace) {
	for _, ns := range namespaces {
		mod := &Module{
			name:      ns.Name,
			MethodSet: NewMethodSet(),
		}
		// Add inner classes to the module — check ClassRegistry first,
		// then named type registry (for facade-defined types).
		for _, memberName := range ns.Members {
			var memberType types.Type
			if t, err := types.ClassRegistry.Get(memberName); err == nil {
				memberType = t
			} else if t, ok := types.LookupNamedType(memberName); ok {
				memberType = t
			} else {
				continue
			}
			cls := &Class{
				name:      memberName,
				_type:     memberType,
				MethodSet: NewMethodSet(),
				Module:    mod,
			}
			mod.Classes = append(mod.Classes, cls)
		}
		// Register the module in the Root's scope chain so ResolveVar finds it
		root.ScopeChain[0].Set(ns.Name, mod)
	}
}

// stripRequires removes `require` calls that match facades from the Root's
// statements and calls scope injectors for each matched require. Used by
// ParseBytes (gauntlet tests, stdin) where loadFile's require handling doesn't run.
func stripRequires(root *Root, allFacades types.FacadeConfig) {
	var remaining []Node
	for _, stmt := range root.Statements {
		if call, ok := stmt.(*MethodCall); ok && call.Receiver == nil && call.MethodName == "require" {
			if len(call.Args) == 1 {
				if name := extractStringArg(call.Args[0]); name != "" {
					if allFacades != nil {
						if _, hasBind := allFacades[name]; hasBind {
							if inject, ok := requireScopeInjectors[name]; ok {
								inject(root)
							}
							continue // strip
						}
					}
				}
			}
		}
		remaining = append(remaining, stmt)
	}
	root.Statements = remaining
}

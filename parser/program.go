package parser

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/redneckbeard/thanos/facades"
	"github.com/redneckbeard/thanos/types"
)

// NoGems disables gem source resolution via system Ruby. Facades still work.
var NoGems bool

// builtinRequires lists require names that thanos handles natively via its
// type system. These are silently stripped without needing a facade or gem source.
var builtinRequires = map[string]bool{
	"set": true,
}

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
	root.loadPaths = resolveLoadPaths(filepath.Dir(absPath))

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

// findThanosConfig walks up from dir looking for .thanos/config.yaml and
// extracts the ruby_command value. Returns "ruby" if not found.
func findThanosConfig(dir string) string {
	for {
		path := filepath.Join(dir, ".thanos", "config.yaml")
		if f, err := os.Open(path); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "ruby_command:") {
					val := strings.TrimSpace(strings.TrimPrefix(line, "ruby_command:"))
					if val != "" {
						return val
					}
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "ruby"
}

// resolveRubyCommand finds the best ruby binary when the default "ruby" is
// requested. It checks for rbenv and asdf shims which are commonly set up
// in shell init files that exec.Command won't source.
func resolveRubyCommand(rubyCmd string) string {
	if rubyCmd != "ruby" {
		return rubyCmd
	}
	// Check rbenv shim first
	home, err := os.UserHomeDir()
	if err != nil {
		return rubyCmd
	}
	for _, shimPath := range []string{
		filepath.Join(home, ".rbenv", "shims", "ruby"),
		filepath.Join(home, ".asdf", "shims", "ruby"),
	} {
		if _, err := os.Stat(shimPath); err == nil {
			return shimPath
		}
	}
	return rubyCmd
}

// resolveRubyLoadPaths shells out to the configured Ruby command to get $LOAD_PATH.
// Returns nil if Ruby is unavailable or errors.
func resolveRubyLoadPaths(rubyCmd string) []string {
	rubyCmd = resolveRubyCommand(rubyCmd)
	parts := strings.Fields(rubyCmd)
	if len(parts) == 0 {
		return nil
	}
	var cmd *exec.Cmd
	rubyScript := `puts $:; puts Gem.path.flat_map { |p| Dir.glob(File.join(p, "gems", "*", "lib")) }`
	if len(parts) > 1 {
		// Multi-word commands (e.g., "bundle exec ruby") need a shell
		// so that PATH and shims are set up correctly.
		fullCmd := rubyCmd + ` -e '` + rubyScript + `'`
		cmd = exec.Command("bash", "-lc", fullCmd)
	} else {
		cmd = exec.Command(parts[0], "-e", rubyScript)
	}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var paths []string
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !seen[line] {
			seen[line] = true
			paths = append(paths, line)
		}
	}
	return paths
}

// loadPathsCacheFile returns the path to the load paths cache file,
// searching upward from dir for a .thanos directory.
func loadPathsCacheDir(dir string) string {
	for {
		candidate := filepath.Join(dir, ".thanos")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// resolveLoadPaths gets Ruby load paths, using a cache when available.
// The cache is stored in .thanos/load_paths.cache and invalidated when
// the ruby_command changes.
func resolveLoadPaths(dir string) []string {
	if NoGems {
		return nil
	}
	rubyCmd := findThanosConfig(dir)
	cacheDir := loadPathsCacheDir(dir)

	// Try reading cache
	if cacheDir != "" {
		cachePath := filepath.Join(cacheDir, "load_paths.cache")
		if data, err := os.ReadFile(cachePath); err == nil {
			lines := strings.Split(string(data), "\n")
			if len(lines) > 1 {
				// First line is a hash of the ruby command for invalidation
				expectedHash := fmt.Sprintf("# ruby_command_hash: %x", sha256.Sum256([]byte(rubyCmd)))
				if lines[0] == expectedHash {
					var paths []string
					for _, line := range lines[1:] {
						line = strings.TrimSpace(line)
						if line != "" {
							paths = append(paths, line)
						}
					}
					if len(paths) > 0 {
						return paths
					}
				}
			}
		}
	}

	// Resolve fresh
	paths := resolveRubyLoadPaths(rubyCmd)

	// Write cache if we have a .thanos directory
	if cacheDir != "" && len(paths) > 0 {
		hash := fmt.Sprintf("# ruby_command_hash: %x", sha256.Sum256([]byte(rubyCmd)))
		content := hash + "\n" + strings.Join(paths, "\n") + "\n"
		_ = os.WriteFile(filepath.Join(cacheDir, "load_paths.cache"), []byte(content), 0644)
	}

	return paths
}

// resolveGemRequire searches load paths for a require name and returns the
// absolute path to the .rb file, or "" if not found.
func resolveGemRequire(name string, loadPaths []string) string {
	for _, dir := range loadPaths {
		candidate := filepath.Join(dir, name+".rb")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
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
	l := NewLexerWithRoot(b, root, absPath)
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
						// Built-in type — silently strip
						if builtinRequires[name] {
							continue // strip
						}
						// No facade — try resolving via Ruby load paths
						if gemPath := resolveGemRequire(name, root.loadPaths); gemPath != "" {
							fmt.Fprintf(os.Stderr, "warning: require '%s': resolved from gem source at %s — compilation may be incomplete\n", name, gemPath)
							if err := loadFile(gemPath, root, loaded); err != nil {
								// Gem files may contain unsupported Ruby constructs.
								// Warn and continue rather than failing the entire compilation.
								fmt.Fprintf(os.Stderr, "warning: require '%s': %v (continuing without this dependency)\n", name, err)
							}
							continue // strip
						}
						return fmt.Errorf("line %d: cannot locate source for require '%s' (no facade and no gem source found)", call.LineNo(), name)
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
// Also resolves gem sources via load paths when no facade matches.
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
					// Built-in type — silently strip
					if builtinRequires[name] {
						continue // strip
					}
					// No facade — try resolving via Ruby load paths
					if gemPath := resolveGemRequire(name, root.loadPaths); gemPath != "" {
						fmt.Fprintf(os.Stderr, "warning: require '%s': resolved from gem source at %s — compilation may be incomplete\n", name, gemPath)
						// Parse the gem file into the root — errors are non-fatal for gems
						if data, err := os.ReadFile(gemPath); err == nil {
							savedErrors := append([]error{}, root.Errors...)
							p := yyNewParser()
							l := NewLexerWithRoot(data, root, gemPath)
							p.Parse(l)
							if pErr := root.ParseError(); pErr != nil {
								fmt.Fprintf(os.Stderr, "warning: require '%s': %v (continuing without this dependency)\n", name, pErr)
								root.Errors = savedErrors
							}
						}
						continue // strip
					}
				}
			}
		}
		remaining = append(remaining, stmt)
	}
	root.Statements = remaining
}

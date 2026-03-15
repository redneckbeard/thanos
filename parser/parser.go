//go:generate ./gen_parser.sh

// package parser contains the requisite components for generating
// type-annotated Ruby ASTs in Go. At a high level, there are three such
// components: a lexer, implemented by hand; a parser, generated with goyacc;
// and a Node interface that, when implemented according to a specific
// convention, allows for some basic static analysis to tag AST nodes with a
// Type provided by the types package in this repository. It also provides
// utility functions for interacting with this chain of components from outside
// the package.

package parser

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/redneckbeard/thanos/facades"
	"github.com/redneckbeard/thanos/types"
)

func ParseFile(filename string) (*Root, error) {
	if filename != "" {
		// Use multi-file parser for file-based input (handles require_relative)
		var path string
		if filepath.IsAbs(filename) {
			path = filename
		} else {
			dir, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			path = filepath.Join(dir, filename)
		}
		return ParseProgram(path)
	}
	// stdin fallback
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return ParseBytes(b)
}

func ParseString(s string) (*Root, error) {
	return ParseBytes([]byte(s))
}

// loadAndRegisterFacades loads built-in facades, registers them in the type
// system, and returns any scoped namespaces (for :: resolution).
func loadAndRegisterFacades() []types.FacadeNamespace {
	allFacades, err := facades.LoadBuiltins()
	if err != nil {
		return nil
	}
	var allNamespaces []types.FacadeNamespace
	for requireName, lib := range allFacades {
		namespaces := types.RegisterFacade(requireName, lib)
		allNamespaces = append(allNamespaces, namespaces...)
	}
	types.ClassRegistry.Initialize()
	return allNamespaces
}

func ParseBytes(b []byte) (*Root, error) {
	types.ClassRegistry.Reset()
	types.ClassRegistry.Initialize()

	// Load built-in facades so that gauntlet tests and stdin input
	// can use facade-provided modules (e.g., Base64, Digest::SHA256).
	namespaces := loadAndRegisterFacades()

	parser := yyNewParser()
	l := NewLexer(b)
	parser.Parse(l)

	// Resolve Ruby load paths for gem source resolution
	if dir, err := os.Getwd(); err == nil {
		l.Root.loadPaths = resolveLoadPaths(dir)
	}

	// Register facade namespaces in scope for :: resolution
	registerFacadeNamespaces(l.Root, namespaces)

	// Strip `require` calls that match facades and inject scope entries.
	allFacades, _ := facades.LoadBuiltins()
	stripRequires(l.Root, allFacades)

	if err := l.Root.Analyze(); err != nil {
		return l.Root, err
	}
	if err := l.Root.ParseError(); err != nil {
		return l.Root, err
	} else {
		return l.Root, nil
	}
}

func DebugLevel() int {
	if val, found := os.LookupEnv("DEBUG"); found {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

func LogDebug(verbosity int, format string, v ...interface{}) {
	if DebugLevel() == verbosity {
		log.Printf(format, v...)
	}
}

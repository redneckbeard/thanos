//go:generate goyacc -l -o ruby.go ruby.y

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

	"github.com/redneckbeard/thanos/types"
)

func ParseFile(filename string) (*Root, error) {
	var f *os.File
	if filename == "" {
		f = os.Stdin
	} else {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		path := filepath.Join(dir, filename)
		f, err = os.Open(path)
		if err != nil {
			panic(err)
		}
	}
	if b, err := io.ReadAll(f); err != nil {
		return nil, err
	} else {
		return ParseBytes(b)
	}
}

func ParseString(s string) (*Root, error) {
	return ParseBytes([]byte(s))
}

func ParseBytes(b []byte) (*Root, error) {
	types.ClassRegistry.Initialize()
	parser := yyNewParser()
	l := NewLexer(b)
	parser.Parse(l)
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

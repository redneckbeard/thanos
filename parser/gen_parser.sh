#!/bin/bash
goyacc -l -o ruby.go ruby.y
sed 's/yyErrorVerbose = false/yyErrorVerbose = true/' ruby.go > ruby.go.verbose
sed 's/"syntax error: unexpected "/__yyfmt__.Sprintf("syntax error, line %d: unexpected ", currentLineNo)/' ruby.go.verbose > ruby.go.linenos
mv ruby.go.linenos ruby.go
rm ruby.go.*

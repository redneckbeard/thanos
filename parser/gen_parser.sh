#!/bin/bash
OUTPUT=$(goyacc -l -o ruby.go ruby.y 2>&1)
echo "$OUTPUT"
sed 's/yyErrorVerbose = false/yyErrorVerbose = true/' ruby.go > ruby.go.verbose
sed 's/"syntax error: unexpected "/formatSyntaxError(currentFile, currentLineNo) + "unexpected "/' ruby.go.verbose > ruby.go.linenos
mv ruby.go.linenos ruby.go
rm ruby.go.*

# Record conflicts in CONFLICTS.md
SHA=$(git log --oneline -1 -- ruby.y 2>/dev/null | cut -d' ' -f1 || echo "unknown")
SR=$(echo "$OUTPUT" | grep -o '[0-9]* shift/reduce' | grep -o '[0-9]*' || echo "0")
RR=$(echo "$OUTPUT" | grep -o '[0-9]* reduce/reduce' | grep -o '[0-9]*' || echo "0")
DATE=$(date +%Y-%m-%d)
echo "| $DATE | $SHA | ${SR:-0} | ${RR:-0} | |" >> CONFLICTS.md

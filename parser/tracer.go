package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

// AnalysisTracer records events during type analysis for debugging.
// When non-nil, the analysis pipeline emits structured trace events.
var Tracer *AnalysisTracer

type TraceEvent struct {
	Phase   string // e.g. "module-constants", "first-pass", "method-analyze"
	Action  string // e.g. "GetType", "ResolveVar", "SetType", "AnalyzeMethod"
	Node    string // string representation of the node
	Type    string // resolved type (if any)
	Scope   string // scope chain summary
	Detail  string // extra context
	Line    int
	File    string
	Depth   int // nesting depth for indentation
}

type AnalysisTracer struct {
	Events []TraceEvent
	depth  int
	phase  string
}

func NewTracer() *AnalysisTracer {
	return &AnalysisTracer{}
}

func (t *AnalysisTracer) SetPhase(phase string) {
	if t == nil {
		return
	}
	t.Events = append(t.Events, TraceEvent{
		Phase:  phase,
		Action: "phase",
		Depth:  0,
	})
	t.phase = phase
	t.depth = 0
}

func (t *AnalysisTracer) Enter(action, node string, line int) {
	if t == nil {
		return
	}
	t.Events = append(t.Events, TraceEvent{
		Phase:  t.phase,
		Action: action,
		Node:   node,
		Line:   line,
		Depth:  t.depth,
	})
	t.depth++
}

func (t *AnalysisTracer) Exit(action, node string, line int, typ types.Type) {
	if t == nil {
		return
	}
	t.depth--
	if t.depth < 0 {
		t.depth = 0
	}
	var typeStr string
	if typ != nil {
		typeStr = typ.String()
	}
	t.Events = append(t.Events, TraceEvent{
		Phase:  t.phase,
		Action: action + " →",
		Node:   node,
		Type:   typeStr,
		Line:   line,
		Depth:  t.depth,
	})
}

func (t *AnalysisTracer) Record(action, detail string) {
	if t == nil {
		return
	}
	t.Events = append(t.Events, TraceEvent{
		Phase:  t.phase,
		Action: action,
		Detail: detail,
		Depth:  t.depth,
	})
}

func (t *AnalysisTracer) RecordScope(action, name string, scope ScopeChain) {
	if t == nil {
		return
	}
	names := make([]string, len(scope))
	for i, s := range scope {
		names[i] = s.Name()
	}
	t.Events = append(t.Events, TraceEvent{
		Phase:  t.phase,
		Action: action,
		Node:   name,
		Scope:  strings.Join(names, " → "),
		Depth:  t.depth,
	})
}

// WriteProcess writes the process trace in a human-readable format.
func (t *AnalysisTracer) WriteProcess(w io.Writer) {
	for _, e := range t.Events {
		indent := strings.Repeat("  ", e.Depth)
		switch e.Action {
		case "phase":
			fmt.Fprintf(w, "\n══ %s ══\n", e.Phase)
		default:
			var parts []string
			parts = append(parts, fmt.Sprintf("%s%s", indent, e.Action))
			if e.Node != "" {
				parts = append(parts, e.Node)
			}
			if e.Type != "" {
				parts = append(parts, fmt.Sprintf("=> %s", e.Type))
			}
			if e.Scope != "" {
				parts = append(parts, fmt.Sprintf("[scope: %s]", e.Scope))
			}
			if e.Detail != "" {
				parts = append(parts, fmt.Sprintf("(%s)", e.Detail))
			}
			if e.Line > 0 {
				loc := fmt.Sprintf("line %d", e.Line)
				if e.File != "" {
					loc = fmt.Sprintf("%s:%d", e.File, e.Line)
				}
				parts = append(parts, loc)
			}
			fmt.Fprintln(w, strings.Join(parts, " "))
		}
	}
}

// nodeLabel returns a concise label for a node suitable for trace output.
func nodeLabel(n Node) string {
	switch v := n.(type) {
	case *IdentNode:
		return fmt.Sprintf("ident(%s)", v.Val)
	case *MethodCall:
		if v.Receiver != nil {
			return fmt.Sprintf("call(%s.%s)", nodeLabel(v.Receiver), v.MethodName)
		}
		return fmt.Sprintf("call(%s)", v.MethodName)
	case *AssignmentNode:
		if len(v.Left) > 0 {
			return fmt.Sprintf("assign(%s)", nodeLabel(v.Left[0]))
		}
		return "assign(?)"
	case *ConstantNode:
		if v.Namespace != "" {
			return fmt.Sprintf("const(%s::%s)", v.Namespace, v.Val)
		}
		return fmt.Sprintf("const(%s)", v.Val)
	case *IntNode:
		return fmt.Sprintf("int(%s)", v.Val)
	case *Float64Node:
		return fmt.Sprintf("float(%s)", v.Val)
	case *StringNode:
		return "string(...)"
	case *BooleanNode:
		return fmt.Sprintf("bool(%s)", v.Val)
	case *SymbolNode:
		return fmt.Sprintf("sym(%s)", v.Val)
	case *ArrayNode:
		return fmt.Sprintf("array(%d elems)", len(v.Args))
	case *HashNode:
		return fmt.Sprintf("hash(%d pairs)", len(v.Pairs))
	case *IVarNode:
		return fmt.Sprintf("ivar(%s)", v.Val)
	case *CVarNode:
		return fmt.Sprintf("cvar(%s)", v.Val)
	case *ReturnNode:
		return "return"
	case *Condition:
		return "if"
	case *WhileNode:
		return "while"
	case *CaseNode:
		return "case"
	case *SelfNode:
		return "self"
	case *NilNode:
		return "nil"
	case *InfixExpressionNode:
		return fmt.Sprintf("infix(%s)", v.Operator)
	case Statements:
		return fmt.Sprintf("stmts(%d)", len(v))
	default:
		s := fmt.Sprintf("%T", n)
		// Strip *parser. prefix
		if idx := strings.LastIndex(s, "."); idx >= 0 {
			s = s[idx+1:]
		}
		return s
	}
}

package parser

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteAnnotations outputs the analyzed Ruby source with type annotations
// as comments showing what the analyzer inferred for variables and methods.
func WriteAnnotations(w io.Writer, root *Root, source []byte) {
	lines := strings.Split(string(source), "\n")
	annotations := collectAnnotations(root)

	for i, line := range lines {
		lineNo := i + 1
		fmt.Fprintln(w, line)
		if anns, ok := annotations[lineNo]; ok {
			for _, ann := range anns {
				fmt.Fprintf(w, "  # ^ %s\n", ann)
			}
		}
	}
}

type annotation struct {
	line int
	text string
}

func collectAnnotations(root *Root) map[int][]string {
	anns := map[int][]string{}
	add := func(line int, text string) {
		anns[line] = append(anns[line], text)
	}

	// Collect from top-level methods
	for name, m := range root.MethodSetStack.Peek().Methods {
		describeMethod(m, name, "", add)
	}

	// Collect from classes
	for _, cls := range root.Classes {
		describeClass(cls, add)
	}

	// Collect from modules
	for _, mod := range root.TopLevelModules {
		describeModule(mod, add)
	}

	// Collect from top-level statements
	for _, stmt := range root.Statements {
		describeNode(stmt, "", add)
	}

	// Sort annotations within each line
	for line := range anns {
		sort.Strings(anns[line])
	}

	return anns
}

func describeMethod(m *Method, name, prefix string, add func(int, string)) {
	if m.uncallable {
		add(m.LineNo(), fmt.Sprintf("%s%s: uncallable (never called)", prefix, name))
		return
	}
	var params []string
	for _, p := range m.Params {
		t := "?"
		if p.Type() != nil {
			t = p.Type().String()
		}
		params = append(params, fmt.Sprintf("%s: %s", p.Name, t))
	}
	ret := "?"
	if m.ReturnType() != nil {
		ret = m.ReturnType().String()
	}
	add(m.LineNo(), fmt.Sprintf("%s%s(%s) -> %s", prefix, name, strings.Join(params, ", "), ret))
}

func describeClass(cls *Class, add func(int, string)) {
	prefix := cls.name + "#"
	add(cls.LineNo(), fmt.Sprintf("class %s", cls.name))

	// Instance variables
	for name, ivar := range cls.IVarMap() {
		t := "?"
		if ivar.Type() != nil {
			t = ivar.Type().String()
		}
		add(cls.LineNo(), fmt.Sprintf("  @%s: %s", name, t))
	}

	// Instance methods
	for name, m := range cls.MethodSet.Methods {
		describeMethod(m, name, prefix, add)
	}

	// Class methods
	for _, m := range cls.ClassMethods {
		describeMethod(m, m.Name, cls.name+".", add)
	}
}

func describeModule(mod *Module, add func(int, string)) {
	add(mod.LineNo(), fmt.Sprintf("module %s", mod.name))

	// Constants
	for _, c := range mod.Constants {
		t := "?"
		if c.Type() != nil {
			t = c.Type().String()
		}
		add(constantLineNo(c), fmt.Sprintf("%s::%s: %s", mod.name, c.Name(), t))
	}

	// Module methods
	for _, m := range mod.ClassMethods {
		describeMethod(m, m.Name, mod.name+".", add)
	}

	// Classes in module
	for _, cls := range mod.Classes {
		describeClass(cls, add)
	}

	// Sub-modules
	for _, sub := range mod.Modules {
		describeModule(sub, add)
	}
}

func describeNode(n Node, context string, add func(int, string)) {
	switch node := n.(type) {
	case *AssignmentNode:
		for i, left := range node.Left {
			switch lhs := left.(type) {
			case *IdentNode:
				t := "?"
				if i < len(node.Right) && node.Right[i].Type() != nil {
					t = node.Right[i].Type().String()
				} else if lhs.Type() != nil {
					t = lhs.Type().String()
				}
				add(lhs.LineNo(), fmt.Sprintf("%s: %s", lhs.Val, t))
			case *ConstantNode:
				t := "?"
				if lhs.Type() != nil {
					t = lhs.Type().String()
				}
				name := lhs.Val
				if lhs.Namespace != "" {
					name = lhs.Namespace + "::" + name
				}
				add(lhs.LineNo(), fmt.Sprintf("%s: %s", name, t))
			}
		}
	case *MethodCall:
		if node.Type() != nil {
			label := node.MethodName
			if node.Receiver != nil {
				label = nodeLabel(node.Receiver) + "." + label
			}
			add(node.LineNo(), fmt.Sprintf("%s => %s", label, node.Type()))
		}
	}
}

// IVarMap returns the instance variable map for annotation output.
func (cls *Class) IVarMap() map[string]*IVar {
	return cls.ivars
}

// constantLineNo returns the constant's source line number.
func constantLineNo(c *Constant) int {
	if c.Val != nil {
		return c.Val.LineNo()
	}
	return 0
}

package parser

import "github.com/redneckbeard/thanos/types"

// requireScopeInjectors maps require names to functions that inject scope
// entries into a Root when the corresponding `require` is processed.
// This makes `require 'csv'` register CSV::Row and CSV::Table in the scope
// so that ScopeAccessNode and ConstantNode can resolve them.
var requireScopeInjectors = map[string]func(*Root){
	"csv":        injectCSVScope,
	"net/http":   injectNetHTTPScope,
	"shellwords": injectShellwordsScope,
	"uri":        injectURIScope,
	"yaml":       injectYAMLScope,
	"zlib":       injectZlibScope,
}

func injectCSVScope(root *Root) {
	mod := &Module{
		name:      "CSV",
		_type:     types.CSVClass,
		MethodSet: NewMethodSet(),
	}

	// Look up inner types from the named type registry (populated by
	// facade JSON "types" and/or csv/types.go init()).
	for _, name := range []string{"CSV::Row", "CSV::Table"} {
		if t, ok := types.LookupNamedType(name); ok {
			cls := &Class{
				name:      name,
				_type:     t,
				MethodSet: NewMethodSet(),
				Module:    mod,
			}
			mod.Classes = append(mod.Classes, cls)
		}
	}

	root.ScopeChain[0].Set("CSV", mod)
}

func injectNetHTTPScope(root *Root) {
	// Net is the outer module, HTTP is the inner class.
	// Net::HTTP.get(...) resolves as: scope["Net"] → Module → ConstGet("HTTP") → Class
	netMod := &Module{
		name:      "Net",
		MethodSet: NewMethodSet(),
	}

	httpCls := &Class{
		name:      "HTTP",
		_type:     types.NetHTTPClass,
		MethodSet: NewMethodSet(),
		Module:    netMod,
	}

	// Register request verb classes inside HTTP (for Net::HTTP::Get.new etc.)
	// These are added as Constants on the Class so ConstGet resolves them.
	for _, verb := range []string{"Get", "Post", "Put", "Patch", "Delete", "Head"} {
		if t, ok := types.LookupNamedType("Net::HTTP::" + verb); ok {
			httpCls.AddConstant(&Constant{
				name:  verb,
				_type: t,
			})
		}
	}

	netMod.Classes = append(netMod.Classes, httpCls)

	// Register Net::HTTPResponse on the Net module for direct access
	if t, ok := types.LookupNamedType("Net::HTTPResponse"); ok {
		respCls := &Class{
			name:      "HTTPResponse",
			_type:     t,
			MethodSet: NewMethodSet(),
			Module:    netMod,
		}
		netMod.Classes = append(netMod.Classes, respCls)
	}

	root.ScopeChain[0].Set("Net", netMod)
}

func injectShellwordsScope(root *Root) {
	injectSimpleModuleScope(root, "Shellwords")
}

func injectURIScope(root *Root) {
	mod := injectSimpleModuleScope(root, "URI")
	// Register the URI facade type so URI.parse returns it
	if t, ok := types.LookupNamedType("URI"); ok {
		cls := &Class{
			name:      "URI",
			_type:     t,
			MethodSet: NewMethodSet(),
			Module:    mod,
		}
		mod.Classes = append(mod.Classes, cls)
	}
}

func injectYAMLScope(root *Root) {
	injectSimpleModuleScope(root, "YAML")
}

func injectZlibScope(root *Root) {
	injectSimpleModuleScope(root, "Zlib")
}

// injectSimpleModuleScope registers a module-only facade (no inner types)
// in the Root's scope chain. Returns the module for further customization.
func injectSimpleModuleScope(root *Root, name string) *Module {
	cls, err := types.ClassRegistry.Get(name)
	var t types.Type
	if err == nil {
		t = cls
	}
	mod := &Module{
		name:      name,
		_type:     t,
		MethodSet: NewMethodSet(),
	}
	root.ScopeChain[0].Set(name, mod)
	return mod
}

package parser

import (
	"flag"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/redneckbeard/thanos/types"
)

var caseNum int

func init() {
	flag.IntVar(&caseNum, "case_num", 0, "which test case to run")
}

func TestPrecedence(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{":foo && true + 7", "(:foo && (true + 7))"},
		{"!true && false", "(!true && false)"},
		{"2 * 10 ** 2 + 5", "((2 * (10 ** 2)) + 5)"},
		{"foo = 5 + 10", "(foo = (5 + 10))"},
		//{"@foo = 5 + 10", "(@foo = (5 + 10))"},
		//{"@@foo = 5 + 10", "(@@foo = (5 + 10))"},
		{"$foo = 5 + 10", "($foo = (5 + 10))"},
		{"foo = meth(1, bar = 2)", "(foo = (meth(1, (bar = 2))))"},
		{"foo = obj.meth(1, bar = 2)", "(foo = (obj.meth(1, (bar = 2))))"},
		{"foo = [1, 2]", "(foo = [1, 2])"},
		{"foo = obj.meth(1, 2)", "(foo = (obj.meth(1, 2)))"},
		{"if foo + 2 then bar + 1 else baz * 5 end", "(if (foo + 2) (bar + 1) (else (baz * 5)))"},
		{"if foo + 2 then bar + 1 elsif quux then 10 else baz * 5 end", "(if (foo + 2) (bar + 1) (if quux 10 (else (baz * 5))))"},
		{"unless foo + 2 then bar + 1 else baz * 5 end", "(if !(foo + 2) (bar + 1) (else (baz * 5)))"},
		{`if foo + 2
		  bar + 1
		elsif quux
		  10
		else
		  baz * 5
		end`, "(if (foo + 2) (bar + 1) (if quux 10 (else (baz * 5))))"},
		{`if foo < 1 && bar >= 6
		   true
			elsif foo == bar
			 true
			else
			 false
		  end`, "(if ((foo < 1) && (bar >= 6)) true (if (foo == bar) true (else false)))"},
		{"foo[0] = :fluff", "(foo[0] = :fluff)"},
		{"[0, 1] << 2", "([0, 1] << 2)"},
		{"def x(n); return n + 1; end; x(5)", `(def x(n); (return (n + 1)); end)
(x(5))`},
		{"x > 5 ? true : false", "(if (x > 5) true (else false))"},
		{"[1,2,3].each do |x|; x + 1; end", "([1, 2, 3].each(block = (|x| (return (x + 1)))))"},
		{"[1,2,3].reduce(0) {|acc, n| acc + n }", "([1, 2, 3].reduce(0, block = (|acc, n| (return (acc + n)))))"},
		{"def x(); [1,2,3].reduce(0) {|acc, n| acc + n }; end", "(def x(); (return ([1, 2, 3].reduce(0, block = (|acc, n| (return (acc + n)))))); end)"},
		{`foo = "string"`, `(foo = "string")`},
		{`foo = ""`, `(foo = "")`},
		{`foo = "bar#{"baz"}quux#{5}"`, `(foo = ("bar%squux%d" % ("baz", 5)))`},
		{`foo = {:bar => "baz", :quux => "foo"}`, `(foo = {:bar => "baz", :quux => "foo"})`},
		{`puts "I've got rhythm"`, `(Kernel.puts("I've got rhythm"))`},
		{`foo = bar[1]`, `(foo = bar[1])`},
		{`foo = bar["string"]`, `(foo = bar["string"])`},
		{`foo([1,2,3,4]).select do |x|
  x % 2 == 0
end.length`, `(((foo([1, 2, 3, 4])).select(block = (|x| ((x % 2) == 0)))).length())`},
		{`foo = :steve # comment goes here
		bar = foo`, "(foo = :steve)\n(bar = foo)"},
		{`# comment goes here
		bar = foo`, "(bar = foo)"},
		{`foo = :cookies
		# comment goes here
		bar = foo`, "(foo = :cookies)\n(bar = foo)"},
		{`foo = :cookies
		bar = foo
		# comment goes here
		`, "(foo = :cookies)\n(bar = foo)"},
		{`3...4`, `(3...4)`},
		{`3..`, `(3..)`},
		{`3..4`, `(3..4)`},
		{`foo[3...]`, `foo[(3...)]`},
		{`foo, bar = quux 4`, `((foo, bar) = (quux(4)))`},
		{`def foo; return 4, true; end`, `(def foo(); (return 4, true); end)`},
		{`foo.gsub(/foo/, "bar")`, `(foo.gsub(/foo/, "bar"))`},
		{`class Foo; def bar; puts "blah"; end; def baz; puts "blah"; end; end`,
			`Foo([] (def bar(); (Kernel.puts("blah")); end); (def baz(); (Kernel.puts("blah")); end))`},
		{`foo += x`, `(foo = (foo + x))`},
		{`case x when 'a' then 1 when 'b' then 2 else 3 end`, `(case x (when ('a') 1); (when ('b') 2); (else 3))`},
		{`class Foo
		    attr_writer :foo
		    attr_accessor :baz
				attr_reader :quux
		    def bar
				  puts "blah"
				end
			end`,
			`Foo([@foo+w, @baz+rw, @quux+r] (def bar(); (Kernel.puts("blah")); end); (def foo=(foo); (@foo = foo); end); (def quux(); (return @quux); end))`},
		{`def foo(bar = "baz"); puts bar; end`, `(def foo(bar); (Kernel.puts(bar)); end)`},
		{`5.even?`, `(5.even?())`},
		{`5.2.positive?`, `(5.2.positive?())`},
		{`-5.2.positive?`, `(-5.2.positive?())`},
		{`puts []`, `(Kernel.puts([]))`},
		{`puts (x + y) / 4`, `(Kernel.puts(((x + y) / 4)))`},
		{`Pi = 3.14`, `(Pi = 3.14)`},
		{`Math::Pi`, `(Math::Pi)`},
		{`while x > 2
		x -= 1
		end`, `(while (x > 2) ((x = (x - 1))))`},
		{`while x > 2 do
		x -= 1
		end`, `(while (x > 2) ((x = (x - 1))))`},
		{`until x > 2 do
		x += 1
		end`, `(while !(x > 2) ((x = (x + 1))))`},

		// none of these tests will have exactly correct output, because the
		// `return` will not get applied until full analysis is complete, which
		// can't happen without a method call with a block. They are here to prove
		// that `yield` gets collapsed into the same AST structure as an explicit
		// block.
		{`def foo(&blk); blk.call("foo"); end`, `(def foo(&blk); (blk.call("foo")); end)`},
		{`def foo; yield("foo"); end`, `(def foo(&blk); (blk.call("foo")); end)`},
		{`def foo; yield "foo"; end`, `(def foo(&blk); (blk.call("foo")); end)`},
		{`def foo; yield(); end`, `(def foo(&blk); (blk.call()); end)`},
		{`def foo; yield; end`, `(def foo(&blk); (blk.call()); end)`},

		{`a = b, c`, `(a = [b, c])`},
		{`a, b = c, d`, `((a, b) = (c, d))`},
		{`a, b = c`, `((a, b) = c)`},
	}

	for i, tt := range tests {
		if caseNum == 0 || caseNum == i+1 {
			p, err := ParseString(tt.input)
			if p.String() != tt.expected {
				t.Errorf("[%d] Expected %q but got %q", i+1, tt.expected, p.String())
				if err != nil {
					t.Errorf("[%d] Parse errors: %s", i+1, err)
				}
			}
		}
	}
}

func TestMethodParamInferenceHappyPath(t *testing.T) {
	tests := []struct {
		input         string
		argumentTypes map[string]types.Type
		ReturnType    types.Type
	}{
		{
			input:         `def foo(bar, baz); bar + baz; end; foo(1, 2)`,
			argumentTypes: map[string]types.Type{"bar": types.IntType, "baz": types.IntType},
			ReturnType:    types.IntType,
		},
		{
			input: `def foo(bar, baz)
		   if bar
				 bar
			 elsif baz
				 baz
			 else
				 10
			 end
		 end
		 foo(1, 2)`,
			argumentTypes: map[string]types.Type{"bar": types.IntType, "baz": types.IntType},
			ReturnType:    types.IntType,
		},
		{
			input:         `def foo(bar, baz); [bar, baz]; end; foo(false, true)`,
			argumentTypes: map[string]types.Type{"bar": types.BoolType, "baz": types.BoolType},
			ReturnType:    types.NewArray(types.BoolType),
		},
		{
			input:         `def foo(bar); bar[0] = 4; end; foo([1, 2, 3])`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.IntType)},
			ReturnType:    types.IntType,
		},
		{
			input:         `def foo(bar, baz); bar << baz; end; foo([1, 2], 3)`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.IntType), "baz": types.IntType},
			ReturnType:    types.NewArray(types.IntType),
		},
		{
			input:         `def foo(bar, baz); return baz; end; foo([1, 2], false)`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.IntType), "baz": types.BoolType},
			ReturnType:    types.BoolType,
		},
		{
			input:         `def foo(bar, baz);  baz ? bar : 0; end; foo(1, false)`,
			argumentTypes: map[string]types.Type{"bar": types.IntType, "baz": types.BoolType},
			ReturnType:    types.IntType,
		},
		{
			input: `def foo(bar) bar.map do |x| x % 2 == 0 end; end
foo([1,2,3,4,5])`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.IntType)},
			ReturnType:    types.NewArray(types.BoolType),
		},
		{
			input: `def foo(bar) bar + "foo"; end
foo("bar")`,
			argumentTypes: map[string]types.Type{"bar": types.StringType},
			ReturnType:    types.StringType,
		},
		{
			input: `def foo(bar)
		   if bar
				 "sandwiches"
			 else
			 	 "sausages" 
			 end
		 end
		 foo(1)`,
			argumentTypes: map[string]types.Type{"bar": types.IntType},
			ReturnType:    types.StringType,
		},
		{
			input: `def foo(bar)
		   if bar
				 "sandwiches"
			 else
			 	 "sausages" 
			 end
		 end
		 foo(1)`,
			argumentTypes: map[string]types.Type{"bar": types.IntType},
			ReturnType:    types.StringType,
		},
		{
			input: `def foo(bar) bar[2]; end
foo(["bar", "baz", "quux"])`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.StringType)},
			ReturnType:    types.StringType,
		},
		{
			input: `def foo(bar) bar["foo"]; end
foo({"bar" => true, "baz" => false})`,
			argumentTypes: map[string]types.Type{"bar": types.NewHash(types.StringType, types.BoolType)},
			ReturnType:    types.BoolType,
		},
		{
			input: `def foo(bar) bar.delete("bar"); end
foo({"bar" => true, "baz" => false})`,
			argumentTypes: map[string]types.Type{"bar": types.NewHash(types.StringType, types.BoolType)},
			ReturnType:    types.BoolType,
		},
		{
			input: `def foo(bar, baz) bar[baz...]; end
foo([1,2,3], 2)`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.IntType), "baz": types.IntType},
			ReturnType:    types.NewArray(types.IntType),
		},
		{
			input: `def foo(bar) bar.is_a?(Array); end
foo([1,2,3])`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.IntType)},
			ReturnType:    types.BoolType,
		},
		{
			input: `def foo(bar); return bar, 4; end
foo([1,2,3])`,
			argumentTypes: map[string]types.Type{"bar": types.NewArray(types.IntType)},
			ReturnType:    types.Multiple{types.NewArray(types.IntType), types.IntType},
		},
		{
			input: `def foo(bar); /.uu./ =~ bar; end
foo("quux")`,
			argumentTypes: map[string]types.Type{"bar": types.StringType},
			ReturnType:    types.BoolType,
		},
		{
			input: `def foo(bar)
		    bar.match(/(x*)(y+)/)[2]
			end
			foo("yyy")`,
			argumentTypes: map[string]types.Type{"bar": types.StringType},
			ReturnType:    types.StringType,
		},
		{
			input: `def foo(bar)
		    bar.match(/(x*)(?<y>y+)/)["y"]
			end
			foo("yyy")`,
			argumentTypes: map[string]types.Type{"bar": types.StringType},
			ReturnType:    types.StringType,
		},
		{
			input: `def foo(bar)
		    bar.match(/(x*)(?<y>y+)/).captures
			end
			foo("yyy")`,
			argumentTypes: map[string]types.Type{"bar": types.StringType},
			ReturnType:    types.NewArray(types.StringType),
		},
		{
			input: `def foo(bar)
			  case bar
				when 1 then "foo"
				when 2 then "bar"
				else
				  "baz"
				end
			end
			foo(:blah)`,
			argumentTypes: map[string]types.Type{"bar": types.SymbolType},
			ReturnType:    types.StringType,
		},
		{
			input: `def foo(bar)
			  case bar
				when 1 then "foo"
				when 2 then "bar"
				else
				  puts "maybe a debugging message"
				end
			end
			foo(:blah)`,
			argumentTypes: map[string]types.Type{"bar": types.SymbolType},
			ReturnType:    types.StringType,
		},
		{
			input: `
			def foo(bar, positive: true)
			   if positive
				   bar
				 else
				   bar * -1
				 end
			end
			foo(5)
			`,
			argumentTypes: map[string]types.Type{"bar": types.IntType, "positive": types.BoolType},
			ReturnType:    types.IntType,
		},
		{
			input: `
			def foo(bar: 10, positive: true)
			   if positive
				   bar
				 else
				   bar * -1
				 end
			end
			foo(positive: false)
			`,
			argumentTypes: map[string]types.Type{"bar": types.IntType, "positive": types.BoolType},
			ReturnType:    types.IntType,
		},
		{
			input: `
			def foo(bar = 10, positive = true)
			   if positive
				   bar
				 else
				   bar * -1
				 end
			end
			foo
			`,
			argumentTypes: map[string]types.Type{"bar": types.IntType, "positive": types.BoolType},
			ReturnType:    types.IntType,
		},
		{
			input: `
			Pi = 3.14

			def foo(radius)
			  Pi * radius ** 2
			end
			foo(4)
			`,
			argumentTypes: map[string]types.Type{"radius": types.IntType},
			ReturnType:    types.FloatType,
		},
		{
			input: `
			def foo(a, b)
			  c, d = a ** 2, b ** 2.0
				c + d
			end
			foo(1, 2)
			`,
			argumentTypes: map[string]types.Type{"a": types.IntType, "b": types.IntType},
			ReturnType:    types.FloatType,
		},
		{
			input: `
			def foo(a, b)
			  c = a, b
				c << 3
			end
			foo(1, 2)
			`,
			argumentTypes: map[string]types.Type{"a": types.IntType, "b": types.IntType},
			ReturnType:    types.NewArray(types.IntType),
		},
	}

	for i, tt := range tests {
		func(i int, tt struct {
			input         string
			argumentTypes map[string]types.Type
			ReturnType    types.Type
		}) {
			defer func() {
				if v := recover(); v != nil {
					t.Errorf("[Test case %d] Encountered panic in processing `%s`:\n%s", i+i, tt.input, debug.Stack())
				}
			}()
			if caseNum == 0 || caseNum == i+1 {
				program, err := ParseString(tt.input)
				if err != nil {
					t.Fatalf("[Test Case %d] %s", i+1, err)
				}
				method, ok := program.GetMethod("foo")
				if !ok {
					t.Fatalf("Could not find method '%s'", "foo")
				}
				for j := 0; j < len(tt.argumentTypes); j++ {
					param, _ := method.GetParam(j)
					if param != method.GetParamByName(param.Name) {
						t.Errorf("[Test Case %d] positional vs optional arg access differs for parameter '%s'", i+1, param.Name)
						break
					}
					expectedType := tt.argumentTypes[param.Name]
					if param.Type() != expectedType {
						t.Errorf("[Test Case %d] type inference failed for parameter '%s': expected %s, but got %s", i+1, param.Name, expectedType, param.Type())
						break
					}
				}
				if !method.ReturnType().Equals(tt.ReturnType) {
					t.Errorf("[Test Case %d] type inference failed for return type for method '%s': expected %s, got %s", i+1, method.Name, tt.ReturnType, method.ReturnType())
				}
			}
		}(i, tt)
	}
}

func TestInstanceMethodParamInferenceHappyPath(t *testing.T) {
	tests := []struct {
		input         string
		argumentTypes []map[string]types.Type
		returnTypes   []types.Type
	}{
		{
			input: `
			class Foo
			  def bar
			    10
			  end

				def baz
          "baz"
			  end
			end

      foo = Foo.new
			foo.bar
			foo.baz
			`,
			argumentTypes: []map[string]types.Type{},
			returnTypes:   []types.Type{types.IntType, types.StringType},
		},
		{
			input: `
			class Foo
			  def bar(x)
			    x + 10
			  end

				def baz(infix)
          "ba#{infix}zz"
			  end
			end

      foo = Foo.new
			foo.bar(5)
			foo.baz("quux")
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{"x": types.IntType},
				map[string]types.Type{"infix": types.StringType},
			},
			returnTypes: []types.Type{types.IntType, types.StringType},
		},
		{
			input: `
			class Foo
			  def bar(x)
			    baz(x) + 10
			  end

				def baz(y)
          y * y 
			  end
			end

      foo = Foo.new
			foo.bar(5)
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{"x": types.IntType},
				map[string]types.Type{"y": types.IntType},
			},
			returnTypes: []types.Type{types.IntType, types.IntType},
		},
		{
			input: `
			def quux(x)
			  x * x
			end

			class Foo
			  def bar(x)
			    baz(x) + 10
			  end

				def baz(y)
          quux(y)  
			  end
			end

      foo = Foo.new
			foo.bar(5)
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{"x": types.IntType},
				map[string]types.Type{"y": types.IntType},
			},
			returnTypes: []types.Type{types.IntType, types.IntType},
		},
		{
			input: `
			class Foo
			  def initialize(x)
				  @x = x
				end
				
			  def bar
				  @x
			  end

				def baz
				  "blah-#{bar}"
				end
			end

      foo = Foo.new(10).baz
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{},
				map[string]types.Type{},
			},
			returnTypes: []types.Type{types.IntType, types.StringType},
		},
		{
			input: `
			class Foo
			  attr_reader :x

			  def initialize(x)
				  @x = x
				end
				
			  def bar
				  x
			  end

				def baz
				  x = "blah-#{bar()}"
					x
				end
			end

      foo = Foo.new(10).baz
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{},
				map[string]types.Type{},
			},
			returnTypes: []types.Type{types.IntType, types.StringType},
		},
		{
			input: `
			class BaseFoo
			  attr_reader :x

			  def initialize(x)
				  @x = x
				end
				
			  def bar
				  x
			  end

				def baz
				  x = "blah-#{bar()}"
					x
				end
			end

			class Foo < BaseFoo
			end

      foo = Foo.new(10).baz
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{},
				map[string]types.Type{},
			},
			returnTypes: []types.Type{types.IntType, types.StringType},
		},
		{
			input: `
			class Foo
			  BAR = 100
				
			  def bar
				  BAR
			  end

				def baz
				  bar * 2.0
				end
			end

      foo = Foo.new(10).baz
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{},
				map[string]types.Type{},
			},
			returnTypes: []types.Type{types.IntType, types.FloatType},
		},
		{
			input: `
			class Foo
			  def bar
					Bar::BAZ
			  end

				def baz
				  bar * 2.0
				end
			end

			class Bar
			  BAZ = 100
			end

      foo = Foo.new(10).baz
			`,
			argumentTypes: []map[string]types.Type{
				map[string]types.Type{},
				map[string]types.Type{},
			},
			returnTypes: []types.Type{types.IntType, types.FloatType},
		},
	}

	for i, tt := range tests {
		if caseNum == 0 || caseNum == i+1 {
			program, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("[Test Case %d] %s", i+1, err)
			}
			class := program.Classes[0]
			for j, name := range []string{"bar", "baz"} {
				method, ok := class.MethodSet.Methods[name]
				if !ok {
					t.Fatalf("Could not find method '%s'", name)
				}
				if len(method.Params) > 0 {
					for k := 0; k < len(tt.argumentTypes[j]); k++ {
						param, _ := method.GetParam(k)
						if param != method.GetParamByName(param.Name) {
							t.Errorf("[Test Case %d] positional vs optional arg access differs for parameter '%s'", i+1, param.Name)
							break
						}
						expectedType := tt.argumentTypes[j][param.Name]
						if param.Type() != expectedType {
							t.Errorf("[Test Case %d] type inference failed for parameter '%s': expected %s, but got %s", i+1, param.Name, expectedType, param.Type())
							break
						}
					}
				}
				if !method.ReturnType().Equals(tt.returnTypes[j]) {
					t.Errorf("[Test Case %d] type inference failed for return type for method '%s': expected %s, got %s", i+1, method.Name, tt.returnTypes[j], method.ReturnType())
					break
				}
			}
		}
	}
}

func TestBlockParamInferenceHappyPath(t *testing.T) {
	tests := []struct {
		input              string
		blockArgumentTypes map[string]types.Type
		blockReturnType    types.Type
		ReturnType         types.Type
	}{
		{
			input: `def foo(&blk)
							 blk.call(10)
						 end
						 foo() do |x| 
						   square = x * x
							 "#{square}"
						 end`,
			blockArgumentTypes: map[string]types.Type{"x": types.IntType},
			blockReturnType:    types.StringType,
			ReturnType:         types.StringType,
		},
		{
			input: `def foo(x, y, &blk)
							 x * blk.call(y)
						 end
						 foo(7, 8) do |b| 
						   b / 10.0
						 end`,
			blockArgumentTypes: map[string]types.Type{"b": types.IntType},
			blockReturnType:    types.FloatType,
			ReturnType:         types.FloatType,
		},
	}

	for i, tt := range tests {
		func(i int, tt struct {
			input              string
			blockArgumentTypes map[string]types.Type
			blockReturnType    types.Type
			ReturnType         types.Type
		}) {
			defer func() {
				if v := recover(); v != nil {
					t.Errorf("[Test case %d] Encountered panic in processing `%s`:\n%s", i+i, tt.input, debug.Stack())
				}
			}()
			if caseNum == 0 || caseNum == i+1 {
				program, err := ParseString(tt.input)
				if err != nil {
					t.Fatalf("[Test Case %d] %s", i+1, err)
				}
				method, ok := program.GetMethod("foo")
				if !ok {
					t.Fatalf("Could not find method '%s'", "foo")
				}
				for j := 0; j < len(tt.blockArgumentTypes); j++ {
					param, _ := method.Block.GetParam(j)
					if param != method.Block.GetParamByName(param.Name) {
						t.Errorf("[Test Case %d] positional vs optional arg access differs for parameter '%s'", i+1, param.Name)
						break
					}
					expectedType := tt.blockArgumentTypes[param.Name]
					if param.Type() != expectedType {
						t.Errorf("[Test Case %d] type inference failed for parameter '%s': expected %s, but got %s", i+1, param.Name, expectedType, param.Type())
						break
					}
				}
				if !method.ReturnType().Equals(tt.ReturnType) {
					t.Errorf("[Test Case %d] type inference failed for return type for method '%s': expected %s, got %s", i+1, method.Name, tt.ReturnType, method.ReturnType())
				}
			}
		}(i, tt)
	}
}

//TODO need a test that locals are not reassigned with a new type

func TestMethodParamInferenceErrors(t *testing.T) {
	tests := []struct {
		input         string
		expectedError string
	}{
		{`def foo(bar, bar)
		    bar + baz
		  end
			foo(1, 2)`, "line 1: parameter 'bar' declared twice"},
		{`def foo(bar, baz)
		    bar + baz
			end
			foo(1, 2, 3)`, "line 4: method 'foo' called with 3 arguments but 2 expected"},
		{`def foo(bar, baz)
		    bar + baz
			end
			foo(1)`, "line 4: method 'foo' called with 1 positional arguments but 2 expected"},
		{`def foo(bar, baz)
		    bar + baz
			end
			foo(1, 2)
			foo(true, 2)`, "line 5: method 'foo' called with BoolType for parameter 'bar' but 'bar' was previously seen as IntType"},
		{`def foo(bar, baz)
		    bar - baz
			end`, "line 1: unable to detect type signature of method 'foo' because it is never called"},
		{`def foo(bar, baz)
		    if bar == baz
				  true
				else
				  7
				end
			end
			foo(1, 2)`, "line 2: Different branches of conditional returned different types: (if (bar == baz) true (else 7))"},
		{`def foo(bar, baz)
		    [bar, baz]
			end
			foo(1, true)`, "line 2: Heterogenous array membership detected adding BoolType"},
		{`def foo(bar)
		    bar[0] = true
			end
			foo([1, 2])`, "line 2: Attempted to assign BoolType member to Array(IntType)"},
		{`def foo(bar, baz)
		    bar << baz
			end
			foo([1, 2], true)`, "line 2: Tried to append BoolType to Array(IntType)"},
		{`def foo(bar, baz)
		    quux = 7
			  if baz then return bar end
				return quux
			end
			foo([1, 2], true)`, "line 3: Detected conflicting return types IntType and Array(IntType) in method 'foo'"},
		{`def foo(bar)
		    bar[1] 
			end
			foo(true)`, "line 2: BoolType is not a supported type for bracket access"},
		{`def foo(bar, baz)
		    bar[1..baz] 
			end
			foo([1,2,3,4,5], true)`, "line 2: Tried to construct range from disparate types IntType and BoolType"},
		{`def foo(bar, baz)
		    case baz
				when true then "string"
				when false then 10
				end
			end
			foo([1,2,3,4,5], true)`, "line 5: Case statement branches return conflicting types StringType and IntType"},
		{`def foo(bar, baz)
			  Math::PI * bar * baz
			end
			foo(2, 2.5)`, "line 2: No such class 'Math'"},
		{`class Math
		    E = 2.718
		  end

		  def foo(bar, baz)
			  Math::PI * bar * baz
			end
			foo(2, 2.5)`, "line 6: Class 'Math' has no constant 'PI'"},
	}

	for i, tt := range tests {
		if caseNum == 0 || caseNum == i+1 {
			p, err := ParseString(tt.input)
			if err == nil {
				t.Errorf("[Test Case %d] Expected error `%s` but got none", i+1, tt.expectedError)
			} else if tt.expectedError != err.Error() {
				t.Errorf("[Test Case %d] Expected error `%s` but got `%s`", i+1, tt.expectedError, err)
				fmt.Println(p.Errors)
			}
		}
	}
}

func TestTypesAssigned(t *testing.T) {
	input := `# some basic stuff about how this works
def foo(bar) 
		bar.map do |x| 
		    x % 2 == 0 # tag
			end
		end
		# explanation of method call
    foo([1,2,3,4,5])
`

	p, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	m := p.Objects[0].(*Method)
	c := m.Body.Statements[0].(*ReturnNode).Val[0].(*MethodCall)
	expectedType := types.NewArray(types.IntType)
	if c.Receiver.Type() != expectedType {
		t.Fatal("Still not getting type set on the IdentNode")
	}
	blockExpr := c.Block.Body.Statements[0].(*ReturnNode).Val[0]
	if blockExpr.LineNo() != 4 {
		t.Fatalf("expected method to have line number 4, got %d", blockExpr.LineNo())
	}
	if m.LineNo() != 2 {
		t.Fatalf("expected method to have line number 2, got %d", m.LineNo())
	}
	if len(p.Comments) != 3 {
		t.Fatal("Expected 3 comments")
	}
	comments := []struct {
		txt    string
		lineNo int
	}{
		{"# some basic stuff about how this works", 1},
		{"# tag", 4},
		{"# explanation of method call", 7},
	}
	for _, c := range comments {
		comment := p.Comments[c.lineNo]
		if c.txt != comment.Text {
			t.Fatalf("Comment on line %d did not match. Expected '%s', got '%s'", c.lineNo, c.txt, comment.Text)
		}
	}
}
func TestGauntlet(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`
gauntlet("foo") do
  [1, 2, 3].each do |x|
	  [:x, :y, :z].each do |y|
			puts "#{x} #{y}"
		end
	end
end`,
			[]string{`[1, 2, 3].each do |x|
	  [:x, :y, :z].each do |y|
			puts "#{x} #{y}"
		end
	end`,
			},
		},
		{`gauntlet("foo") do
		  puts 10 * 100
		end`,
			[]string{`puts 10 * 100`}},
		{`gauntlet("foo") do
		  x = 30
			y = 40

		  puts x ** y
		end`,
			[]string{`x = 30
			y = 40

		  puts x ** y`}},
		{`gauntlet("simple class, no attrs") do
  class Foo
    def swap(dot_separated)
      dot_separated.gsub(/(\w+)\.(\w+)/, '\2.\1')
    end
  end
  puts Foo.new.swap("left.right")
end`,
			[]string{`class Foo
    def swap(dot_separated)
      dot_separated.gsub(/(\w+)\.(\w+)/, '\2.\1')
    end
  end
  puts Foo.new.swap("left.right")`}},
		{`gauntlet("foo") do
		  puts 10 * 100
		end
		gauntlet("bar") do
		  puts "this is a whatever"
		end`,
			[]string{`puts 10 * 100`, `puts "this is a whatever"`}},
	}

	for i, tt := range tests {
		if caseNum == 0 || caseNum == i+1 {
			p, _ := ParseString(tt.input)
			calls := p.CurrentMethodSet().Calls["gauntlet"]
			if len(calls) == 0 {
				t.Errorf("[%d] : No calls made to gauntlet", i+1)
				break
			}
			for j, call := range calls {
				raw := strings.TrimSpace(call.RawBlock)
				if raw != tt.expected[j] {
					t.Errorf("[%d] : Expected raw source of block to be '%s' but got '%s'", i+1, tt.expected[j], raw)
				}
			}
		}
	}
}

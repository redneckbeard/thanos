package parser

import (
	"testing"
)

func TestLexer(t *testing.T) {
	input := `ActiveRecord::Base : :foo $global 34 55.5 == => = if 
  += -= *= x /= %= ! * x / % & && | ||
  < > <= >= << <=> ,;.)+{}-]?
  definitely def self end then else unless true false 
  return nil module class do yield begin rescue while
  ensure elsif case when until for break next super alias 
  @foo @@bar != ** =~ !~ >> :baz? mutate!( under_score[ | -10 key: [ ( foo2
  `

	tests := []struct {
		expectedType    int
		expectedLiteral string
	}{
		{CONSTANT, "ActiveRecord"},
		{SCOPE, "::"},
		{CONSTANT, "Base"},
		{COLON, ":"},
		{SYMBOL, ":foo"},
		{GVAR, "$global"},
		{INT, "34"},
		{FLOAT, "55.5"},
		{EQ, "=="},
		{HASHROCKET, "=>"},
		{ASSIGN, "="},
		{IF, "if"},
		{NEWLINE, "\n"},
		{ADDASSIGN, "+="},
		{SUBASSIGN, "-="},
		{MULASSIGN, "*="},
		{IDENT, "x"},
		{DIVASSIGN, "/="},
		{MODASSIGN, "%="},
		{BANG, "!"},
		{ASTERISK, "*"},
		{IDENT, "x"},
		{SLASH, "/"},
		{MODULO, "%"},
		{AND, "&"},
		{LOGICALAND, "&&"},
		{PIPE, "|"},
		{LOGICALOR, "||"},
		{NEWLINE, "\n"},
		{LT, "<"},
		{GT, ">"},
		{LTE, "<="},
		{GTE, ">="},
		{LSHIFT, "<<"},
		{SPACESHIP, "<=>"},
		{COMMA, ","},
		{SEMICOLON, ";"},
		{DOT, "."},
		{RPAREN, ")"},
		{PLUS, "+"},
		{LBRACE, "{"},
		{RBRACE, "}"},
		{MINUS, "-"},
		{RBRACKET, "]"},
		{QMARK, "?"},
		{NEWLINE, "\n"},
		{IDENT, "definitely"},
		{DEF, "def"},
		{SELF, "self"},
		{END, "end"},
		{THEN, "then"},
		{ELSE, "else"},
		{UNLESS, "unless"},
		{TRUE, "true"},
		{FALSE, "false"},
		{NEWLINE, "\n"},
		{RETURN, "return"},
		{NIL, "nil"},
		{MODULE, "module"},
		{CLASS, "class"},
		{DO, "do"},
		{YIELD, "yield"},
		{BEGIN, "begin"},
		{RESCUE, "rescue"},
		{WHILE, "while"},
		{NEWLINE, "\n"},
		{ENSURE, "ensure"},
		{ELSIF, "elsif"},
		{CASE, "case"},
		{WHEN, "when"},
		{UNTIL, "until"},
		{FOR, "for"},
		{BREAK, "break"},
		{NEXT, "next"},
		{SUPER, "super"},
		{ALIAS, "alias"},
		{NEWLINE, "\n"},
		{IVAR, "@foo"},
		{CVAR, "@@bar"},
		{NEQ, "!="},
		{POW, "**"},
		{MATCH, "=~"},
		{NOTMATCH, "!~"},
		{RSHIFT, ">>"},
		{SYMBOL, ":baz?"},
		{METHODIDENT, "mutate!"},
		{LPAREN, "("},
		{IDENT, "under_score"},
		{LBRACKET, "["},
		{PIPE, "|"},
		{UNARY_NUM, "-"},
		{INT, "10"},
		{LABEL, "key:"},
		{LBRACKETSTART, "["},
		{LPARENSTART, "("},
		{IDENT, "foo2"},
	}

	l := NewLexer([]byte(input))

	for i, tt := range tests {
		token := <-l.stream

		if token.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tokenNames[tt.expectedType], tokenNames[token.Type])
		}

		if token.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, token.Literal)
		}
	}
}

func TestStatefulLexing(t *testing.T) {
	tests := []struct {
		input    string
		tokens   []int
		literals []string
	}{
		{
			`""`,
			[]int{STRINGBEG, STRINGEND},
			[]string{`"`, `"`},
		},
		{
			`"foo"`,
			[]int{STRINGBEG, STRINGBODY, STRINGEND},
			[]string{`"`, "foo", `"`},
		},
		{
			`"#{foo}"`,
			[]int{STRINGBEG, INTERPBEG, IDENT, INTERPEND, STRINGEND},
			[]string{`"`, "#{", "foo", "}", `"`},
		},
		{
			`"foo#{bar}"`,
			[]int{STRINGBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGEND},
			[]string{`"`, "foo", "#{", "bar", "}", `"`},
		},
		{
			`"#{bar}foo"`,
			[]int{STRINGBEG, INTERPBEG, IDENT, INTERPEND, STRINGBODY, STRINGEND},
			[]string{`"`, "#{", "bar", "}", "foo", `"`},
		},
		{
			`"#{foo["bar#{baz}"]}"`,
			[]int{STRINGBEG, INTERPBEG, IDENT, LBRACKET, STRINGBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGEND, RBRACKET, INTERPEND, STRINGEND},
			[]string{`"`, "#{", "foo", "[", `"`, "bar", "#{", "baz", "}", `"`, "]", "}", `"`},
		},
		{
			`"#{{foo: "bar"}}"`,
			[]int{STRINGBEG, INTERPBEG, LBRACE, LABEL, STRINGBEG, STRINGBODY, STRINGEND, RBRACE, INTERPEND, STRINGEND},
			[]string{`"`, "#{", "{", "foo:", `"`, "bar", `"`, "}", "}", `"`},
		},
		{
			`"foo#{bar}baz#{quux}"`,
			[]int{STRINGBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGEND},
			[]string{`"`, "foo", "#{", "bar", "}", "baz", "#{", "quux", "}", `"`},
		},
		{
			`# this is a preceding comment
			"foo#{bar}baz#{quux}" # this is an inline comment
			# this is a trailing comment`,
			[]int{COMMENT, STRINGBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGEND, COMMENT, COMMENT},
			[]string{"# this is a preceding comment\n", `"`, "foo", "#{", "bar", "}", "baz", "#{", "quux", "}", `"`, "# this is an inline comment\n", `# this is a trailing comment`},
		},
		{
			`0...3`,
			[]int{INT, DOT3, INT},
			[]string{`0`, "...", "3"},
		},
		{
			`0..3`,
			[]int{INT, DOT2, INT},
			[]string{`0`, "..", "3"},
		},
		{
			`0...`,
			[]int{INT, DOT3},
			[]string{`0`, "..."},
		},
		{
			`0..`,
			[]int{INT, DOT2},
			[]string{`0`, ".."},
		},
		{
			`/foo/`,
			[]int{REGEXBEG, STRINGBODY, REGEXEND},
			[]string{`/`, "foo", "/"},
		},
		{
			`10 /foo/`,
			[]int{INT, SLASH, IDENT, SLASH},
			[]string{`10`, `/`, "foo", "/"},
		},
		{
			`/foo#{bar}/`,
			[]int{REGEXBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, REGEXEND},
			[]string{`/`, "foo", "#{", "bar", "}", `/`},
		},
		{
			`/foo#{bar}(\w+)/`,
			[]int{REGEXBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGBODY, REGEXEND},
			[]string{`/`, "foo", "#{", "bar", "}", `(\w+)`, `/`},
		},
		{
			`'foo#{bar}'`,
			[]int{RAWSTRINGBEG, STRINGBODY, RAWSTRINGEND},
			[]string{`'`, "foo#{bar}", `'`},
		},
		{
			`%w{foo bar baz}`,
			[]int{RAWWORDSBEG, STRINGBODY, RAWSTRINGEND},
			[]string{`%w{`, "foo bar baz", `}`},
		},
		{
			`%w$foo bar baz$`,
			[]int{RAWWORDSBEG, STRINGBODY, RAWSTRINGEND},
			[]string{`%w$`, "foo bar baz", `$`},
		},
		{
			`%W|foo #{bar} baz|`,
			[]int{WORDSBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGBODY, STRINGEND},
			[]string{`%W|`, "foo ", "#{", "bar", "}", " baz", `|`},
		},
		{
			`%W{foo #{bar} baz}`,
			[]int{WORDSBEG, STRINGBODY, INTERPBEG, IDENT, INTERPEND, STRINGBODY, STRINGEND},
			[]string{`%W{`, "foo ", "#{", "bar", "}", " baz", `}`},
		},
		{
			`%W{foo #{%w{b a r}} baz}`,
			[]int{WORDSBEG, STRINGBODY, INTERPBEG, RAWWORDSBEG, STRINGBODY, RAWSTRINGEND, INTERPEND, STRINGBODY, STRINGEND},
			[]string{`%W{`, "foo ", "#{", "%w{", "b a r", "}", "}", " baz", `}`},
		},
		{
			`5.even?`,
			[]int{INT, DOT, METHODIDENT},
			[]string{"5", ".", "even?"},
		},
		{
			`-5.0.positive?`,
			[]int{UNARY_NUM, FLOAT, DOT, METHODIDENT},
			[]string{"-", "5.0", ".", "positive?"},
		},
		{
			`puts []`,
			[]int{IDENT, LBRACKETSTART, RBRACKET},
			[]string{"puts", "[", "]"},
		},
		{
			`puts([])`,
			[]int{IDENT, LPAREN, LBRACKETSTART, RBRACKET, RPAREN},
			[]string{"puts", "(", "[", "]", ")"},
		},
		{
			`[1]`,
			[]int{LBRACKETSTART, INT, RBRACKET},
			[]string{"[", "1", "]"},
		},
		{
			`(x)`,
			[]int{LPARENSTART, IDENT, RPAREN},
			[]string{"(", "x", ")"},
		},
		{
			`puts (x)`,
			[]int{IDENT, LPARENSTART, IDENT, RPAREN},
			[]string{"puts", "(", "x", ")"},
		},
		{
			`puts(x)`,
			[]int{IDENT, LPAREN, IDENT, RPAREN},
			[]string{"puts", "(", "x", ")"},
		},
		{
			`return x if bar`,
			[]int{RETURN, IDENT, IF_MOD, IDENT},
			[]string{"return", "x", "if", "bar"},
		},
		{
			`@foo[x]`,
			[]int{IVAR, LBRACKET, IDENT, RBRACKET},
			[]string{"@foo", "[", "x", "]"},
		},
		{
			"`man -P cat #{\"date\"}`",
			[]int{XSTRINGBEG, STRINGBODY, INTERPBEG, STRINGBEG, STRINGBODY, STRINGEND, INTERPEND, STRINGEND},
			[]string{"`", "man -P cat ", "#{", `"`, "date", `"`, "}", "`"},
		},
		{
			`"\""`,
			[]int{STRINGBEG, STRINGBODY, STRINGEND},
			[]string{`"`, `\"`, `"`},
		},
		{
			`'\''`,
			[]int{RAWSTRINGBEG, STRINGBODY, RAWSTRINGEND},
			[]string{`'`, `\'`, `'`},
		},
		{
			`%w{foo bar \}baz}`,
			[]int{RAWWORDSBEG, STRINGBODY, RAWSTRINGEND},
			[]string{`%w{`, `foo bar \}baz`, `}`},
		},
		{
			`[
			1,
			2
			]`,
			[]int{LBRACKETSTART, INT, COMMA, INT, NEWLINE, RBRACKET},
			[]string{"[", "1", ",", "2", "\n", "]"},
		},
		{`foo(
		1,
		2
		)`,
			[]int{IDENT, LPAREN, INT, COMMA, INT, NEWLINE, RPAREN},
			[]string{"foo", "(", "1", ",", "2", "\n", ")"},
		},
	}

	for i, tt := range tests {
		if caseNum == 0 || caseNum == i+1 {
			l := NewLexer([]byte(tt.input))

			for j := 0; j < len(tt.tokens); j++ {
				token := <-l.stream

				if token.Type != tt.tokens[j] {
					t.Errorf("tests[%d] - token %d type wrong. expected=%q, got=%q",
						i+1, j, tokenNames[tt.tokens[j]], tokenNames[token.Type])
				}

				if token.Literal != tt.literals[j] {
					t.Errorf("tests[%d] - token %d literal wrong. expected=%q, got=%q",
						i+1, j, tt.literals[j], token.Literal)
				}
			}
		}
	}
}

package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

var currentLineNo = 0

var Illegal int = 99999999

var validPuncts = map[string]int{
	"(":   LPAREN,
	")":   RPAREN,
	".":   DOT,
	"..":  DOT2,
	"...": DOT3,
	";":   SEMICOLON,
	"?":   QMARK,
	"[":   LBRACKET,
	"]":   RBRACKET,
	"{":   LBRACEBLOCK,
	"}":   RBRACE,
	",":   COMMA,
	"!":   BANG,
	"!=":  NEQ,
	"!~":  NOTMATCH,
	"%":   MODULO,
	"%=":  MODASSIGN,
	"&&":  LOGICALAND,
	"&":   AND,
	"*":   ASTERISK,
	"**":  POW,
	"*=":  MULASSIGN,
	"+":   PLUS,
	"+=":  ADDASSIGN,
	"-":   MINUS,
	"-=":  SUBASSIGN,
	"/":   SLASH,
	"/=":  DIVASSIGN,
	":":   COLON,
	"::":  SCOPE,
	"<":   LT,
	"<<":  LSHIFT,
	"<<=": LSHIFTASSIGN,
	"<=":  LTE,
	"<=>": SPACESHIP,
	"=":   ASSIGN,
	"==":  EQ,
	"=>":  HASHROCKET,
	"=~":  MATCH,
	">":   GT,
	">=":  GTE,
	">>":  RSHIFT,
	">>=": RSHIFTASSIGN,
	"|":   PIPE,
	"||":  LOGICALOR,
}

var keywords = map[string]int{
	"alias":    ALIAS,
	"begin":    BEGIN,
	"break":    BREAK,
	"case":     CASE,
	"class":    CLASS,
	"def":      DEF,
	"do":       DO,
	"else":     ELSE,
	"elsif":    ELSIF,
	"end":      END,
	"ensure":   ENSURE,
	"false":    FALSE,
	"for":      FOR,
	"gauntlet": IDENT,
	"if":       IF,
	"module":   MODULE,
	"next":     NEXT,
	"nil":      NIL,
	// Neither private nor protected are keywords in Ruby but are rather methods.
	// However, they function effectively as keywords, and because of how Method
	// structs are being constructed here, it's much simpler to simply flip a bit
	// to change state on the class when we encounter either than to record the
	// order of method definition relative to the position of any seen
	// `private`/`protected` method calls.
	"private":   PRIVATE,
	"protected": PROTECTED,
	"rescue":    RESCUE,
	"return":    RETURN,
	"self":      SELF,
	"super":     SUPER,
	"then":      THEN,
	"true":      TRUE,
	"unless":    UNLESS,
	"until":     UNTIL,
	"when":      WHEN,
	"while":     WHILE,
	"yield":     YIELD,
}

type LexState int

const (
	Clear LexState = iota
	InSymbol
	InGlobal
	InFloat
	InNumber
	InCVar
	InIVar
	InPunct
	InInterpString
	InInterp
	InRawString
	InRegex
	OpenBrace
	InEscapeSequence
)

type Token struct {
	Type     int
	Literal  string
	LineNo   int
	RawBlock string
}

func (t Token) String() string {
	return fmt.Sprintf("%s[%q]", tokenNames[t.Type], t.Literal)
}

type Lexer struct {
	*bytes.Buffer
	Root                   *Root
	lineNo                 int
	stream                 chan Token
	read                   []rune
	State                  *Stack[LexState]
	lastToken              int
	rawSource              []rune
	lastParsedToken        Token
	gauntlet               bool
	spaceConsumed          bool
	StringDelim            *Stack[rune]
	resetExpr, skipNewline bool
	cond, cmdArg           *StackState
	condStack, cmdArgStack *Stack[*StackState]
}

func NewLexer(buf []byte) *Lexer {
	l := &Lexer{
		Buffer:        bytes.NewBuffer(buf),
		lineNo:        1,
		stream:        make(chan Token),
		Root:          NewRoot(),
		State:         &Stack[LexState]{},
		StringDelim:   &Stack[rune]{},
		cond:          NewStackState("cond"),
		cmdArg:        NewStackState("cmdarg"),
		spaceConsumed: true,
	}
	go l.Tokenize()
	return l
}

func (l *Lexer) Lex(lval *yySymType) int {
	token := <-l.stream
	lval.str = token.Literal
	currentLineNo = token.LineNo
	l.lastParsedToken = token
	return token.Type
}

func (l *Lexer) Error(e string) {
	l.Root.AddError(errors.New(e))
}

func (l *Lexer) Peek() (rune, error) {
	b, _, err := l.ReadRune()
	l.UnreadRune()
	return b, err
}

func (l *Lexer) Read() (rune, error) {
	b, _, err := l.ReadRune()
	return b, err
}

func (l *Lexer) Advance() (rune, rune, error) {
	chr, _ := l.Read()
	l.read = append(l.read, chr)
	next, err := l.Peek()
	return chr, next, err
}

func (l *Lexer) RewindBuffer() {
	l.read = l.read[:len(l.read)-1]
}

func (l *Lexer) ResetBuffer() {
	l.read = []rune{}
}

func (l *Lexer) Emit(t int) {
	l.resetExpr = false
	if l.gauntlet {
		l.rawSource = append(l.rawSource, l.read...)
	} else {
		l.rawSource = []rune{}
	}
	tok := Token{t, string(l.read), l.lineNo, string(l.rawSource)}
	l.stream <- tok
	l.lastToken = t
	l.skipNewline = false
	l.ResetBuffer()
	l.spaceConsumed = false
}

func (l *Lexer) AtExprStart() bool {
	if l.resetExpr {
		return true
	}
	midExprTokens := []int{
		NIL, SYMBOL, STRING, INT, FLOAT, TRUE, FALSE, DEF, END, SELF, CONSTANT,
		IVAR, CVAR, GVAR, METHODIDENT, IDENT, DO,
		RBRACE, STRINGEND, RBRACKET, RPAREN,
	}

	for _, tok := range midExprTokens {
		if l.lastToken == tok {
			return false
		}
	}

	return true
}

func (l *Lexer) pushStringDelim(delim rune) {
	switch delim {
	case '{':
		delim = '}'
	case '[':
		delim = ']'
	case '(':
		delim = ')'
	}
	l.StringDelim.Push(delim)
}

func (l *Lexer) Tokenize() {
	chr, err := l.Peek()
	for err == nil {
		switch {
		case l.State.Peek() == InInterpString:
			err = l.lexString()
		case l.State.Peek() == InRegex:
			err = l.lexRegex()
		case unicode.IsSpace(chr):
			err = l.lexWhitespace()
		case unicode.IsPunct(chr) || unicode.IsSymbol(chr):
			err = l.lexPunct()
		case unicode.IsLetter(chr) || chr == '_':
			err = l.lexWord()
		case unicode.IsDigit(chr):
			err = l.lexNumber()
		default:
			panic("Unknown Unicode character category encountered")
		}
		chr, err = l.Peek()
	}
	close(l.stream)
}

func (l *Lexer) lexWhitespace() error {
	chr, _, err := l.Advance()
	if chr == '\n' {
		if !l.skipNewline {
			l.Emit(NEWLINE)
		}
		l.lineNo++
	} else {
		if l.gauntlet {
			l.rawSource = append(l.rawSource, l.read...)
		}
		l.ResetBuffer()
	}
	l.spaceConsumed = true
	return err
}

func (l *Lexer) lexNumber() error {
	_, next, err := l.Advance()
	for err == nil && unicode.IsDigit(next) {
		_, next, err = l.Advance()
	}
	// If we've already seen a decimal point, we've come back. Any further
	// punctuation belongs to something else.
	if l.State.Peek() == InFloat {
		l.Emit(FLOAT)
		return err
	}
	next, err = l.Peek()
	if err != nil && err != io.EOF {
		return err
	}
	// If we see a decimal point, we hand over lexing to lexPunct. If we're looking
	// at a float, we'll end up back above in the correct state; if not, we'll emit
	// the int from there.
	if next == '.' {
		l.State.Push(InNumber)
		defer l.State.Pop()
		return l.lexPunct()
	}

	l.Emit(INT)
	return err
}

func (l *Lexer) lexPunct() error {
	curr, next, err := l.Advance()

	if curr == '.' && l.State.Peek() == InNumber {
		if unicode.IsDigit(next) {
			l.State.Push(InFloat)
			defer l.State.Pop()
			return l.lexNumber()
		} else {
			l.RewindBuffer()
			l.Emit(INT)
			l.State.Pop()
			l.read = []rune{'.'}
		}
	}

	// Defer to other lexing methods when the token is prefixed with non-alpha but
	if unicode.IsLetter(next) {
		switch curr {
		case ':':
			return l.lexSymbol()
		case '$':
			return l.lexGlobal()
		case '@':
			return l.lexAttribute()
		case '%':
			if l.AtExprStart() {
				return l.lexPercentLiteral()
			}
		}
	}

	switch curr {
	case '@':
		if next == '@' {
			return l.lexAttribute()
		}
	case '"', '`':
		if l.State.Peek() == InInterpString {
			l.State.Pop()
			l.Emit(STRINGEND)
			return err
		} else {
			if curr == '`' {
				l.pushStringDelim('`')
				l.State.Push(InInterpString)
				l.Emit(XSTRINGBEG)
			}
			return l.lexString()
		}
	case '\'':
		return l.lexRawString()
	case ',':
		l.Emit(COMMA)
		l.skipNewline = true
		return err
	case '}':
		var exitedState bool
		switch l.State.Peek() {
		case InInterp:
			exitedState = true
			l.Emit(INTERPEND)
		case OpenBrace:
			exitedState = true
			l.Emit(RBRACE)
		}
		if exitedState {
			l.State.Pop()
		}
		return err
	case '{':
		l.State.Push(OpenBrace)
		l.emitOpenMatching(LBRACEBLOCK)
		l.skipNewline = true
		return err
	case '#':
		return l.lexComment()
	case '/':
		if l.State.Peek() == InRegex {
			l.State.Pop()
			l.Emit(REGEXEND)
			return err
		}
		switch l.lastToken {
		case RBRACKET, RBRACE, RPAREN, INT, FLOAT, IDENT, CONSTANT, METHODIDENT:
			// keep going
		default:
			return l.lexRegex()
		}
	}

	for err == nil && (unicode.IsPunct(next) || unicode.IsSymbol(next)) {
		next, err = l.Peek()
		nextPunct := string(append(l.read, next))
		tok, currDefined := validPuncts[string(l.read)]
		_, nextDefined := validPuncts[nextPunct]
		if currDefined && !nextDefined {
			switch tok {
			case PLUS, MINUS:
				if l.AtExprStart() {
					l.Emit(UNARY_NUM)
					return err
				}
			case LPAREN, LBRACKET:
				if l.spaceConsumed {
					l.Emit(exprStartTokens[tok])
					l.skipNewline = true
					return err
				}
				l.emitOpenMatching(tok)
				l.skipNewline = true
				return err
			}

			l.Emit(tok)
			return nil
		}
		curr, next, err = l.Advance()
	}
	if err != nil && err != io.EOF {
		return err
	}
	if _, defined := validPuncts[string(l.read)]; !defined {
		l.Emit(Illegal)
		return err
	}
	validTok := validPuncts[string(l.read)]
	switch validTok {
	case PLUS, MINUS:
		if l.AtExprStart() {
			l.Emit(UNARY_NUM)
			return err
		}
	case LPAREN, LBRACKET:
		if l.spaceConsumed {
			l.Emit(exprStartTokens[validTok])
			l.skipNewline = true
			return err
		}
		l.emitOpenMatching(validTok)
		l.skipNewline = true
		return err
	}
	l.Emit(validTok)
	return err
}

func (l *Lexer) emitOpenMatching(tok int) {
	switch l.lastToken {
	case RPAREN, RBRACKET, RBRACE, IDENT, CONSTANT, METHODIDENT, YIELD, IVAR, STRINGEND, RAWSTRINGEND, SUPER:
		l.Emit(tok)
	default:
		l.Emit(exprStartTokens[tok])
	}
}

func (l *Lexer) lexSymbol() error {
	l.State.Push(InSymbol)
	defer l.State.Pop()
	return l.lexWord()
}

func (l *Lexer) lexGlobal() error {
	l.State.Push(InGlobal)
	defer l.State.Pop()
	return l.lexWord()
}

func (l *Lexer) lexWord() error {
	_, next, err := l.Advance()
	for err == nil && (unicode.IsLetter(next) || unicode.IsDigit(next) || next == '_') {
		_, next, err = l.Advance()
	}
	if err != nil && err != io.EOF {
		return err
	}
	if p, defined := keywords[string(l.read)]; !defined {
		if unicode.IsUpper(l.read[0]) {
			l.Emit(CONSTANT)
		} else {
			switch l.State.Peek() {
			case InSymbol:
				next, err = l.Peek()
				if next == '!' || next == '?' {
					l.Advance()
				}
				l.Emit(SYMBOL)
			case InGlobal:
				l.Emit(GVAR)
			case InIVar:
				l.Emit(IVAR)
			case InCVar:
				l.Emit(CVAR)
			default:
				next, err = l.Peek()
				if next == '!' || next == '?' {
					l.Advance()
					l.Emit(METHODIDENT)
				} else if next == ':' {
					l.Advance()
					l.Emit(LABEL)
				} else {
					l.Emit(IDENT)
				}
			}
		}
	} else {
		switch string(l.read) {
		case "do":
			if l.cond.IsActive() {
				l.Emit(DO_COND)
			} else if l.cmdArg.IsActive() { // || l.atExprStart() ??
				l.Emit(DO_BLOCK)
			} else {
				l.Emit(p)
			}

		case "if", "unless", "while", "until", "rescue":
			if !l.AtExprStart() {
				l.Emit(keywordModifierTokens[p])
				break
			}
			fallthrough
		// Hack to support rspec-lite syntax for tests. Goal is to capture the
		// raw source of the block argument to the method call. Current design
		// of the lexer means that putting this hack in the parser has to
		// contend with concurrency so it's easy to manage all the state here
		// and just include that string in the token wrapper.
		case "gauntlet":
			l.gauntlet = true
			l.Emit(p)
		default:
			l.Emit(p)
		}
	}
	return err
}

func (l *Lexer) lexAttribute() error {
	curr, _, err := l.Advance()
	if curr == '@' {
		if err != nil {
			return err
		}
		l.State.Push(InCVar)
		defer l.State.Pop()
		return l.lexWord()
	}
	l.State.Push(InIVar)
	defer l.State.Pop()
	return l.lexWord()
}

func (l *Lexer) lexComment() error {
	_, next, err := l.Advance()
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		if next == '\n' {
			break
		}
		if lastLoop {
			break
		}
		_, next, err = l.Advance()
	}
	l.Emit(COMMENT)
	return err
}

func (l *Lexer) lexRegex() error {
	if l.State.Peek() != InRegex {
		l.Emit(REGEXBEG)
		l.State.Push(InRegex)
	}

	curr, next, err := l.Advance()
	if curr == '/' {
		l.State.Pop()
		l.Emit(REGEXEND)
		return err
	}
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		switch {
		case curr == '/':
			l.State.Pop()
			l.Emit(REGEXEND)
		case next == '/':
			if len(l.read) > 0 {
				l.Emit(STRINGBODY)
			}
			l.Advance()
			l.State.Pop()
			l.Emit(REGEXEND)
			return err
		case curr == '#' && next == '{':
			if len(l.read) > 1 {
				l.RewindBuffer()
				l.Emit(STRINGBODY)
				l.read = []rune{'#'}
			}
			l.Advance()
			l.State.Push(InInterp)
			l.Emit(INTERPBEG)
			return err
		}
		if lastLoop {
			break
		}
		curr, next, err = l.Advance()
	}
	return err
}

func (l *Lexer) lexString() error {
	if l.State.Peek() != InInterpString {
		l.Emit(STRINGBEG)
		l.State.Push(InInterpString)
		l.pushStringDelim('"')
	}
	curr, next, err := l.Advance()
	// empty string, should be handled better
	if curr == l.StringDelim.Peek() {
		l.State.Pop()
		l.StringDelim.Pop()
		l.Emit(STRINGEND)
		return err
	}
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		switch {
		case curr == '\\':
			l.Advance()
		case curr == l.StringDelim.Peek():
			if len(l.read) > 1 {
				l.RewindBuffer()
				l.Emit(STRINGBODY)
				l.read = []rune{l.StringDelim.Peek()}
			}
			l.State.Pop()
			l.Emit(STRINGEND)
			l.StringDelim.Pop()
			return err
		case next == l.StringDelim.Peek():
			if len(l.read) > 0 {
				l.Emit(STRINGBODY)
			}
			l.Advance()
			l.State.Pop()
			l.Emit(STRINGEND)
			l.StringDelim.Pop()
			return err
		case curr == '#' && next == '{':
			if len(l.read) > 1 {
				l.RewindBuffer()
				l.Emit(STRINGBODY)
				l.read = []rune{'#'}
			}
			l.Advance()
			l.State.Push(InInterp)
			l.Emit(INTERPBEG)
			return err
		}
		if lastLoop {
			break
		}
		curr, next, err = l.Advance()
	}
	return err
}

func (l *Lexer) lexRawString() error {
	if l.State.Peek() != InRawString {
		l.Emit(RAWSTRINGBEG)
		l.State.Push(InRawString)
		l.pushStringDelim('\'')
	}
	curr, next, err := l.Advance()
	// empty string, should be handled better
	if curr == l.StringDelim.Peek() {
		l.Emit(RAWSTRINGEND)
		l.State.Pop()
		l.StringDelim.Pop()
		return err
	}
	for {
		var lastLoop bool
		if err == io.EOF {
			lastLoop = true
		}
		switch {
		case curr == '\\':
			curr, next, err = l.Advance()
		case curr == l.StringDelim.Peek():
			if len(l.read) > 1 {
				l.RewindBuffer()
				l.Emit(STRINGBODY)
				l.read = []rune{l.StringDelim.Peek()}
			}
			l.Emit(RAWSTRINGEND)
			l.State.Pop()
			l.StringDelim.Pop()
			return err
		case next == l.StringDelim.Peek():
			if len(l.read) > 0 {
				l.Emit(STRINGBODY)
			}
			l.Advance()
			l.Emit(RAWSTRINGEND)
			l.State.Pop()
			l.StringDelim.Pop()
			return err
		}
		if lastLoop {
			break
		}
		curr, next, err = l.Advance()
	}
	return err
}

func (l *Lexer) lexPercentLiteral() error {
	curr, next, err := l.Advance()
	if err != nil {
		return err
	}
	if !unicode.IsPunct(next) && !unicode.IsSymbol(next) {
		return fmt.Errorf("'%c' is not a valid delimiter for a percent literal", next)
	}
	l.pushStringDelim(next)
	l.Advance()
	switch curr {
	case 'w':
		l.Emit(RAWWORDSBEG)
		l.State.Push(InRawString)
		return l.lexRawString()
	case 'W':
		l.Emit(WORDSBEG)
		l.State.Push(InInterpString)
		return l.lexString()
	case 'x':
		l.Emit(RAWXSTRINGBEG)
		l.State.Push(InInterpString)
		return l.lexRawString()
	case 'X':
		l.Emit(XSTRINGBEG)
		l.State.Push(InInterpString)
		return l.lexString()
	case 'q', 'Q', 'r', 'R', 'i', 'I', 's', 'S':
		return fmt.Errorf("'%%%c' literals are not supported", curr)
	default:
		return fmt.Errorf("'%c' is not a valid type of percent literal", curr)
	}
}

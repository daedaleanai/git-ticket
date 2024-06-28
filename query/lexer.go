package query

import (
	"fmt"
	"strings"
	"unicode"
)

// Represents the type of a token parsed by the lexer
type TokenType string

const (
	// An identifier token is a consecutive series of characters that is not made of one of the reserved characters below, and does not include whitespace
	IdentToken TokenType = "IdentToken"
	// Left parenthesis `(`
	LparenToken = "LparenToken"
	// Right parenthesis `)`
	RparenToken = "RparenToken"
	// Comma `,`
	CommaToken = "CommaToken"
	// double-quoted string `"a string"` may contain any characters between its delimiters.
	StringToken = "StringToken"
	// regex string `r"[a-f0-9]+"`.
	RegexToken = "RegexToken"
	// Represents the end of the token stream
	EofToken = "EofToken"
)

// A Span represents the range location of a token, measured in bytes
type Span struct {
	Begin int
	End   int
}

func (s Span) Extend(other Span) Span {
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	return Span{
		Begin: min(s.Begin, other.Begin),
		End:   max(s.End, other.End),
	}
}

// A Token represents the lexical unit returned by the lexer
type Token struct {
	TokenType TokenType
	Literal   string
	Span      Span
}

// A Lexer allows the user to parse a string into tokens
type Lexer struct {
	query string
	pos   int
}

// NewLexer creates a lexer instance
func NewLexer(query string) Lexer {
	return Lexer{query: query, pos: 0}
}

func newSingleCharacterToken(ty TokenType, literal string, pos int) Token {
	endPos := pos
	if ty != EofToken {
		endPos += 1
	}

	return Token{
		TokenType: ty,
		Literal:   literal,
		Span:      Span{pos, endPos},
	}
}

// NextToken Returns the next token in the stream, or an EofToken if the stream has ended
func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	curChar := l.curChar()
	if simpleTokenType, ok := simpleTokenTypeMap[curChar]; ok {
		token := newSingleCharacterToken(simpleTokenType, string(curChar), l.pos)
		l.advance() // consume the char
		return token, nil
	}

	if curChar == '"' {
		return l.parseStringToken()
	}

	peekChar := l.peekChar()
	if curChar == 'r' && peekChar == '"' {
		return l.parseRegexToken()
	}

	return l.parseIdentToken()
}

func (l *Lexer) curChar() byte {
	if l.pos < len(l.query) {
		return l.query[l.pos]
	}
	// Indicates EOF
	return 0
}

func (l *Lexer) peekChar() byte {
	if l.pos+1 < len(l.query) {
		return l.query[l.pos+1]
	}
	// Indicates EOF
	return 0
}

var simpleTokenTypeMap map[byte]TokenType = map[byte]TokenType{
	'(': LparenToken,
	')': RparenToken,
	',': CommaToken,
	0:   EofToken,
}

func isWhitespace(ch byte) bool {
	return ch == '\n' || ch == '\t' || ch == ' ' || ch == '\r'
}

func isEofChar(ch byte) bool {
	return ch == 0
}

func isKnownToken(ch byte) bool {
	_, known := simpleTokenTypeMap[ch]
	return known || ch == '"'
}

func (l *Lexer) parseIdentToken() (Token, error) {
	begin := l.pos
	for ch := l.curChar(); !isWhitespace(ch) && !isKnownToken(ch) && !isEofChar(ch); ch = l.advance() {
	}
	end := l.pos

	token := Token{
		TokenType: IdentToken,
		Literal:   l.query[begin:end],
		Span: Span{
			Begin: begin,
			End:   end,
		},
	}

	return token, nil
}

type UnterminatedTokenError struct {
	Input string
	Span  Span
}

func (e *UnterminatedTokenError) Error() string {
	return fmt.Sprintf("Unterminated string token: %s", e.Input[e.Span.Begin:e.Span.End])
}

func (l *Lexer) parseStringToken() (Token, error) {
	if l.curChar() != '"' {
		panic("parseStringToken expects the string to start with \"")
	}
	endIdx := strings.Index(l.query[l.pos+1:], "\"")
	if endIdx < 0 {
		return Token{}, &UnterminatedTokenError{l.query, Span{l.pos, len(l.query)}}
	}
	endIdx += l.pos + 1

	token := Token{
		TokenType: StringToken,
		Literal:   l.query[l.pos+1 : endIdx],
		Span: Span{
			Begin: l.pos,
			End:   endIdx + 1,
		},
	}
	l.pos = endIdx + 1
	return token, nil
}

func (l *Lexer) parseRegexToken() (Token, error) {
	if l.curChar() != 'r' || l.peekChar() != '"' {
		panic("parseRegexToken expects the string to start with r\"")
	}
	l.advance()
	token, err := l.parseStringToken()
	token.TokenType = RegexToken
	token.Span.Begin -= 1
	return token, err
}

func (l *Lexer) advance() byte {
	l.pos += 1
	return l.curChar()
}

func (l *Lexer) skipWhitespace() {
	for ch := l.curChar(); isWhitespace(ch); ch = l.advance() {
	}
}

type token struct {
	qualifier string
	value     string
}

// tokenize parse and break a input into tokens ready to be
// interpreted later by a parser to get the semantic.
func tokenize(query string) ([]token, error) {
	fields, err := splitQuery(query)
	if err != nil {
		return nil, err
	}

	var tokens []token
	for _, field := range fields {
		split := strings.SplitN(field, ":", 2)
		if len(split) != 2 {
			return nil, fmt.Errorf("can't tokenize \"%s\"", field)
		}

		if len(split[0]) == 0 {
			return nil, fmt.Errorf("can't tokenize \"%s\": empty qualifier", field)
		}
		if len(split[1]) == 0 {
			return nil, fmt.Errorf("empty value for qualifier \"%s\"", split[0])
		}

		tokens = append(tokens, token{
			qualifier: split[0],
			value:     removeQuote(split[1]),
		})
	}
	return tokens, nil
}

func splitQuery(query string) ([]string, error) {
	lastQuote := rune(0)
	inQuote := false

	isToken := func(r rune) bool {
		switch {
		case !inQuote && isQuote(r):
			lastQuote = r
			inQuote = true
			return true
		case inQuote && r == lastQuote:
			lastQuote = rune(0)
			inQuote = false
			return true
		case inQuote:
			return true
		default:
			return !unicode.IsSpace(r)
		}
	}

	var result []string
	var token strings.Builder
	for _, r := range query {
		if isToken(r) {
			token.WriteRune(r)
		} else {
			if token.Len() > 0 {
				result = append(result, token.String())
				token.Reset()
			}
		}
	}

	if inQuote {
		return nil, fmt.Errorf("unmatched quote")
	}

	if token.Len() > 0 {
		result = append(result, token.String())
	}

	return result, nil
}

func isQuote(r rune) bool {
	return r == '"' || r == '\''
}

func removeQuote(field string) string {
	runes := []rune(field)
	if len(runes) >= 2 {
		r1 := runes[0]
		r2 := runes[len(runes)-1]

		if r1 == r2 && isQuote(r1) {
			return string(runes[1 : len(runes)-1])
		}
	}
	return field
}

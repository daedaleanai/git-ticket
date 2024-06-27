package query

import (
	"fmt"
	"strings"
	"unicode"
)

// Represents the type of a token parsed by the lexer
type TokenType int

const (
	// An identifier token is a consecutive series of characters that is not made of one of the reserved characters below, and does not include whitespace
	IdentToken TokenType = iota
	// Left parenthesis `(`
	LparenToken
	// Right parenthesis `)`
	RparenToken
	// Comma `,`
	CommaToken
	// double-quoted string `"a string"` may contain any characters between its delimiters.
	StringToken
	// Represents the end of the token stream
	EofToken
)

// A Span represents the range location of a token, measured in bytes
type Span struct {
	Begin int
	End   int
}

// A Token represents the unit of returned by the lexer
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

// NextToken Returns the next token in the stream, or an EofToken if the stream has ended
func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	curChar := l.curChar()
	if simpleTokenType, ok := simpleTokenTypeMap[curChar]; ok {
		token := Token{
			TokenType: simpleTokenType,
			Literal:   string(curChar),
			Span:      newSpanAt(l.pos),
		}
		l.advance() // consume the char
		return token, nil
	}

	if curChar == '"' {
		return l.parseStringToken()
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

type UnterminatedStringTokenError struct {
	Input string
	Span  Span
}

func (e *UnterminatedStringTokenError) Error() string {
	return fmt.Sprintf("Unterminated string token: %s", e.Input[e.Span.Begin:e.Span.End])
}

func (l *Lexer) parseStringToken() (Token, error) {
	endIdx := strings.Index(l.query[l.pos+1:], "\"")
	if endIdx < 0 {
		return Token{}, &UnterminatedStringTokenError{l.query, Span{l.pos, len(l.query)}}
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

func (l *Lexer) advance() byte {
	l.pos += 1
	return l.curChar()
}

func (l *Lexer) skipWhitespace() {
	for ch := l.curChar(); isWhitespace(ch); ch = l.advance() {
	}
}

func newSpanAt(pos int) Span {
	return Span{
		Begin: pos,
		End:   pos + 1,
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

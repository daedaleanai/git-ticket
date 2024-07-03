package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	newToken := func(ty TokenType, literal string, begin, end int) Token {
		return Token{
			TokenType: ty,
			Literal:   literal,
			Span: Span{
				Begin: begin,
				End:   end,
			},
		}
	}

	var tests = []struct {
		input  string
		tokens []Token
	}{
		{"\t\n\t \n\r  \t", []Token{}},
		{` ident `, []Token{newToken(IdentToken, "ident", 1, 6)}},
		{`"string"`, []Token{newToken(StringToken, "string", 0, 8)}},
		{
			`add("string")`,
			[]Token{
				newToken(IdentToken, "add", 0, 3),
				newToken(LparenToken, "(", 3, 4),
				newToken(StringToken, "string", 4, 12),
				newToken(RparenToken, ")", 12, 13),
			},
		},
		{
			` add (  "string with spaces", and, more , tokens  )   `,
			[]Token{
				newToken(IdentToken, "add", 1, 4),
				newToken(LparenToken, "(", 5, 6),
				newToken(StringToken, "string with spaces", 8, 28),
				newToken(CommaToken, ",", 28, 29),
				newToken(IdentToken, "and", 30, 33),
				newToken(CommaToken, ",", 33, 34),
				newToken(IdentToken, "more", 35, 39),
				newToken(CommaToken, ",", 40, 41),
				newToken(IdentToken, "tokens", 42, 48),
				newToken(RparenToken, ")", 50, 51),
			},
		},
		{
			`label(impact:some-doc)`,
			[]Token{
				newToken(IdentToken, "label", 0, 5),
				newToken(LparenToken, "(", 5, 6),
				newToken(IdentToken, "impact:some-doc", 6, 21),
				newToken(RparenToken, ")", 21, 22),
			},
		},
		{
			`label("impact:some-doc")`,
			[]Token{
				newToken(IdentToken, "label", 0, 5),
				newToken(LparenToken, "(", 5, 6),
				newToken(StringToken, "impact:some-doc", 6, 23),
				newToken(RparenToken, ")", 23, 24),
			},
		},
		{
			`label(r"impact:some-doc")`,
			[]Token{
				newToken(IdentToken, "label", 0, 5),
				newToken(LparenToken, "(", 5, 6),
				newToken(RegexToken, "impact:some-doc", 6, 24),
				newToken(RparenToken, ")", 24, 25),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			lexer := NewLexer(tc.input)

			idx := 0
			for {
				tok, err := lexer.NextToken()
				assert.NoError(t, err)

				if tok.TokenType == EofToken {
					assert.Equal(t, idx, len(tc.tokens))
					break
				}

				assert.Less(t, idx, len(tc.tokens))
				assert.Equal(t, tok, tc.tokens[idx])

				idx += 1
			}
		})
	}
}

func TestLexerFailures(t *testing.T) {
	var tests = []struct {
		input     string
		err       error
		errString string
	}{
		{`    "unterminated string`, &UnterminatedTokenError{`    "unterminated string`, Span{4, 24}}, `Unterminated string token: "unterminated string`},
	}

	for _, tc := range tests {
		lexer := NewLexer(tc.input)

		_, err := lexer.NextToken()
		assert.Error(t, err)
		assert.EqualValues(t, err, tc.err)
		assert.EqualValues(t, err.Error(), tc.errString)
	}
}

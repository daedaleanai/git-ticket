package query

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
)

type keywordParser func(p *Parser) (AstNode, *ParseError)

var keywordParsers map[string]keywordParser

func init() {
	keywordParsers = map[string]keywordParser{
		"status":      parseStatusExpression,
		"author":      parseAuthorExpression,
		"assignee":    parseAssigneeExpression,
		"ccb":         parseCcbExpression,
		"ccb-pending": parseCcbPendingExpression,
		"actor":       parseActorExpression,
		"participant": parseParticipantExpression,
		"label":       parseLabelExpression,
		"title":       parseTitleExpression,
		"not":         parseNotExpression,
		"create-before": func(parser *Parser) (AstNode, *ParseError) {
			return parseCreationDateFilter(parser, true)
		},
		"create-after": func(parser *Parser) (AstNode, *ParseError) {
			return parseCreationDateFilter(parser, false)
		},
		"edit-before": func(parser *Parser) (AstNode, *ParseError) {
			return parseEditDateFilter(parser, true)
		},
		"edit-after": func(parser *Parser) (AstNode, *ParseError) {
			return parseEditDateFilter(parser, false)
		},
		"all":      parseAndFilter,
		"any":      parseOrFilter,
		"sort":     parseSortOrder,
		"color-by": parseColor,
	}
}

// Parser is a query parser
type Parser struct {
	curToken Token
	lexer    Lexer
	context  parseContext
}

// ParseError represents an error found while parsing a query
type ParseError struct {
	query   string
	span    Span
	message string
}

// parseContext is used to keep the current nested expression parsing for error reporting
type parseContext struct {
	query      string
	parseStack []string
}

func (p *parseContext) push(s string) {
	p.parseStack = append(p.parseStack, s)
}

func (p *parseContext) pop() {
	p.parseStack = p.parseStack[:len(p.parseStack)-1]
}

const UNDERLINE = "\x1b[4m"
const UNDERLINE_RESET = "\x1b[24m"
const RED = "\x1b[31m"
const RESET_COLOR = "\x1b[0m"

func (e *ParseError) Error() string {
	var highlightedError bytes.Buffer
	firstPart := e.query[:e.span.Begin]
	secondPart := e.query[e.span.Begin:e.span.End]
	thirdPart := e.query[e.span.End:]
	highlightedError.WriteString(firstPart)
	highlightedError.WriteString(RED)
	highlightedError.WriteString(UNDERLINE)
	highlightedError.WriteString(secondPart)
	highlightedError.WriteString(UNDERLINE_RESET)
	highlightedError.WriteString(RESET_COLOR)
	highlightedError.WriteString(thirdPart)
	return fmt.Sprintf("%s\n%s\n", e.message, highlightedError.String())
}

func newParseError(context *parseContext, span Span, message string) *ParseError {
	parseStack := []string{}
	for _, e := range context.parseStack {
		parseStack = append([]string{e}, parseStack...)
	}
	ctxString := strings.Join(parseStack, "\n\t")
	return &ParseError{context.query, span, fmt.Sprintf("%s\n\t%s", message, ctxString)}
}

func NewParser(query string) (*Parser, error) {
	l := NewLexer(query)
	tok, err := l.NextToken()
	if err != nil {
		return nil, err
	}

	return &Parser{
		lexer:    l,
		curToken: tok,
		context: parseContext{
			query: query,
		},
	}, nil
}

func (p *Parser) advance() *ParseError {
	tok, err := p.lexer.NextToken()
	if err != nil {
		if lexErr, ok := err.(*UnterminatedTokenError); ok {
			return newParseError(&p.context, lexErr.Span, "Unterminated token")
		}
		return newParseError(&p.context, p.curToken.Span, err.Error())
	}

	p.curToken = tok
	return nil
}

func (p *Parser) expectTokenTypeAndAdvance(t TokenType) *ParseError {
	if p.curToken.TokenType != t {
		return newParseError(&p.context, p.curToken.Span, fmt.Sprintf("Expected token of type %q", t))
	}

	return p.advance()
}

func (p *Parser) Parse() (*CompiledQuery, error) {
	query := CompiledQuery{}

	for {
		switch p.curToken.TokenType {
		case EofToken:
			return &query, nil

		case IdentToken, StringToken:
			err := p.parseQueryStatement(&query)
			if err != nil {
				return &query, err
			}

		default:
			return &query, newParseError(&p.context, p.curToken.Span, fmt.Sprintf("Invalid query. Unexpected node of type %s", p.curToken.TokenType))
		}
	}
}

func (p *Parser) parseQueryStatement(query *CompiledQuery) error {
	keyword := p.curToken.Literal
	specificParser, ok := keywordParsers[keyword]
	if !ok {
		return newParseError(&p.context, p.curToken.Span, "Invalid query statement keyword")
	}

	node, err := specificParser(p)
	if err != nil {
		return err
	}

	if filterNode, ok := node.(FilterNode); ok {
		if query.FilterNode != nil {
			return newParseError(&p.context, filterNode.Span(), "Multiple filtering criteria was specified")
		}
		query.FilterNode = filterNode
		return nil
	}

	if orderByNode, ok := node.(*OrderByNode); ok {
		if query.OrderNode != nil {
			return newParseError(&p.context, orderByNode.Span(), "Multiple ordering criteria was specified")
		}
		query.OrderNode = orderByNode
		return nil
	}

	if colorByNode, ok := node.(*ColorByNode); ok {
		if query.ColorNode != nil {
			return newParseError(&p.context, colorByNode.Span(), "Multiple coloring criteria was specified")
		}
		query.ColorNode = colorByNode
		return nil
	}

	if _, ok := node.(*LiteralNode); ok {
		return newParseError(&p.context, p.curToken.Span, "Literal node found at top level of query")
	}

	return newParseError(&p.context, p.curToken.Span, fmt.Sprintf("Unhandled statement node type: %s", reflect.TypeOf(node)))
}

func (p *Parser) parseExpression() (AstNode, *ParseError) {
	if p.curToken.TokenType == RegexToken {
		tok := p.curToken

		regex, err := regexp.Compile(tok.Literal)
		if err != nil {
			return nil, newParseError(&p.context, p.curToken.Span, "Invalid regular expression")
		}

		advanceErr := p.advance()
		return &RegexNode{Token: tok, Regex: *regex}, advanceErr
	}

	if p.curToken.TokenType != IdentToken && p.curToken.TokenType != StringToken {
		return nil, newParseError(&p.context, p.curToken.Span, fmt.Sprintf("Expression cannot begin with token: %s", p.curToken.TokenType))
	}

	litTok := p.curToken
	specificParser, ok := keywordParsers[litTok.Literal]
	if !ok {
		err := p.advance()
		return &LiteralNode{Token: litTok}, err
	}

	return specificParser(p)
}

func (parser *Parser) parseDelimitedExpressionList() ([]AstNode, Span, *ParseError) {
	nodes := []AstNode{}
	span := parser.curToken.Span

	err := parser.expectTokenTypeAndAdvance(LparenToken)
	if err != nil {
		return nil, span, err
	}

	if parser.curToken.TokenType == RparenToken {
		span = span.Extend(parser.curToken.Span)
		err := parser.advance()
		return nodes, span, err
	}

	for {
		expr, err := parser.parseExpression()
		if err != nil {
			return nodes, span, err
		}
		nodes = append(nodes, expr)

		switch parser.curToken.TokenType {
		case RparenToken:
			span = span.Extend(parser.curToken.Span)
			err = parser.advance()
			return nodes, span, err
		case CommaToken:
			err = parser.advance()
			if err != nil {
				return nodes, span, err
			}
		default:
			return nil, span, newParseError(&parser.context, parser.curToken.Span, fmt.Sprintf("Unexpected delimiter in delimited expression: %s", parser.curToken.TokenType))
		}
	}
}

func (parser *Parser) parseDelimitedLiteralList() ([]*LiteralNode, Span, *ParseError) {
	nodes := []*LiteralNode{}
	span := parser.curToken.Span

	err := parser.expectTokenTypeAndAdvance(LparenToken)
	if err != nil {
		return nil, span, err
	}

	if parser.curToken.TokenType == RparenToken {
		span = span.Extend(parser.curToken.Span)
		err := parser.advance()
		return nodes, span, err
	}

	for {
		if parser.curToken.TokenType != IdentToken && parser.curToken.TokenType != StringToken {
			return nodes, span, newParseError(&parser.context, parser.curToken.Span, "Expected Literal")
		}
		nodes = append(nodes, &LiteralNode{parser.curToken})

		err = parser.advance()
		if err != nil {
			return nodes, span, err
		}

		switch parser.curToken.TokenType {
		case RparenToken:
			span = span.Extend(parser.curToken.Span)
			err = parser.advance()
			return nodes, span, err
		case CommaToken:
			err = parser.advance()
			if err != nil {
				return nodes, span, err
			}
		default:
			return nil, span, newParseError(&parser.context, parser.curToken.Span, fmt.Sprintf("Unexpected delimiter in delimited expression: %s", parser.curToken.TokenType))
		}
	}
}

func (parser *Parser) parseDelimitedLiteralMatcher() (LiteralMatcherNode, Span, *ParseError) {
	nodes, span, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, span, err
	}

	if len(nodes) != 1 {
		return nil, span, newParseError(&parser.context, span, "Expected exactly on expression within the delimiters")
	}

	literal, ok := nodes[0].(LiteralMatcherNode)
	if !ok {
		return nil, span, newParseError(&parser.context, nodes[0].Span(), "Expected Literal matcher (either string or regexp) expression")
	}

	return literal, span, err
}

func parseStatusExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Status expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	list, span, err := parser.parseDelimitedLiteralList()
	if err != nil {
		return nil, err
	}

	node := &StatusFilter{
		span: firstToken.Span.Extend(span),
	}

	appendStatus := func(token Token) *ParseError {
		if strings.EqualFold(token.Literal, "ALL") {
			node.Statuses = append(node.Statuses, bug.AllStatuses()...)
		} else {
			status, err := bug.StatusFromString(token.Literal)
			if err != nil {
				return newParseError(&parser.context, token.Span, "Invalid ticket status")
			}
			node.Statuses = append(node.Statuses, status)
		}
		return nil
	}

	for _, literalNode := range list {
		err := appendStatus(literalNode.Token)
		if err != nil {
			return node, err
		}
	}

	return node, err
}

func parseAuthorExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Author expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &AuthorFilter{Author: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseAssigneeExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Assignee expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &AssigneeFilter{Assignee: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseCcbExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing CCB expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &CcbFilter{Ccb: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseCcbPendingExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing CCB Pending expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &CcbPendingFilter{Ccb: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseActorExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Actor expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &ActorFilter{Actor: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseParticipantExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Participant expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &ParticipantFilter{Participant: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseLabelExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Label expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &LabelFilter{Label: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseTitleExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Title expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, span, err := parser.parseDelimitedLiteralMatcher()
	return &TitleFilter{Title: matcher, span: firstToken.Span.Extend(span)}, err
}

func parseNotExpression(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Not expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	list, innerSpan, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	span := firstToken.Span.Extend(innerSpan)

	if len(list) != 1 {
		return nil, newParseError(&parser.context, innerSpan, "Expected a single expression")
	}

	filter, ok := list[0].(FilterNode)
	if !ok {
		return nil, newParseError(&parser.context, list[0].Span(), "Expected filter expression")
	}

	return &NotFilter{Inner: filter, span: span}, nil
}

func parseCreationDateFilter(parser *Parser, before bool) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Creation Date expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, innerSpan, err := parser.parseDelimitedLiteralMatcher()
	if err != nil {
		return nil, err
	}

	literalNode, ok := matcher.(*LiteralNode)
	if !ok {
		return nil, newParseError(&parser.context, matcher.Span(), "Expected Literal expression")
	}

	span := firstToken.Span.Extend(innerSpan)

	date, err := parseTimeToken(&parser.context, literalNode.Token)
	if err != nil {
		return nil, err
	}

	return &CreationDateFilter{Date: date, Before: before, span: span}, nil
}

func parseEditDateFilter(parser *Parser, before bool) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Edit Date expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, innerSpan, err := parser.parseDelimitedLiteralMatcher()
	if err != nil {
		return nil, err
	}

	literalNode, ok := matcher.(*LiteralNode)
	if !ok {
		return nil, newParseError(&parser.context, matcher.Span(), "Expected Literal expression")
	}

	span := firstToken.Span.Extend(innerSpan)

	date, err := parseTimeToken(&parser.context, literalNode.Token)
	if err != nil {
		return nil, err
	}

	return &EditDateFilter{Date: date, Before: before, span: span}, nil
}

func parseAndFilter(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing All expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	list, innerSpan, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	span := firstToken.Span.Extend(innerSpan)

	filters := []FilterNode{}
	for _, n := range list {
		f, ok := n.(FilterNode)
		if !ok {
			return nil, newParseError(&parser.context, n.Span(), "Expected filter expression")
		}
		filters = append(filters, f)
	}

	return &AndFilter{Inner: filters, span: span}, nil
}

func parseOrFilter(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Any expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	list, innerSpan, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	span := firstToken.Span.Extend(innerSpan)

	filters := []FilterNode{}
	for _, n := range list {
		f, ok := n.(FilterNode)
		if !ok {
			return nil, newParseError(&parser.context, n.Span(), "Expected filter expression")
		}
		filters = append(filters, f)
	}

	return &OrFilter{Inner: filters, span: span}, nil
}

func parseSortOrder(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Sort expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	matcher, innerSpan, err := parser.parseDelimitedLiteralMatcher()
	if err != nil {
		return nil, err
	}

	literalNode, ok := matcher.(*LiteralNode)
	if !ok {
		return nil, newParseError(&parser.context, matcher.Span(), "Expected Literal expression")
	}

	span := firstToken.Span.Extend(innerSpan)

	var orderBy OrderBy
	var orderDirection OrderDirection

	switch literalNode.Token.Literal {
	// default ASC
	case "id-desc":
		orderBy = OrderById
		orderDirection = OrderDescending
	case "id", "id-asc":
		orderBy = OrderById
		orderDirection = OrderAscending

	// default DESC
	case "creation", "creation-desc":
		orderBy = OrderByCreation
		orderDirection = OrderDescending
	case "creation-asc":
		orderBy = OrderByCreation
		orderDirection = OrderAscending

	// default DESC
	case "edit", "edit-desc":
		orderBy = OrderByEdit
		orderDirection = OrderDescending
	case "edit-asc":
		orderBy = OrderByEdit
		orderDirection = OrderAscending

	default:
		return nil, newParseError(&parser.context, literalNode.Span(), "Unknown sorting")
	}

	return &OrderByNode{OrderBy: orderBy, OrderDirection: orderDirection, span: span}, nil
}

func parseColor(parser *Parser) (AstNode, *ParseError) {
	ctx := &parser.context
	ctx.push("While parsing Color expression")
	defer ctx.pop()

	firstToken := parser.curToken
	err := parser.advance()
	if err != nil {
		return nil, err
	}

	nodes, innerSpan, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	if len(nodes) != 1 {
		return nil, newParseError(&parser.context, innerSpan, "Expected exactly on expression within the delimiters")
	}

	colorFilter, ok := nodes[0].(ColorFilterNode)
	if !ok {
		return nil, newParseError(&parser.context, nodes[0].Span(), "Expected Color filter expression")
	}

	span := firstToken.Span.Extend(innerSpan)
	return &ColorByNode{ColorFilter: colorFilter, span: span}, nil
}

func parseTimeToken(context *parseContext, input Token) (time.Time, *ParseError) {
	var formats = []string{"2006-01-02T15:04:05", "2006-01-02"}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, input.Literal, time.Local)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, newParseError(context, input.Span, "Invalid time")
}

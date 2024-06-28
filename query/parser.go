package query

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
)

type keywordParser func(p *Parser) (AstNode, error)

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
		"create-before": func(parser *Parser) (AstNode, error) {
			return parseCreationDateFilter(parser, true)
		},
		"create-after": func(parser *Parser) (AstNode, error) {
			return parseCreationDateFilter(parser, false)
		},
		"edit-before": func(parser *Parser) (AstNode, error) {
			return parseEditDateFilter(parser, true)
		},
		"edit-after": func(parser *Parser) (AstNode, error) {
			return parseEditDateFilter(parser, false)
		},
		"all":      parseAndFilter,
		"any":      parseOrFilter,
		"sort":     parseSortOrder,
		"color-by": parseColor,
	}
}

type Parser struct {
	curToken Token
	lexer    Lexer
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
	}, nil
}

func (p *Parser) advance() error {
	tok, err := p.lexer.NextToken()
	p.curToken = tok
	return err
}

func (p *Parser) expectTokenTypeAndAdvance(t TokenType) error {
	if p.curToken.TokenType != t {
		return fmt.Errorf("Expected %q", t)
	}

	return p.advance()
}

func (p *Parser) Parse() (*CompiledQuery, error) {
	query := CompiledQuery{}

	for {
		switch p.curToken.TokenType {
		case EofToken:
			return &query, nil

		case IdentToken:
			err := p.parseQueryStatement(&query)
			if err != nil {
				return &query, err
			}

		default:
			return &query, fmt.Errorf("Invalid query: unexpected node of type %s", p.curToken.TokenType)
		}
	}
}

func (p *Parser) parseQueryStatement(query *CompiledQuery) error {
	keyword := p.curToken.Literal
	specificParser, ok := keywordParsers[keyword]
	if !ok {
		return fmt.Errorf("Invalid keyword: %q", keyword)
	}

	// Consume first node, since it was already matched
	err := p.advance()
	if err != nil {
		return err
	}

	node, err := specificParser(p)
	if err != nil {
		return err
	}

	if filterNode, ok := node.(FilterNode); ok {
		if query.FilterNode != nil {
			// TODO: Point to both nodes and report error nicely
			return fmt.Errorf("Multiple filtering criteria specified!")
		}
		query.FilterNode = filterNode
		return nil
	}

	if orderByNode, ok := node.(*OrderByNode); ok {
		if query.OrderNode != nil {
			// TODO: Point to both nodes and report error nicely
			return fmt.Errorf("Multiple ordering criteria specified!")
		}
		query.OrderNode = orderByNode
		return nil
	}

	if colorByNode, ok := node.(*ColorByNode); ok {
		if query.ColorNode != nil {
			// TODO: Point to both nodes and report error nicely
			return fmt.Errorf("Multiple coloring criteria specified!")
		}
		query.ColorNode = colorByNode
		return nil
	}

	if n, ok := node.(*LiteralNode); ok {
		return fmt.Errorf("Invalid query: %s", n.token.Literal)
	}

	return fmt.Errorf("Unhandled statement node type: %s", reflect.TypeOf(node))
}

func (p *Parser) parseExpression() (AstNode, error) {
	if p.curToken.TokenType != IdentToken && p.curToken.TokenType != StringToken {
		return nil, fmt.Errorf("Expression cannot begin with token: %s", p.curToken.TokenType)
	}

	litTok := p.curToken

	err := p.advance()
	if err != nil {
		return nil, err
	}

	specificParser, ok := keywordParsers[litTok.Literal]
	if !ok {
		return &LiteralNode{token: litTok}, nil
	}

	node, err := specificParser(p)
	return node, err
}

func (parser *Parser) parseDelimitedExpressionList() ([]AstNode, error) {
	nodes := []AstNode{}

	err := parser.expectTokenTypeAndAdvance(LparenToken)
	if err != nil {
		return nil, err
	}

	if parser.curToken.TokenType == RparenToken {
		err := parser.advance()
		return nodes, err
	}

	for {
		expr, err := parser.parseExpression()
		if err != nil {
			return nodes, err
		}
		nodes = append(nodes, expr)

		switch parser.curToken.TokenType {
		case RparenToken:
			err = parser.advance()
			return nodes, err
		case CommaToken:
			err = parser.advance()
			if err != nil {
				return nodes, err
			}
		default:
			return nodes, fmt.Errorf("Unexpected delimiter in delimited expression: %s", parser.curToken.TokenType)
		}
	}
}

func (parser *Parser) parseDelimitedLiteral() (string, error) {
	err := parser.expectTokenTypeAndAdvance(LparenToken)
	if err != nil {
		return "", err
	}

	if ty := parser.curToken.TokenType; ty != StringToken && ty != IdentToken {
		return "", fmt.Errorf("status expects a token type")
	}

	result := parser.curToken.Literal

	err = parser.advance()
	if err != nil {
		return "", err
	}

	err = parser.expectTokenTypeAndAdvance(RparenToken)
	return result, err
}

func parseStatusExpression(parser *Parser) (AstNode, error) {
	list, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	node := &StatusFilter{}

	appendStatus := func(lit string) error {
		if strings.EqualFold(lit, "ALL") {
			node.Statuses = append(node.Statuses, bug.AllStatuses()...)
		} else {
			status, err := bug.StatusFromString(lit)
			if err != nil {
				return fmt.Errorf("Invalid status name: %s", err)
			}
			node.Statuses = append(node.Statuses, status)
		}
		return nil
	}

	for _, n := range list {
		lit, ok := n.(*LiteralNode)
		if !ok {
			return nil, fmt.Errorf("status() expects a comman separated list of statuses")
		}

		err := appendStatus(lit.token.Literal)
		if err != nil {
			return node, err
		}
	}

	return node, err
}

func parseAuthorExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &AuthorFilter{AuthorName: lit}, err
}

func parseAssigneeExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &AssigneeFilter{AssigneeName: lit}, err
}

func parseCcbExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &CcbFilter{CcbName: lit}, err
}

func parseCcbPendingExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &CcbPendingFilter{CcbName: lit}, err
}

func parseActorExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &ActorFilter{ActorName: lit}, err
}

func parseParticipantExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &ParticipantFilter{ParticipantName: lit}, err
}

func parseLabelExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &LabelFilter{LabelName: lit}, err
}

func parseTitleExpression(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	return &TitleFilter{Title: lit}, err
}

func parseNotExpression(parser *Parser) (AstNode, error) {
	list, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	if len(list) != 1 {
		return nil, fmt.Errorf(`"no" filter only supports a single inner expression`)
	}

	filter, ok := list[0].(FilterNode)
	if !ok {
		return nil, fmt.Errorf("Invalid")
	}

	return &NotFilter{Inner: filter}, nil
}

func parseCreationDateFilter(parser *Parser, before bool) (AstNode, error) {
	literal, err := parser.parseDelimitedLiteral()
	if err != nil {
		return nil, err
	}

	date, err := parseTime(literal)
	if err != nil {
		return nil, err
	}

	return &CreationDateFilter{Date: date, Before: before}, nil
}

func parseEditDateFilter(parser *Parser, before bool) (AstNode, error) {
	literal, err := parser.parseDelimitedLiteral()
	if err != nil {
		return nil, err
	}

	date, err := parseTime(literal)
	if err != nil {
		return nil, err
	}

	return &EditDateFilter{Date: date, Before: before}, nil
}

func parseAndFilter(parser *Parser) (AstNode, error) {
	list, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	filters := []FilterNode{}
	for _, n := range list {
		f, ok := n.(FilterNode)
		if !ok {
			return nil, fmt.Errorf("Can only contain filters")
		}
		filters = append(filters, f)
	}

	return &AndFilter{Inner: filters}, nil
}

func parseOrFilter(parser *Parser) (AstNode, error) {
	list, err := parser.parseDelimitedExpressionList()
	if err != nil {
		return nil, err
	}

	filters := []FilterNode{}
	for _, n := range list {
		f, ok := n.(FilterNode)
		if !ok {
			return nil, fmt.Errorf("Can only contain filters")
		}
		filters = append(filters, f)
	}

	return &OrFilter{Inner: filters}, nil
}

func parseSortOrder(parser *Parser) (AstNode, error) {
	lit, err := parser.parseDelimitedLiteral()
	if err != nil {
		return nil, err
	}

	var orderBy OrderBy
	var orderDirection OrderDirection

	switch lit {
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
		return nil, fmt.Errorf("unknown sorting %s", lit)
	}

	return &OrderByNode{OrderBy: orderBy, OrderDirection: orderDirection}, nil
}

func parseColor(parser *Parser) (AstNode, error) {
	// TODO: Implement
	return &ColorByNode{}, nil
}

// Parse parse a query DSL
//
// Ex: "status:open author:descartes sort:edit-asc"
//
// Supported filter qualifiers and syntax are described in docs/queries.md
func Parse(query string) (*Query, error) {
	tokens, err := tokenize(query)
	if err != nil {
		return nil, err
	}

	q := &Query{
		OrderBy:        OrderByCreation,
		OrderDirection: OrderDescending,
	}
	sortingDone := false
	coloringDone := false

	for _, t := range tokens {
		switch t.qualifier {
		case "status", "state":
			if strings.EqualFold(t.value, "ALL") {
				q.Status = bug.AllStatuses()
				continue
			}
			status, err := bug.StatusFromString(t.value)
			if err != nil {
				return nil, err
			}
			q.Status = append(q.Status, status)
		case "author":
			q.Author = append(q.Author, t.value)
		case "assignee":
			q.Assignee = append(q.Assignee, t.value)
		case "ccb":
			q.Ccb = append(q.Ccb, t.value)
		case "ccb-pending":
			q.CcbPending = append(q.CcbPending, t.value)
		case "actor":
			q.Actor = append(q.Actor, t.value)
		case "participant":
			q.Participant = append(q.Participant, t.value)
		case "label":
			q.Label = append(q.Label, t.value)
		case "title":
			q.Title = append(q.Title, t.value)
		case "no":
			switch t.value {
			case "label":
				q.NoLabel = true
			default:
				return nil, fmt.Errorf("unknown \"no\" filter \"%s\"", t.value)
			}
		case "create-before":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.CreateBefore = parsedTime
		case "create-after":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.CreateAfter = parsedTime
		case "edit-before":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.EditBefore = parsedTime
		case "edit-after":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.EditAfter = parsedTime
		case "sort":
			if sortingDone {
				return nil, fmt.Errorf("multiple sorting")
			}
			err = parseSorting(q, t.value)
			if err != nil {
				return nil, err
			}
			sortingDone = true
		case "color-by":
			if coloringDone {
				return nil, fmt.Errorf("multiple coloring")
			}
			err = parseColoring(q, t.value)
			if err != nil {
				return nil, err
			}
			coloringDone = true

		default:
			return nil, fmt.Errorf("unknown qualifier \"%s\"", t.qualifier)
		}
	}
	return q, nil
}

func parseSorting(q *Query, value string) error {
	switch value {
	// default ASC
	case "id-desc":
		q.OrderBy = OrderById
		q.OrderDirection = OrderDescending
	case "id", "id-asc":
		q.OrderBy = OrderById
		q.OrderDirection = OrderAscending

	// default DESC
	case "creation", "creation-desc":
		q.OrderBy = OrderByCreation
		q.OrderDirection = OrderDescending
	case "creation-asc":
		q.OrderBy = OrderByCreation
		q.OrderDirection = OrderAscending

	// default DESC
	case "edit", "edit-desc":
		q.OrderBy = OrderByEdit
		q.OrderDirection = OrderDescending
	case "edit-asc":
		q.OrderBy = OrderByEdit
		q.OrderDirection = OrderAscending

	default:
		return fmt.Errorf("unknown sorting %s", value)
	}

	return nil
}

func parseTime(input string) (time.Time, error) {
	var formats = []string{"2006-01-02T15:04:05", "2006-01-02"}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, input, time.Local)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("unrecognized time format")
}

func parseColoring(q *Query, value string) error {
	if strings.HasPrefix(value, "ccb-pending-user:") {
		q.ColorBy = ColorByCcbPendingByUser
		q.ColorByCcbUserName = ColorByCcbUserName(strings.TrimPrefix(value, "ccb-pending-user:"))
		return nil
	}

	if strings.HasPrefix(value, "label:") {
		q.ColorBy = ColorByLabel
		q.ColorByLabelPrefix = ColorByLabelPrefix(strings.TrimPrefix(value, "label:"))
		return nil
	}

	if value == "author" {
		q.ColorBy = ColorByAuthor
		return nil
	}

	if value == "assignee" {
		q.ColorBy = ColorByAssignee
		return nil
	}

	return fmt.Errorf("unknown coloring %s", value)
}

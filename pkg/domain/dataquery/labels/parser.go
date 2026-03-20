package labels

import (
	"fmt"
	"strings"
	"unicode"
)

// Parser parses label filter expressions
type Parser struct {
	input   string
	pos     int
	tokens  []token
	current int
}

type token struct {
	typ   tokenType
	value string
}

type tokenType int

const (
	tokEOF tokenType = iota
	tokIdent
	tokString
	tokEq
	tokNeq
	tokReMatch
	tokReNotMatch
	tokLParen
	tokRParen
	tokAnd
	tokOr
)

// Parse parses a label filter expression and returns an AST
func Parse(input string) (Expr, error) {
	p := &Parser{input: input}
	if err := p.tokenize(); err != nil {
		return nil, err
	}
	return p.parse()
}

func (p *Parser) tokenize() error {
	input := strings.TrimSpace(p.input)
	for i := 0; i < len(input); i++ {
		// Skip whitespace
		for i < len(input) && unicode.IsSpace(rune(input[i])) {
			i++
		}
		if i >= len(input) {
			break
		}

		ch := input[i]

		// Check for operators
		if i+1 < len(input) {
			twoChar := input[i : i+2]
			switch twoChar {
			case "!=":
				p.tokens = append(p.tokens, token{tokNeq, "!="})
				i++
				continue
			case "=~":
				p.tokens = append(p.tokens, token{tokReMatch, "=~"})
				i++
				continue
			case "!~":
				p.tokens = append(p.tokens, token{tokReNotMatch, "!~"})
				i++
				continue
			}
		}

		switch ch {
		case '=':
			p.tokens = append(p.tokens, token{tokEq, "="})
		case '(':
			p.tokens = append(p.tokens, token{tokLParen, "("})
		case ')':
			p.tokens = append(p.tokens, token{tokRParen, ")"})
		case '"', '\'':
			// Parse string literal
			quote := ch
			start := i + 1
			i++
			for i < len(input) && input[i] != byte(quote) {
				if input[i] == '\\' && i+1 < len(input) {
					i++ // skip escaped char
				}
				i++
			}
			if i >= len(input) {
				return fmt.Errorf("unterminated string literal")
			}
			p.tokens = append(p.tokens, token{tokString, input[start:i]})
		default:
			// Parse identifier or keyword
			if isIdentStart(ch) {
				start := i
				for i < len(input) && isIdentChar(input[i]) {
					i++
				}
				value := strings.ToUpper(input[start:i])
				switch value {
				case "AND":
					p.tokens = append(p.tokens, token{tokAnd, "AND"})
				case "OR":
					p.tokens = append(p.tokens, token{tokOr, "OR"})
				default:
					p.tokens = append(p.tokens, token{tokIdent, input[start:i]})
				}
				i-- // adjust for loop increment
			} else {
				return fmt.Errorf("unexpected character: %c", ch)
			}
		}
	}
	p.tokens = append(p.tokens, token{tokEOF, ""})
	return nil
}

func isIdentStart(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isIdentChar(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_'
}

func (p *Parser) peek() token {
	if p.current >= len(p.tokens) {
		return token{tokEOF, ""}
	}
	return p.tokens[p.current]
}

func (p *Parser) advance() token {
	tok := p.peek()
	p.current++
	return tok
}

func (p *Parser) parse() (Expr, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.peek().typ == tokOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: OpOr, Left: left, Right: right}
	}

	return left, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.peek().typ == tokAnd {
		p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: OpAnd, Left: left, Right: right}
	}

	return left, nil
}

func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.peek()

	switch tok.typ {
	case tokLParen:
		p.advance()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().typ != tokRParen {
			return nil, fmt.Errorf("expected closing parenthesis")
		}
		p.advance()
		return expr, nil

	case tokIdent:
		return p.parseComparison()

	default:
		return nil, fmt.Errorf("unexpected token: %v", tok.typ)
	}
}

func (p *Parser) parseComparison() (Expr, error) {
	keyTok := p.advance()
	if keyTok.typ != tokIdent {
		return nil, fmt.Errorf("expected identifier, got %v", keyTok.typ)
	}

	opTok := p.advance()
	var op ComparisonOp
	switch opTok.typ {
	case tokEq:
		op = OpEq
	case tokNeq:
		op = OpNeq
	case tokReMatch:
		op = OpReMatch
	case tokReNotMatch:
		op = OpReNotMatch
	default:
		return nil, fmt.Errorf("expected comparison operator, got %v", opTok.typ)
	}

	valueTok := p.advance()
	if valueTok.typ != tokString && valueTok.typ != tokIdent {
		return nil, fmt.Errorf("expected string value, got %v", valueTok.typ)
	}

	return &Comparison{
		Key:   keyTok.value,
		Op:    op,
		Value: valueTok.value,
	}, nil
}

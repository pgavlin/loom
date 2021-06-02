package loom

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ParseString(s string) (SExpression, error) {
	return Parse(strings.NewReader(s))
}

func Parse(r io.Reader) (SExpression, error) {
	p := &parser{l: &lexer{r: bufio.NewReader(r)}}
	return p.parseExpression(0, false)
}

type parser struct {
	l *lexer
	t interface{}
}

func (p *parser) peek() interface{} {
	if p.t == nil {
		p.t, _ = p.l.next()
	}
	return p.t
}

func (p *parser) next() (interface{}, error) {
	if p.t != nil {
		v := p.t
		p.t = nil
		return v, nil
	}
	return p.l.next()
}

func (p *parser) parseExpression(qq int, splice bool) (SExpression, error) {
	tok, err := p.next()
	if err != nil {
		return nil, err
	}

	switch tok := tok.(type) {
	case SExpression:
		return tok, nil
	case rune:
		switch tok {
		case '(':
			if p.peek() == ')' {
				p.next()
				return nil, nil
			}

			first, err := p.parseExpression(qq, true)
			if err != nil {
				return nil, err
			}
			if first == Symbol("quasiquote") {
				qq++
			} else if qq > 0 {
				if first == Symbol("unquote") {
					qq--
				} else if splice && first == Symbol("unquote-splicing") {
					splice = false
				}
			}

			head := &Pair{car: first}
			tail := head
			for {
				switch p.peek() {
				case ')':
					p.next()
					return head, nil
				case Symbol("."):
					p.next()
					last, err := p.parseExpression(qq, true)
					if err != nil {
						return nil, err
					}
					if tok, _ := p.next(); tok != ')' {
						return nil, fmt.Errorf("unexpected token %v", tok)
					}
					tail.cdr = last
					return head, nil
				}

				next, err := p.parseExpression(qq, true)
				if err != nil {
					return nil, err
				}
				p := &Pair{car: next}
				tail.cdr, tail = p, p
			}
		case '[':
			var vec Vector
			for {
				if p.peek() == ')' {
					return vec, nil
				}

				el, err := p.parseExpression(qq, true)
				if err != nil {
					return nil, err
				}
				vec = append(vec, el)
			}
		case '\'':
			el, err := p.parseExpression(qq, false)
			if err != nil {
				return nil, err
			}
			return Vector{Symbol("quote"), el}.ToList(), nil
		case '`':
			el, err := p.parseExpression(qq+1, false)
			if err != nil {
				return nil, err
			}
			return Vector{Symbol("quasiquote"), el}.ToList(), nil
		case ',':
			if qq == 0 {
				return nil, fmt.Errorf("unquote must be nested inside quasiquation")
			}

			el, err := p.parseExpression(qq-1, false)
			if err != nil {
				return nil, err
			}
			return Vector{Symbol("unquote"), el}.ToList(), nil
		case '@':
			if qq == 0 || !splice {
				return nil, fmt.Errorf("unquote-splicing must be nested inside quasiquation")
			}

			el, err := p.parseExpression(qq-1, false)
			if err != nil {
				return nil, err
			}
			return Vector{Symbol("unquote-splicing"), el}.ToList(), nil
		case '#':
			if _, err := p.parseExpression(qq, splice); err != nil {
				return nil, err
			}
			if p.peek() == 0 {
				return nil, nil
			}
			return p.parseExpression(qq, splice)
		default:
			return nil, fmt.Errorf("unexpected token %v", tok)
		}
	default:
		return nil, fmt.Errorf("unexpected token %v", tok)
	}
}

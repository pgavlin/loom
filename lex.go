package loom

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/big"
	"strings"
	"unicode"
)

type lexer struct {
	r *bufio.Reader
}

var identifierInitial = []*unicode.RangeTable{
	unicode.Lu,
	unicode.Ll,
	unicode.Lt,
	unicode.Lm,
	unicode.Lo,
	unicode.Mn,
	unicode.Nl,
	unicode.No,
	unicode.Pd,
	unicode.Pc,
	unicode.Po,
	unicode.Sc,
	unicode.Sm,
	unicode.Sk,
	unicode.So,
	unicode.Co,
}

var indentifierSubsequent = append(identifierInitial,
	unicode.Nd,
	unicode.Mc,
	unicode.Me,
)

func (l *lexer) read() (rune, error) {
	c, _, err := l.r.ReadRune()
	if err != nil {
		if err == io.EOF {
			return 0, nil
		}
	}
	return c, nil
}

func (l *lexer) peek() rune {
	c, _ := l.read()
	l.r.UnreadRune()
	return c
}

func (l *lexer) next() (interface{}, error) {
	for {
		c, err := l.read()
		if err != nil {
			return nil, err
		}

		switch c {
		case 0:
			return nil, io.EOF
		case '(', ')', '\'', '`':
			return c, nil
		case ',':
			if l.peek() == '@' {
				return '@', nil
			}
			return ',', nil
		case '"':
			return l.string()
		case ';':
			if err := l.lineComment(); err != nil {
				return nil, err
			}
		case '#':
			k, _, err := l.r.ReadRune()
			if err != nil {
				return nil, err
			}
			switch k {
			case 'b':
				return l.numPrefix(2)
			case 'o':
				return l.numPrefix(8)
			case 'd':
				return l.numPrefix(10)
			case 'x':
				return l.numPrefix(16)
			case 'i':
				return l.numPrefix(0)
			case 'e':
				return l.numPrefix(0)
			case 't':
				// TODO: #true
				return Boolean(true), nil
			case 'f':
				// TODO: #false
				return Boolean(false), nil
			case '\\':
				return l.char()
			case '(':
				return '[', nil
			case ';':
				return '#', nil
			case '!':
				if err := l.directive(); err != nil {
					return nil, err
				}
			case '|':
				if err := l.blockComment(); err != nil {
					return nil, err
				}
			default:
				// TODO: #u8(
				return string([]rune{c, k}), nil
			}
		case '-', '+', '.':
			return l.num(c, 10, true)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return l.num(c, 10, false)
		case '!', '$', '%', '&', '*', '/', ':', '<', '=', '>', '?', '^', '_', '~':
			return l.identifier(c)
		case '|':
			return l.symbol()
		default:
			if beginsIdentifier(c) {
				return l.identifier(c)
			}
			if !isSpace(c) {
				return c, nil
			}
		}
	}
}

func (l *lexer) lineComment() (err error) {
	for c := rune(0); c != '\n'; {
		if c, err = l.read(); err != nil {
			return err
		}
	}
	return nil
}

func (l *lexer) blockComment() error {
	nest := 1
	for nest > 0 {
		c, err := l.read()
		if err != nil {
			return err
		}
		switch c {
		case '#':
			k, _, err := l.r.ReadRune()
			if err != nil {
				return err
			}
			if k == '|' {
				nest++
			}
		case '|':
			k, _, err := l.r.ReadRune()
			if err != nil {
				return err
			}
			if k == '#' {
				nest--
			}
		}
	}
	return nil
}

func (l *lexer) directive() error {
	return nil
}

func (l *lexer) numPrefix(radix int) (interface{}, error) {
	c, err := l.read()
	if err != nil {
		return nil, err
	}

	// prefix
	if c == '#' {
		if c, err = l.read(); err != nil {
			return nil, err
		}
		if radix == 0 {
			switch c {
			case 'b':
				radix = 2
			case 'o':
				radix = 8
			case 'd':
				radix = 10
			case 'x':
				radix = 16
			default:
				return nil, fmt.Errorf("invalid radix #%v", c)
			}
		} else {
			switch c {
			case 'i', 'e':
				// OK (TODO: do not ignore exactness)
			default:
				return nil, fmt.Errorf("invalid exactness #%v", c)
			}
		}
		if c, err = l.read(); err != nil {
			return nil, err
		}
	}

	return l.num(c, radix, false)
}

func (l *lexer) num(c rune, radix int, maybeIdentifier bool) (v interface{}, err error) {
	var text strings.Builder
	for {
		text.WriteRune(c)

		c, err = l.read()
		if err != nil {
			return nil, err
		}
		if !continuesIdentifier(c) {
			l.r.UnreadRune()
			break
		}
	}

	// TODO: number lexer

	var f big.Float

	s := text.String()
	switch s {
	case "+inf.0":
		f.SetInf(false)
		return Number{&f}, nil
	case "-inf.0":
		f.SetInf(true)
		return Number{&f}, nil
	case "+nan.0", "-nan.0":
		f.SetFloat64(math.NaN())
		return Number{&f}, nil
	}

	// Rationals
	if i := strings.IndexByte(s, '/'); i != -1 {
		var num, denom big.Int
		_, numOk := num.SetString(s[:i], radix)
		_, denomOk := denom.SetString(s[i+1:], radix)
		if numOk && denomOk {
			var rat big.Rat
			rat.SetFrac(&num, &denom)
			f.SetRat(&rat)
			return Number{&f}, nil
		}
	} else if _, _, err = f.Parse(s, radix); err == nil {
		return Number{&f}, nil
	}

	if !maybeIdentifier {
		return nil, fmt.Errorf("invalid number literal '%s'", s)
	}

	return Symbol(s), nil
}

func (l *lexer) char() (interface{}, error) {
	return nil, fmt.Errorf("NYI: chars")
}

func (l *lexer) string() (interface{}, error) {
	var s strings.Builder
	for {
		c, err := l.read()
		if err != nil {
			return nil, err
		}
		switch c {
		case '"', 0:
			return String(s.String()), nil
		case '\\':
			k, err := l.read()
			if err != nil {
				return nil, err
			}
			switch k {
			case '\\', '"':
				c = k
			case 'a':
				c = '\a'
			case 'b':
				c = '\b'
			case 't':
				c = '\t'
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case ' ', '\t':
				for {
					if k, err = l.read(); err != nil {
						return nil, err
					}
					if k == '\n' {
						break
					}
					if k != ' ' && k != '\t' {
						return nil, fmt.Errorf("unterminated line continuation")
					}
				}
				for k = l.peek(); k == ' ' || k == '\t'; k = l.peek() {
					if _, err = l.read(); err != nil {
						return nil, err
					}
				}
				continue
			case 'x':
				c = 0
				for {
					if k, err = l.read(); err != nil {
						return nil, err
					}
					if k == ';' {
						break
					}
					d, ok := hexDigit(k)
					if !ok {
						return nil, fmt.Errorf("invalid hex digit '%v'", k)
					}
					c = c*16 + d
				}
			default:
				return nil, fmt.Errorf("invalid escape sequence '\\%v'", k)
			}
		}
		s.WriteRune(c)
	}
}

func (l *lexer) identifier(first rune) (interface{}, error) {
	var id strings.Builder
	id.WriteRune(first)

	for {
		c, err := l.read()
		if err != nil {
			return nil, err
		}
		if !continuesIdentifier(c) {
			l.r.UnreadRune()
			return Symbol(id.String()), nil
		}
		id.WriteRune(c)
	}
}

func (l *lexer) symbol() (interface{}, error) {
	return nil, fmt.Errorf("NYI: symbols")
}

func beginsIdentifier(c rune) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || unicode.IsOneOf(identifierInitial, c)
}

func continuesIdentifier(c rune) bool {
	return c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.' || c == '@' || beginsIdentifier(c)
}

func hexDigit(c rune) (rune, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A', true
	case c >= 'a' && c <= 'f':
		return c - 'a', true
	default:
		return 0, false
	}
}

func radixDigit(c rune, radix int) (v int, ok bool) {
	switch {
	case c >= '0' && c <= '9':
		v = int(c - '0')
	case c >= 'A' && c <= 'Z':
		v = int(c - 'A')
	case c >= 'a' && c <= 'z':
		v = int(c - 'a')
	default:
		return 0, false
	}
	return v, v < radix
}

func isSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == 0
}

package loom

import (
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"
)

// Value
type Value interface {
	MarshalSExp() SExpression
}

// SExpressions
type SExpression interface {
	Value

	write(w io.Writer) error
}

// Encode writes a textual representation of v to w.
func Encode(w io.Writer, v Value) error {
	if p, ok := v.(SExpression); ok {
		return p.write(w)
	}
	if v == nil {
		_, err := w.Write([]byte("()"))
		return err
	}
	return v.MarshalSExp().write(w)
}

// Encode returns the textual representation of v.
func EncodeToString(v Value) string {
	var b strings.Builder
	Encode(&b, v)
	return b.String()
}

// Number
type Number struct {
	f *big.Float
}

func (n Number) write(w io.Writer) error {
	text, err := n.f.MarshalText()
	if err != nil {
		return err
	}
	_, err = w.Write(text)
	return err
}

func (n Number) MarshalSExp() SExpression {
	return n
}

func NewInt(x int64) Number {
	var f big.Float
	f.SetInt64(x)
	return Number{&f}
}

func NewUint(x uint64) Number {
	var f big.Float
	f.SetUint64(x)
	return Number{&f}
}

func NewFloat(x float64) Number {
	var f big.Float
	f.SetFloat64(x)
	return Number{&f}
}

func (n Number) Int() (int64, bool) {
	x, acc := n.f.Int64()
	return x, acc == big.Exact
}

func (n Number) Uint() (uint64, bool) {
	x, acc := n.f.Uint64()
	return x, acc == big.Exact
}

func (n Number) Float64() (float64, bool) {
	x, acc := n.f.Float64()
	return x, acc == big.Exact
}

// Boolean
type Boolean bool

func (b Boolean) MarshalSExp() SExpression {
	return b
}

func (b Boolean) write(w io.Writer) error {
	text := "#t"
	if !b {
		text = "#f"
	}
	_, err := w.Write([]byte(text))
	return err
}

// Truthy returns the truth value of v. Any value besides false is considered true.
func Truthy(v Value) bool {
	b, ok := v.(Boolean)
	return !ok || bool(b)
}

// Pair
type Pair struct {
	car Value
	cdr Value
}

func Cons(car, cdr Value) *Pair {
	return &Pair{car: car, cdr: cdr}
}

func (p *Pair) MarshalSExp() SExpression {
	return p
}

func (p *Pair) write(w io.Writer) error {
	if _, err := w.Write([]byte("(")); err != nil {
		return err
	}
	first := true
	for p != nil {
		if !first {
			if _, err := w.Write([]byte(" ")); err != nil {
				return err
			}
		}
		first = false

		if err := Encode(w, p.car); err != nil {
			return err
		}
		if p.cdr == nil {
			break
		}
		tail, ok := p.cdr.(*Pair)
		if !ok {
			if _, err := w.Write([]byte(" . ")); err != nil {
				return err
			}
			if err := Encode(w, p.cdr); err != nil {
				return err
			}
			break
		}
		p = tail
	}
	_, err := w.Write([]byte(")"))
	return err
}

// Car returns the car field of the pair.
func (p *Pair) Car() Value {
	return p.car
}

// Cdr returns the cdr field of the pair.
func (p *Pair) Cdr() Value {
	return p.cdr
}

// ToVector converts the list to a vector.
func (p *Pair) ToVector() Vector {
	var vec Vector
	for p != nil {
		vec = append(vec, p.car)
		if p.cdr == nil {
			return vec
		}
		tail, ok := p.cdr.(*Pair)
		if !ok {
			vec = append(vec, p.cdr)
			return vec
		}
		p = tail
	}
	return vec
}

func (p *Pair) next() (*Pair, bool) {
	next, ok := p.cdr.(*Pair)
	return next, ok
}

func (p *Pair) len() int {
	if p == nil {
		return 0
	}

	l := 1
	for {
		next, ok := p.cdr.(*Pair)
		if !ok {
			if p.cdr == nil {
				return l
			}
			return l + 1
		}
		p, l = next, l+1
	}
}

type splice struct {
	p *Pair
}

func (s splice) MarshalSExp() SExpression {
	return Vector{Symbol("splice"), s.p}.ToList()
}

// Symbol
type Symbol string

func (s Symbol) MarshalSExp() SExpression {
	return s
}

func (s Symbol) write(w io.Writer) error {
	_, err := w.Write([]byte(s))
	return err
}

// Binding
type binding struct {
	where *scope
	name  Symbol
}

func (b *binding) MarshalSExp() SExpression {
	return b.name
}

// Character
type Character rune

func (c Character) MarshalSExp() SExpression {
	return c
}

func (c Character) write(w io.Writer) error {
	var buf [8]byte
	len := utf8.EncodeRune(buf[:], rune(c))
	_, err := w.Write(buf[:len])
	return err
}

// String
type String string

func (s String) MarshalSExp() SExpression {
	return s
}

func (s String) write(w io.Writer) error {
	_, err := w.Write([]byte(s))
	return err
}

// Vector
type Vector []Value

func (v Vector) MarshalSExp() SExpression {
	return v
}

func (v Vector) write(w io.Writer) error {
	if _, err := w.Write([]byte("(vector")); err != nil {
		return err
	}
	for _, e := range v {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if err := Encode(w, e); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte(")"))
	return err
}

// ToList converts the vector to a list.
func (v Vector) ToList() SExpression {
	var head SExpression
	for i := len(v) - 1; i >= 0; i-- {
		head = &Pair{car: v[i], cdr: head}
	}
	return head
}

// Procedure
type Procedure interface {
	Value

	Apply(args Vector) Value
}

type ProcedureFunc func(args Vector) Value

func (f ProcedureFunc) MarshalSExp() SExpression {
	return Symbol("<builtin procedure>")
}

func (f ProcedureFunc) Apply(args Vector) Value {
	return f(args)
}

type tailCall struct {
	p    Procedure
	args Vector
}

func (t *tailCall) MarshalSExp() SExpression {
	return &Pair{car: Symbol("tail"), cdr: &Pair{car: t.p.MarshalSExp(), cdr: t.args.ToList()}}
}

type procedure struct {
	name       Symbol
	closure    *scope
	formals    []Symbol
	isVariadic bool
	body       []Value
}

func makeFormals(declaration Value) (formals []Symbol, isVariadic bool) {
	const invalidFormals = "⟨formals⟩ must be of the form (⟨variable1⟩ ...), ⟨variable⟩, or (⟨variable1 ⟩ . . . ⟨variablen ⟩ . ⟨variablen+1 ⟩)"

	if sym, ok := declaration.(Symbol); ok {
		return []Symbol{sym}, true
	}

	if declaration == nil {
		return nil, false
	}

	pair, ok := declaration.(*Pair)
	if !ok {
		panic(invalidFormals)
	}

	declared := map[Symbol]struct{}{}
	for {
		sym, ok := pair.car.(Symbol)
		if !ok {
			panic(invalidFormals)
		}
		if _, ok := declared[sym]; ok {
			panic(fmt.Errorf("duplicate formal %v", sym))
		}
		declared[sym] = struct{}{}
		formals = append(formals, sym)

		switch cdr := pair.cdr.(type) {
		case Symbol:
			formals, isVariadic = append(formals, cdr), true
			return
		case *Pair:
			pair = cdr
		default:
			if cdr == nil {
				return
			}
			panic(invalidFormals)
		}
	}
}

func (p *procedure) MarshalSExp() SExpression {
	return Symbol(p.name)
}

func (p *procedure) Apply(args Vector) Value {
	return forceTail(p.apply(args))
}

func (p *procedure) apply(args Vector) Value {
	scope := p.closure.push()

	formals, atLeast := p.formals, ""
	if p.isVariadic {
		formals, atLeast = formals[:len(formals)-1], " at least"
	}

	if len(args) < len(formals) {
		panic(fmt.Sprintf("%v expects%v %d arguments", p.name, atLeast, len(formals)))
	}

	for i, sym := range formals {
		scope.set(sym, args[i])
	}

	if p.isVariadic {
		scope.set(p.formals[len(p.formals)-1], args[len(formals):].ToList())
	}

	for _, x := range p.body[:len(p.body)-1] {
		eval(x, scope, false)
	}

	return eval(p.body[len(p.body)-1], scope, true)
}

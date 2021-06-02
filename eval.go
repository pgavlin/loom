package loom

import (
	"fmt"
)

// scope
type scope struct {
	env    map[Symbol]Value
	syntax map[Symbol]*syntaxRules
	outer  *scope
}

func (s *scope) where(name Symbol) *scope {
	for s != nil {
		if _, ok := s.env[name]; ok {
			return s
		}
		s = s.outer
	}
	return nil
}

func (s *scope) bound(name Symbol) bool {
	return s.where(name) != nil
}

func (s *scope) set(name Symbol, v Value) {
	s.env[name] = v
}

func (s *scope) setIfBound(name Symbol, v Value) bool {
	for s != nil {
		if _, ok := s.env[name]; ok {
			s.env[name] = v
			return true
		}
		s = s.outer
	}
	return false
}

func (s *scope) lookup(name Symbol) (Value, bool) {
	for s != nil {
		if v, ok := s.env[name]; ok {
			return v, true
		}
		s = s.outer
	}
	return nil, false
}

func (s *scope) lookupKeyword(name Symbol) (*syntaxRules, bool) {
	for s != nil {
		if v, ok := s.syntax[name]; ok {
			return v, true
		}
		s = s.outer
	}
	return nil, false
}

func (s *scope) setKeyword(name Symbol, v *syntaxRules) {
	s.syntax[name] = v
}

func (s *scope) push() *scope {
	return &scope{env: map[Symbol]Value{}, syntax: map[Symbol]*syntaxRules{}, outer: s}
}

func (s *scope) pop() *scope {
	return s.outer
}

type Env struct {
	globals *scope
}

func NewEnv() *Env {
	return &Env{globals: globalScope}
}

func (e *Env) With(bindings map[Symbol]Value) *Env {
	if bindings == nil {
		bindings = map[Symbol]Value{}
	}

	return &Env{globals: &scope{env: bindings, outer: e.globals}}
}

func (e *Env) Bound(name Symbol) bool {
	return e.globals.bound(name)
}

func (e *Env) Set(name Symbol, v Value) {
	e.globals.set(name, v)
}

func (e *Env) Eval(expression Value) Value {
	return eval(expression, e.globals, false)
}

func (e *Env) EvalTail(expression Value) Value {
	return eval(expression, e.globals, true)
}

// ⟨variable⟩
//
// An expression consisting of a variable (section 3.1) is a variable reference.
// The value of the variable reference is the value stored in the location to
// which the variable is bound. It is an error to reference an unbound variable.
func evalVariable(e Symbol, scope *scope) Value {
	value, ok := scope.lookup(e)
	if !ok {
		panic(fmt.Sprintf("%v is not bound", string(e)))
	}
	return value
}

// (quote ⟨datum⟩)
// ’⟨datum⟩
// ⟨constant⟩
//
// (quote ⟨datum⟩) evaluates to ⟨datum⟩. ⟨Datum⟩ can be any external
// representation of a Scheme object (see section 3.3). This notation is used to
// include literal constants in Scheme code.
func evalQuote(e *Pair) Value {
	return e.cdr.(*Pair).car
}

func evalQuasiquote(v Value, scope *scope) Value {
	if v == nil {
		return nil
	}

	switch v := v.(type) {
	case Number, Boolean, Character, String, Symbol:
		return v
	case Vector:
		result := make(Vector, 0, len(v))
		for _, v := range v {
			elem := evalQuasiquote(v, scope)
			if splice, ok := elem.(splice); ok {
				for p := splice.p; p != nil; p, _ = p.cdr.(*Pair) {
					result = append(result, p.car)
				}
				continue
			}

			result = append(result, elem)
		}
		return result
	case *Pair:
		switch sym, _ := v.car.(Symbol); sym {
		case "unquote":
			return eval(v.cdr.(*Pair).car, scope, false)
		case "unquote-splicing":
			p, ok := eval(v.cdr.(*Pair).car, scope, false).(*Pair)
			if !ok {
				panic("the argument to unquote-splicing must be a list")
			}
			return splice{p}
		case "quasiquote":
			return evalQuasiquote(v.cdr.(*Pair).car, scope)

		// all else
		default:
			var head, tail *Pair
			for {
				switch sym, _ := v.car.(Symbol); sym {
				case "unquote":
					p, ok := v.cdr.(*Pair)
					if ok && p.cdr == nil {
						tail.cdr = eval(p.car, scope, false)
						return head
					}
				case "unquote-splicing":
					p, ok := v.cdr.(*Pair)
					if ok && p.cdr == nil {
						l, ok := eval(p.car, scope, false).(*Pair)
						if !ok {
							panic("the argument to unquote-splicing must be a list")
						}
						tail.cdr = l
						return head
					}
				}

				elem := evalQuasiquote(v.car, scope)
				var p, q *Pair
				if splice, ok := elem.(splice); ok {
					p, q = splice.p, splice.p
					for q.cdr != nil {
						q = q.cdr.(*Pair)
					}
				} else {
					p = &Pair{car: elem}
					q = p
				}

				if head == nil {
					head, tail = p, q
				} else {
					tail.cdr, tail = p, q
				}

				if p, ok := v.cdr.(*Pair); ok {
					v = p
					continue
				}

				tail.cdr = evalQuasiquote(v.cdr, scope)
				return head
			}
		}
	default:
		panic(fmt.Sprintf("unknown quasiquote type %T", v))
	}
}

// (lambda ⟨formals⟩ ⟨body⟩)
//
// Syntax: ⟨Formals⟩ is a formal arguments list as described below, and ⟨body⟩
// is a sequence of zero or more definitions followed by one or more
// expressions.
//
// Semantics: A lambda expression evaluates to a procedure. The environment in
// effect when the lambda expression was evaluated is remembered as part of the
// procedure. When the procedure is later called with some actual arguments,
// the environment in which the lambda expression was evaluated will be extended
// by binding the variables in the formal argument list to fresh locations, and
// the corresponding actual argument values will be stored in those locations.
// (A fresh location is one that is distinct from every previously existing
// location.) Next, the expressions in the body of the lambda expression (which,
// if it contains definitions, represents a letrec* form — see section 4.2.2)
// will be evaluated sequentially in the extended environment. The results of
// the last expression in the body will be returned as the results of the
// procedure call.
//
// ⟨Formals⟩ have one of the following forms:
//
// - (⟨variable_1⟩ ...): The procedure takes a fixed number of arguments; when
//   the procedure is called, the arguments will be stored in fresh locations
//   that are bound to the corresponding variables.
// - ⟨variable⟩: The procedure takes any number of arguments; when the
//   procedure is called, the sequence of actual arguments is converted into a
//   newly allocated list, and the list is stored in a fresh location that is
//   bound to ⟨variable⟩.
// - (⟨variable_1⟩ ... ⟨variable_n⟩ . ⟨variable_n+1⟩): If a space-delimited
//   period precedes the last variable, then the procedure takes n or more
//   arguments, where n is the number of formal arguments before the period
//   (it is an error if there is not at least one). The value stored in the
//   binding of the last variable will be a newly allocated list of the
//   actual arguments left over after all the other actual arguments have been
//   matched up against the other formal arguments.
//
// It is an error for a ⟨variable⟩ to appear more than once in ⟨formals⟩.
func evalLambda(e *Pair, scope *scope) Value {
	args := e.ToVector()
	if len(args) < 3 {
		panic("lamdba must be of the form (lambda ⟨formals⟩ ⟨body⟩)")
	}
	formals, isVariadic := makeFormals(args[1])
	return &procedure{
		name:       "<lambda>",
		closure:    scope,
		formals:    formals,
		isVariadic: isVariadic,
		body:       args[2:],
	}
}

// (if ⟨test⟩ ⟨consequent⟩ ⟨alternate⟩)
// (if ⟨test⟩ ⟨consequent⟩)
//
// Syntax: ⟨Test⟩, ⟨consequent⟩, and ⟨alternate⟩ are expressions.
//
// Semantics: An if expression is evaluated as follows: first, ⟨test⟩ is
// evaluated. If it yields a true value (see section 6.3), then ⟨consequent⟩
// is evaluated and its values are returned. Otherwise ⟨alternate⟩ is
// evaluated and its values are returned. If ⟨test⟩ yields a false value and no
// ⟨alternate⟩ is specified, then the result of the expression is unspecified.
func evalIf(e *Pair, scope *scope, tail bool) Value {
	args := e.ToVector()
	if len(args) < 3 || len(args) > 4 {
		panic("if must be of the form (if ⟨test⟩ ⟨consequent⟩) or (if ⟨test⟩ ⟨consequent⟩ ⟨alternate⟩)")
	}
	if Truthy(eval(args[1], scope, false)) {
		return eval(args[2], scope, tail)
	}
	if len(args) == 3 {
		return nil
	}
	return eval(args[3], scope, tail)
}

// (set! ⟨variable⟩ ⟨expression⟩)
//
// Semantics: ⟨Expression⟩ is evaluated, and the resulting value is stored in
// the location to which ⟨variable⟩ is bound. It is an error if ⟨variable⟩ is
// not bound either in some region enclosing the set! expression or else
// globally. The result of the set! expression is unspecified.
func evalSet(e *Pair, scope *scope) Value {
	args := e.ToVector()
	if len(args) != 3 {
		panic("set! must be of the form (set! ⟨variable⟩ ⟨expression⟩)")
	}
	sym, ok := args[1].(Symbol)
	if !ok {
		panic("set! must be of the form (set! ⟨variable⟩ ⟨expression⟩)")
	}
	if !scope.setIfBound(sym, eval(args[2], scope, false)) {
		panic(fmt.Sprintf("set!: %v is not bound", sym))
	}
	return nil
}

func isElse(clause *Pair) bool {
	sym, ok := clause.car.(Symbol)
	return ok && sym == "else"
}

func evalClause(arg Value, clause *Pair, scope *scope, tail bool) Value {
	expr, _ := clause.cdr.(*Pair)
	if expr == nil {
		return arg
	}

	if sym, ok := expr.car.(Symbol); ok && sym == "=>" {
		if proc, ok := expr.cdr.(*Pair); ok {
			call := &Pair{car: proc, cdr: &Pair{car: arg}}
			return eval(call, scope, tail)
		}
	}

	return evalBegin(clause, scope, tail)
}

func evalCond(e *Pair, scope *scope, tail bool) Value {
	for e, _ = e.cdr.(*Pair); e != nil; e, _ = e.cdr.(*Pair) {
		clause, ok := e.car.(*Pair)
		if !ok {
			panic("cond clause must be of the form (⟨test⟩ ⟨expression1⟩ ...), (⟨test⟩ => ⟨expression⟩), or (else ⟨expression1⟩ ⟨expression2⟩ ...)")
		}

		if e.cdr == nil && isElse(clause) {
			return evalBegin(clause, scope, tail)
		}

		v := eval(clause.car, scope, false)
		if Truthy(v) {
			return evalClause(v, clause, scope, tail)
		}
	}
	return nil
}

func evalCase(e *Pair, scope *scope, tail bool) Value {
	keyp, _ := e.cdr.(*Pair)
	if keyp == nil {
		panic("case must be of the form (case ⟨key⟩ ⟨clause1⟩ ⟨clause2⟩ ...)")
	}

	key := eval(keyp.car, scope, false)

	for clauses, _ := keyp.cdr.(*Pair); clauses != nil; clauses, _ = clauses.cdr.(*Pair) {
		clause, ok := clauses.car.(*Pair)
		if !ok {
			panic("case clause must be of the form ((⟨datum1⟩ ...) ⟨expression1⟩ ⟨expression2⟩ ...), ((⟨datum1⟩ ...) => ⟨expression⟩), (else ⟨expression1⟩ ⟨expression2⟩ ...), or (else => ⟨expression⟩).")
		}

		if clauses.cdr == nil && isElse(clause) {
			return evalClause(key, clause, scope, tail)
		}

		for accepts, _ := clause.car.(*Pair); accepts != nil; accepts, _ = accepts.cdr.(*Pair) {
			if eqv(key, accepts.car) {
				return evalClause(key, clause, scope, tail)
			}
		}
	}
	return nil
}

func evalAnd(e *Pair, scope *scope, tail bool) Value {
	if e.cdr == nil {
		return Boolean(true)
	}

	e, _ = e.cdr.(*Pair)
	if e.cdr == nil {
		return eval(e.car, scope, tail)
	}

	for ; e != nil; e, _ = e.cdr.(*Pair) {
		if !Truthy(eval(e.car, scope, false)) {
			return Boolean(false)
		}
	}
	return Boolean(true)
}

func evalOr(e *Pair, scope *scope, tail bool) Value {
	if e.cdr == nil {
		return Boolean(true)
	}

	e, _ = e.cdr.(*Pair)
	if e.cdr == nil {
		return eval(e.car, scope, tail)
	}

	for ; e != nil; e, _ = e.cdr.(*Pair) {
		v := eval(e.car, scope, false)
		if Truthy(v) {
			return v
		}
	}
	return Boolean(false)
}

func evalBinding(e Value, scope *scope) (Symbol, Value, bool) {
	binding, ok := e.(*Pair)
	if !ok {
		return "", nil, false
	}
	sym, ok := binding.car.(Symbol)
	if !ok {
		return "", nil, false
	}
	init, ok := binding.cdr.(*Pair)
	if !ok {
		return "", nil, false
	}
	if init.cdr != nil {
		return "", nil, false
	}
	return sym, eval(init.car, scope, false), true
}

func evalBindings(e Value, scope *scope, seq bool) ([]Symbol, Vector, bool) {
	if e == nil {
		return nil, nil, true
	}
	bindings, ok := e.(*Pair)
	if !ok {
		return nil, nil, false
	}

	var names []Symbol
	var values Vector
	for {
		sym, value, ok := evalBinding(bindings.car, scope)
		if !ok {
			return nil, nil, false
		}
		if seq {
			scope.set(sym, value)
		}

		names = append(names, sym)
		values = append(values, value)

		if bindings.cdr == nil {
			return names, values, true
		}
		if bindings, ok = bindings.cdr.(*Pair); !ok {
			return nil, nil, false
		}
	}
}

func evalLet(e *Pair, scope *scope, tail bool) Value {
	const invalidLet = "let must be of the form (let ((⟨variable1⟩ ⟨init1⟩) ...) ⟨body⟩)"

	args := e.ToVector()
	if len(args) == 1 {
		panic(invalidLet)
	}
	args = args[1:]

	sym, isNamedLet := args[0].(Symbol)
	if isNamedLet {
		args = args[1:]
		if len(args) == 0 {
			panic(invalidLet)
		}
	}

	formals, actuals, ok := evalBindings(args[0], scope, false)
	if !ok {
		panic(invalidLet)
	}

	scope = scope.push()
	proc := &procedure{
		name:    sym,
		closure: scope,
		formals: formals,
		body:    args[1:],
	}

	if isNamedLet {
		scope.set(sym, proc)
	}

	return proc.apply(actuals)
}

func evalSeq(e *Pair, scope *scope, tail bool) Value {
	if e == nil {
		return nil
	}

	for {
		next, _ := e.cdr.(*Pair)
		if next == nil {
			return eval(e.car, scope, tail)
		}
		eval(e.car, scope, false)
		e = next
	}
}

func evalBegin(e *Pair, scope *scope, tail bool) Value {
	e, _ = e.cdr.(*Pair)
	return evalSeq(e, scope, tail)
}

// A variable definition binds one or more identifiers and specifies an initial
// value for each of them. The simplest kind of variable definition takes one
// of the following forms:
//
// - (define ⟨variable⟩ ⟨expression⟩)
// - (define (⟨variable⟩ ⟨formals⟩) ⟨body⟩)
//
//   ⟨Formals⟩ are either a sequence of zero or more variables, or a sequence of
//   one or more variables followed by a space-delimited period and another
//   variable (as in a lambda expression). This form is equivalent to
//
//       (define ⟨variable⟩
//         (lambda (⟨formals⟩) ⟨body⟩)).
//
// - (define (⟨variable⟩ . ⟨formal⟩) ⟨body⟩)
//
//   ⟨Formal⟩ is a single variable. This form is equivalent to
//
//       (define ⟨variable⟩
//         (lambda ⟨formal⟩ ⟨body⟩)).
//
func evalDefine(e *Pair, scope *scope) Value {
	const invalidDefine = "define must be of the form (define ⟨variable⟩ ⟨expression⟩), (define (⟨variable⟩ ⟨formals⟩) ⟨body⟩), or (define (⟨variable⟩ . ⟨formal⟩) ⟨body⟩)"

	args := e.ToVector()
	if len(args) < 3 {
		panic(invalidDefine)
	}

	switch v := args[1].(type) {
	case Symbol:
		scope.set(v, eval(args[2], scope, false))
		return nil
	case *Pair:
		sym, ok := v.car.(Symbol)
		if !ok {
			panic(invalidDefine)
		}

		formals, isVariadic := makeFormals(v.cdr)
		scope.set(sym, &procedure{
			name:       sym,
			closure:    scope,
			formals:    formals,
			isVariadic: isVariadic,
			body:       args[2:],
		})
		return nil
	default:
		panic(invalidDefine)
	}
}

func evalDefineSyntax(e *Pair, s *scope) Value {
	const invalidDefineSyntax = "define-syntax must be of the form (define-syntax ⟨keyword⟩ ⟨transformer spec⟩)"
	const invalidSyntaxRules = "syntax-rules must be of the form (syntax-rules (⟨literal⟩ ...) ⟨syntaxrule⟩ ...)"
	const invalidRule = "rules must be of the form ((⟨list pattern⟩) ⟨template⟩)"

	args := e.ToVector()
	if len(args) != 3 {
		panic(invalidDefineSyntax)
	}

	keyword, ok := args[1].(Symbol)
	if !ok {
		panic(invalidDefineSyntax)
	}

	spec, ok := args[2].(*Pair)
	if !ok {
		panic(invalidDefineSyntax)
	}

	if spec, ok := spec.car.(Symbol); !ok || spec != "syntax-rules" {
		panic(invalidDefineSyntax)
	}

	// TODO: special ellipsis
	specArgs := spec.ToVector()
	if len(specArgs) < 3 {
		panic(invalidSyntaxRules)
	}

	literalSpec, ok := specArgs[1].(*Pair)
	if !ok && specArgs[1] != nil {
		panic(invalidSyntaxRules)
	}
	literals := map[Symbol]*scope{}
	for _, l := range literalSpec.ToVector() {
		l, ok := l.(Symbol)
		if !ok {
			panic(invalidSyntaxRules)
		}
		literals[l] = s.where(l)
	}

	rules := make([]syntaxRule, len(specArgs[2:]))
	for i, ruleSpec := range specArgs[2:] {
		specPair, ok := ruleSpec.(*Pair)
		if !ok {
			panic(invalidRule)
		}
		ruleArgs := specPair.ToVector()
		if len(ruleArgs) != 2 {
			panic(invalidRule)
		}
		pattern, ok := ruleArgs[0].(*Pair)
		if !ok {
			panic(invalidRule)
		}
		rules[i] = syntaxRule{pattern: pattern, template: ruleArgs[1]}
	}

	s.setKeyword(keyword, &syntaxRules{
		scope:    s,
		literals: literals,
		rules:    rules,
	})

	return nil
}

func eval(expression Value, scope *scope, tail bool) Value {
	if expression == nil {
		return nil
	}

	switch e := expression.(type) {
	case Number, Boolean, Character, String:
		return e
	case Symbol:
		return evalVariable(e, scope)
	case *binding:
		v, _ := e.where.lookup(e.name)
		return v
	case Vector:
		result := make(Vector, len(e))
		for i, v := range e {
			result[i] = eval(v, scope, false)
		}
		return result
	case *Pair:
		switch sym, _ := e.car.(Symbol); sym {
		// primitive expressions
		case "quote":
			return evalQuote(e)
		case "quasiquote":
			return evalQuasiquote(e.cdr.(*Pair).car, scope)
		case "lambda":
			return evalLambda(e, scope)
		case "if":
			v := evalIf(e, scope, tail)
			if tail {
				return v
			}
			return forceTail(v)
		case "set!":
			return evalSet(e, scope)
		case "include":
			panic("NYI: include")
		case "include-ci":
			panic("NYI: include-ci")

		// derived expressions
		case "cond":
			return evalCond(e, scope, tail)
		case "case":
			return evalCase(e, scope, tail)
		case "and":
			return evalAnd(e, scope, tail)
		case "or":
			return evalOr(e, scope, tail)
		case "let":
			return evalLet(e, scope, tail)
		case "begin":
			return evalBegin(e, scope, tail)

		// variable definitions
		case "define":
			return evalDefine(e, scope)

		// syntax definitions
		case "define-syntax":
			return evalDefineSyntax(e, scope)

		// all else
		default:
			if sym, ok := e.car.(Symbol); ok {
				if syntax, ok := scope.lookupKeyword(sym); ok {
					if v, ok := syntax.match(e, scope); ok {
						return eval(v, scope, tail)
					}
				}
			}

			p, ok := eval(e.car, scope, false).(Procedure)
			if !ok {
				panic("value is not a procedure")
			}
			args := e.ToVector()
			actuals := make(Vector, len(args)-1)
			for i, arg := range args[1:] {
				actuals[i] = eval(arg, scope, false)
			}
			if tail {
				return &tailCall{p: p, args: actuals}
			}
			return forceTail(p.Apply(actuals))
		}
	case *tailCall:
		if tail {
			return e
		}
		return forceTail(e.p.Apply(e.args))
	default:
		panic(fmt.Sprintf("unknown expression type %T", e))
	}
}

func forceTail(v Value) Value {
	for {
		tail, ok := v.(*tailCall)
		if !ok {
			return v
		}
		v = tail.p.Apply(tail.args)
	}
}

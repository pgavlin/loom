package loom

import (
	"errors"
	"fmt"
)

type compiler struct {
	body []instruction
}

func compile(expr Value) []instruction {
	var c compiler
	c.compile(expr, true)
	return c.body
}

func compileBody(exprs []Value) []instruction {
	if len(exprs) == 0 {
		return nil
	}

	var c compiler
	for _, expr := range exprs[:len(exprs)-1] {
		c.compile(expr, false)
	}
	c.compile(exprs[len(exprs)-1], true)
	if len(c.body) > 0 && c.body[len(c.body)-1].code != opTail {
		c.append(instruction{opReturn, nil})
	}
	return c.body
}

func (c *compiler) append(instructions ...instruction) {
	c.body = append(c.body, instructions...)
}

// ⟨variable⟩
//
// An expression consisting of a variable (section 3.1) is a variable reference.
// The value of the variable reference is the value stored in the location to
// which the variable is bound. It is an error to reference an unbound variable.
func (c *compiler) compileVariable(e Symbol) {
	c.append(instruction{opGet, e})
}

// (quote ⟨datum⟩)
// ’⟨datum⟩
// ⟨constant⟩
//
// (quote ⟨datum⟩) evaluates to ⟨datum⟩. ⟨Datum⟩ can be any external
// representation of a Scheme object (see section 3.3). This notation is used to
// include literal constants in Scheme code.
func (c *compiler) compileQuote(e *Pair) {
	c.append(instruction{opQuote, e.cdr.(*Pair).car})
}

//func (c *compiler) compileQuasiquote(v Value, vector, list bool) {
//	if v == nil {
//		c.append(instruction{opQuote, nil})
//		return
//	}
//
//	switch v := v.(type) {
//	case Number, Boolean, Character, String, Symbol:
//		c.append(instruction{opQuote, nil})
//	case Vector:
//		c.append(instruction{opQuote, Vector(nil)})
//		for _, v := range v {
//			c.compileQuasiquote(v, true, false)
//			c.append(instruction{opAppend, nil})
//		}
//	case *Pair:
//		switch sym, _ := v.car.(Symbol); sym {
//		case "unquote":
//			c.compile(v.cdr.(*Pair).car, false)
//		case "unquote-splicing":
//			//
//
//
//
//
//			// TODO: splicing
//		case "quasiquote":
//			c.compileQuasiquote(v.cdr.(*Pair).car, false, false)
//
//		// all else
//		default:
//
//			for {
//				switch sym, _ := v.car.(Symbol); sym {
//				case "unquote":
//					p, ok := v.cdr.(*Pair)
//					if ok && p.cdr == nil {
//						c.
//							tail.cdr = eval(p.car, scope, false)
//						return head
//					}
//				case "unquote-splicing":
//					p, ok := v.cdr.(*Pair)
//					if ok && p.cdr == nil {
//						l, ok := eval(p.car, scope, false).(*Pair)
//						if !ok {
//							panic("the argument to unquote-splicing must be a list")
//						}
//						tail.cdr = l
//						return head
//					}
//				}
//
//				elem := evalQuasiquote(v.car, scope)
//				var p, q *Pair
//				if splice, ok := elem.(splice); ok {
//					p, q = splice.p, splice.p
//					for q.cdr != nil {
//						q = q.cdr.(*Pair)
//					}
//				} else {
//					p = &Pair{car: elem}
//					q = p
//				}
//
//				if head == nil {
//					head, tail = p, q
//				} else {
//					tail.cdr, tail = p, q
//				}
//
//				if p, ok := v.cdr.(*Pair); ok {
//					v = p
//					continue
//				}
//
//				tail.cdr = evalQuasiquote(v.cdr, scope)
//				return head
//			}
//		}
//	default:
//		panic(fmt.Sprintf("unknown quasiquote type %T", v))
//	}
//}

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
func (c *compiler) compileLambda(e *Pair) {
	args := e.ToVector()
	if len(args) < 3 {
		panic(errors.New("lamdba must be of the form (lambda ⟨formals⟩ ⟨body⟩)"))
	}

	formals, isVariadic := makeFormals(args[1])
	proc := &compiledProcedure{
		name:       "<lambda>",
		formals:    formals,
		isVariadic: isVariadic,
		body:       compileBody(args[2:]),
	}
	c.append(instruction{opLambda, proc})
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
func (c *compiler) compileIf(e *Pair, tail bool) {
	args := e.ToVector()
	if len(args) < 3 || len(args) > 4 {
		panic("if must be of the form (if ⟨test⟩ ⟨consequent⟩) or (if ⟨test⟩ ⟨consequent⟩ ⟨alternate⟩)")
	}

	if_ := &compiledProcedure{
		name: "<if-true>",
		body: compileBody(args[2:3]),
	}

	else_ := &compiledProcedure{
		name: "<if-false>",
		body: []instruction{
			{opQuote, nil},
			{opReturn, nil},
		},
	}

	if len(args) == 4 {
		else_.body = compileBody(args[3:4])
	}

	c.compile(args[1], false)

	c.append(instruction{opLambda, if_},
		instruction{opLambda, else_},
		instruction{opIf, nil})

	if tail {
		c.append(instruction{opTail, integer(0)})
	} else {
		c.append(instruction{opCall, integer(0)})
	}
}

// (set! ⟨variable⟩ ⟨expression⟩)
//
// Semantics: ⟨Expression⟩ is evaluated, and the resulting value is stored in
// the location to which ⟨variable⟩ is bound. It is an error if ⟨variable⟩ is
// not bound either in some region enclosing the set! expression or else
// globally. The result of the set! expression is unspecified.
func (c *compiler) compileSet(e *Pair) {
	args := e.ToVector()
	if len(args) != 3 {
		panic("set! must be of the form (set! ⟨variable⟩ ⟨expression⟩)")
	}
	sym, ok := args[1].(Symbol)
	if !ok {
		panic("set! must be of the form (set! ⟨variable⟩ ⟨expression⟩)")
	}
	c.compile(args[2], false)
	c.append(instruction{opSet, sym})
}

//func isElse(clause *Pair) bool {
//	sym, ok := clause.car.(Symbol)
//	return ok && sym == "else"
//}
//
//func evalClause(arg Value, clause *Pair, scope *scope, tail bool) Value {
//	expr, _ := clause.cdr.(*Pair)
//	if expr == nil {
//		return arg
//	}
//
//	if sym, ok := expr.car.(Symbol); ok && sym == "=>" {
//		if proc, ok := expr.cdr.(*Pair); ok {
//			call := &Pair{car: proc, cdr: &Pair{car: arg}}
//			return eval(call, scope, tail)
//		}
//	}
//
//	return evalBegin(clause, scope, tail)
//}
//
//func evalCond(e *Pair, scope *scope, tail bool) Value {
//	for e, _ = e.cdr.(*Pair); e != nil; e, _ = e.cdr.(*Pair) {
//		clause, ok := e.car.(*Pair)
//		if !ok {
//			panic("cond clause must be of the form (⟨test⟩ ⟨expression1⟩ ...), (⟨test⟩ => ⟨expression⟩), or (else ⟨expression1⟩ ⟨expression2⟩ ...)")
//		}
//
//		if e.cdr == nil && isElse(clause) {
//			return evalBegin(clause, scope, tail)
//		}
//
//		v := eval(clause.car, scope, false)
//		if Truthy(v) {
//			return evalClause(v, clause, scope, tail)
//		}
//	}
//	return nil
//}
//
//func evalCase(e *Pair, scope *scope, tail bool) Value {
//	keyp, _ := e.cdr.(*Pair)
//	if keyp == nil {
//		panic("case must be of the form (case ⟨key⟩ ⟨clause1⟩ ⟨clause2⟩ ...)")
//	}
//
//	key := eval(keyp.car, scope, false)
//
//	for clauses, _ := keyp.cdr.(*Pair); clauses != nil; clauses, _ = clauses.cdr.(*Pair) {
//		clause, ok := clauses.car.(*Pair)
//		if !ok {
//			panic("case clause must be of the form ((⟨datum1⟩ ...) ⟨expression1⟩ ⟨expression2⟩ ...), ((⟨datum1⟩ ...) => ⟨expression⟩), (else ⟨expression1⟩ ⟨expression2⟩ ...), or (else => ⟨expression⟩).")
//		}
//
//		if clauses.cdr == nil && isElse(clause) {
//			return evalClause(key, clause, scope, tail)
//		}
//
//		for accepts, _ := clause.car.(*Pair); accepts != nil; accepts, _ = accepts.cdr.(*Pair) {
//			if eqv(key, accepts.car) {
//				return evalClause(key, clause, scope, tail)
//			}
//		}
//	}
//	return nil
//}
//
//func evalAnd(e *Pair, scope *scope, tail bool) Value {
//	if e.cdr == nil {
//		return Boolean(true)
//	}
//
//	e, _ = e.cdr.(*Pair)
//	if e.cdr == nil {
//		return eval(e.car, scope, tail)
//	}
//
//	for ; e != nil; e, _ = e.cdr.(*Pair) {
//		if !Truthy(eval(e.car, scope, false)) {
//			return Boolean(false)
//		}
//	}
//	return Boolean(true)
//}
//
//func evalOr(e *Pair, scope *scope, tail bool) Value {
//	if e.cdr == nil {
//		return Boolean(true)
//	}
//
//	e, _ = e.cdr.(*Pair)
//	if e.cdr == nil {
//		return eval(e.car, scope, tail)
//	}
//
//	for ; e != nil; e, _ = e.cdr.(*Pair) {
//		v := eval(e.car, scope, false)
//		if Truthy(v) {
//			return v
//		}
//	}
//	return Boolean(false)
//}
//
//func evalBinding(e Value, scope *scope) (Symbol, Value, bool) {
//	binding, ok := e.(*Pair)
//	if !ok {
//		return "", nil, false
//	}
//	sym, ok := binding.car.(Symbol)
//	if !ok {
//		return "", nil, false
//	}
//	init, ok := binding.cdr.(*Pair)
//	if !ok {
//		return "", nil, false
//	}
//	if init.cdr != nil {
//		return "", nil, false
//	}
//	return sym, eval(init.car, scope, false), true
//}
//
//func evalBindings(e Value, scope *scope, seq bool) ([]Symbol, Vector, bool) {
//	if e == nil {
//		return nil, nil, true
//	}
//	bindings, ok := e.(*Pair)
//	if !ok {
//		return nil, nil, false
//	}
//
//	var names []Symbol
//	var values Vector
//	for {
//		sym, value, ok := evalBinding(bindings.car, scope)
//		if !ok {
//			return nil, nil, false
//		}
//		if seq {
//			scope.set(sym, value)
//		}
//
//		names = append(names, sym)
//		values = append(values, value)
//
//		if bindings.cdr == nil {
//			return names, values, true
//		}
//		if bindings, ok = bindings.cdr.(*Pair); !ok {
//			return nil, nil, false
//		}
//	}
//}
//
//func evalLet(e *Pair, scope *scope, tail bool) Value {
//	const invalidLet = "let must be of the form (let ((⟨variable1⟩ ⟨init1⟩) ...) ⟨body⟩)"
//
//	args := e.ToVector()
//	if len(args) == 1 {
//		panic(invalidLet)
//	}
//	args = args[1:]
//
//	sym, isNamedLet := args[0].(Symbol)
//	if isNamedLet {
//		args = args[1:]
//		if len(args) == 0 {
//			panic(invalidLet)
//		}
//	}
//
//	formals, actuals, ok := evalBindings(args[0], scope, false)
//	if !ok {
//		panic(invalidLet)
//	}
//
//	scope = scope.push()
//	proc := &procedure{
//		name:    sym,
//		closure: scope,
//		formals: formals,
//		body:    args[1:],
//	}
//
//	if isNamedLet {
//		scope.set(sym, proc)
//	}
//
//	return proc.apply(actuals)
//}
//
//func evalSeq(e *Pair, scope *scope, tail bool) Value {
//	if e == nil {
//		return nil
//	}
//
//	for {
//		next, _ := e.cdr.(*Pair)
//		if next == nil {
//			return eval(e.car, scope, tail)
//		}
//		eval(e.car, scope, false)
//		e = next
//	}
//}
//
//func evalBegin(e *Pair, scope *scope, tail bool) Value {
//	e, _ = e.cdr.(*Pair)
//	return evalSeq(e, scope, tail)
//}

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
func (c *compiler) compileDefine(e *Pair) {
	const invalidDefine = "define must be of the form (define ⟨variable⟩ ⟨expression⟩), (define (⟨variable⟩ ⟨formals⟩) ⟨body⟩), or (define (⟨variable⟩ . ⟨formal⟩) ⟨body⟩)"

	args := e.ToVector()
	if len(args) < 3 {
		panic(invalidDefine)
	}

	switch v := args[1].(type) {
	case Symbol:
		c.compile(args[2], false)
		c.append(instruction{opDefine, v})
	case *Pair:
		sym, ok := v.car.(Symbol)
		if !ok {
			panic(invalidDefine)
		}

		formals, isVariadic := makeFormals(v.cdr)
		proc := &compiledProcedure{
			name:       sym,
			formals:    formals,
			isVariadic: isVariadic,
			body:       compileBody(args[2:]),
		}
		c.append(instruction{opLambda, proc})
		c.append(instruction{opDefine, sym})
	default:
		panic(invalidDefine)
	}
}

func (c *compiler) compile(expression Value, tail bool) {
	if expression == nil {
		c.append(instruction{opQuote, nil})
		return
	}

	switch e := expression.(type) {
	case Number, Boolean, Character, String:
		c.append(instruction{opQuote, e})
	case Symbol:
		c.compileVariable(e)
	case Vector:
		c.append(instruction{opQuote, Vector(nil)})
		for _, v := range e {
			c.compile(v, false)
		}
		c.append(instruction{opVector, integer(len(e))})
	case *Pair:
		switch sym, _ := e.car.(Symbol); sym {
		// primitive expressions
		case "quote":
			c.compileQuote(e)
			//		case "quasiquote":
			//			return evalQuasiquote(e.cdr.(*Pair).car, scope)
		case "lambda":
			c.compileLambda(e)
		case "if":
			c.compileIf(e, tail)
		case "set!":
			c.compileSet(e)
		case "include":
			panic("NYI: include")
		case "include-ci":
			panic("NYI: include-ci")

		// derived expressions
		//		case "cond":
		//			return evalCond(e, scope, tail)
		//		case "case":
		//			return evalCase(e, scope, tail)
		//		case "and":
		//			return evalAnd(e, scope, tail)
		//		case "or":
		//			return evalOr(e, scope, tail)
		//		case "let":
		//			return evalLet(e, scope, tail)
		//		case "begin":
		//			return evalBegin(e, scope, tail)

		// variable definitions
		case "define":
			c.compileDefine(e)

		// all else
		default:
			c.compile(e.car, false)
			args := e.ToVector()[1:]
			for _, arg := range args {
				c.compile(arg, false)
			}
			if tail {
				c.append(instruction{opTail, integer(len(args))})
			} else {
				c.append(instruction{opCall, integer(len(args))})
			}
		}
	default:
		panic(fmt.Sprintf("unknown expression type %T", e))
	}
}

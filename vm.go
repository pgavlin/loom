package loom

import (
	"fmt"
)

type opcode byte

const (
	opQuote opcode = iota
	opGet
	opBinding
	opVector
	opList
	opLambda
	opIf
	opSet
	opDefine
	opCall
	opTail
	opReturn
)

type instruction struct {
	code      opcode
	immediate Value
}

type integer int

func (i integer) MarshalSExp() SExpression {
	return Cons(Symbol("integer"), Cons(NewInt(int64(i)), nil))
}

type compiledProcedure struct {
	name       Symbol
	formals    []Symbol
	isVariadic bool
	body       []instruction
}

func (*compiledProcedure) MarshalSExp() SExpression {
	return Symbol("<compiled procedure>")
}

var callCC = &compiledClosure{
	proc: &compiledProcedure{
		name:    "call-with-current-continuation",
		formals: []Symbol{"procedure", "continuation"},
		body: []instruction{
			{opGet, Symbol("procedure")},
			{opGet, Symbol("continuation")},
			{opTail, integer(1)},
		},
	},
	scope: globalScope,
}

type compiledClosure struct {
	proc  *compiledProcedure
	scope *scope
}

func (*compiledClosure) MarshalSExp() SExpression {
	return Symbol("<compiled closure>")
}

func (c *compiledClosure) Apply(args Vector) Value {
	scope := c.scope.push()

	var vm vm
	vm.assignFormals(c.proc, scope, args)
	vm.init(c, scope)
	return vm.run()
}

type continuation struct {
	stack *frame
	arity int
}

func (c *continuation) MarshalSExp() SExpression {
	return Symbol("<continuation>")
}

func (c *continuation) Apply(args Vector) Value {
	if len(args) != c.arity {
		s := ""
		if c.arity > 1 {
			s = "s"
		}
		panic(fmt.Errorf("continuation expects %v argument%s", c.arity, s))
	}

	// copy the stack, push the argument, and resume
	m := vm{stack: c.stack.copyStack()}
	m.stack.stack = append(m.stack.stack, args[0])
	return m.run()
}

type frame struct {
	caller  *frame
	closure *compiledClosure
	scope   *scope
	stack   Vector
	pc      int
}

func (f *frame) copy() *frame {
	s := make(Vector, len(f.stack))
	copy(s, f.stack)

	return &frame{
		closure: f.closure,
		scope:   f.scope,
		stack:   s,
		pc:      f.pc,
	}
}

func (f *frame) copyStack() *frame {
	// copy each frame
	top := f.copy()
	for f, s := top, f.caller; s != nil; f, s = f.caller, s.caller {
		f.caller = s.copy()
	}
	return top
}

type vm struct {
	stack *frame
}

func (*vm) assignFormals(p *compiledProcedure, scope *scope, args Vector) {
	formals, atLeast := p.formals, ""
	if p.isVariadic {
		formals, atLeast = formals[:len(formals)-1], " at least"
	}

	if len(args) < len(formals) {
		panic(fmt.Errorf("%v expects%v %d arguments", p.name, atLeast, len(formals)))
	}

	for i, sym := range formals {
		scope.set(sym, args[i])
	}

	if p.isVariadic {
		scope.set(p.formals[len(p.formals)-1], args[len(formals):].ToList())
	}
}

func (m *vm) init(closure *compiledClosure, scope *scope) {
	m.stack = &frame{closure: closure, scope: scope}
}

func (m *vm) run() Value {
	body := m.stack.closure.proc.body
	scope := m.stack.scope
	stack := ([]Value)(nil)
	pc := 0

	for {
		inst := &body[pc]
		switch inst.code {
		case opQuote:
			// push immediate
			stack = append(stack, inst.immediate)
		case opGet:
			// push value
			sym := inst.immediate.(Symbol)
			value, ok := scope.lookup(sym)
			if !ok {
				panic(fmt.Errorf("%v is not bound", string(sym)))
			}
			stack = append(stack, value)
		case opBinding:
			b := inst.immediate.(*binding)
			value, _ := b.where.lookup(b.name)
			stack = append(stack, value)
		case opVector:
			// pop n values, push vector
			n := int(inst.immediate.(integer))
			v := make(Vector, n)
			copy(v, stack[len(stack)-n:])
			stack = stack[:len(stack)-n]
			stack = append(stack, v)
		case opList:
			// (v_0 . (v_1 . (v_2 ... (v_n-1 . v_n))))
			n := int(inst.immediate.(integer))

			tail := stack[len(stack)-1]
			values := stack[len(stack)-n-1 : len(stack)-1]
			stack = stack[:len(stack)-n-1]

			head := tail
			for i := len(values) - 1; i >= 0; i-- {
				head = Cons(values[i], tail)
			}
			stack = append(stack, head)
		case opLambda:
			// push a new closure
			proc := inst.immediate.(*compiledProcedure)
			stack = append(stack, &compiledClosure{proc: proc, scope: scope})
		case opIf:
			// pop condition, pop then, pop else, push choice
			else_, if_, cond := stack[len(stack)-1], stack[len(stack)-2], stack[len(stack)-3]
			stack = stack[:len(stack)-2]
			if Truthy(cond) {
				stack[len(stack)-1] = if_
			} else {
				stack[len(stack)-1] = else_
			}
		case opSet:
			// pop value, set symbol
			sym := inst.immediate.(Symbol)
			value := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if !scope.setIfBound(sym, value) {
				panic(fmt.Errorf("set!: %v is not bound", sym))
			}
		case opDefine:
			// pop value, define symbol
			sym := inst.immediate.(Symbol)
			value := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			scope.set(sym, value)
		case opCall:
			// pop args, closure, push frame, assign formals, jump
			nargs := int(inst.immediate.(integer))
			args := Vector(stack[len(stack)-nargs:])
			stack = stack[:len(stack)-nargs]

			proc := stack[len(stack)-1].(Procedure)
			stack = stack[:len(stack)-1]

			switch proc := proc.(type) {
			case *compiledClosure:
				m.stack.stack, m.stack.pc = stack, pc

				if proc == callCC {
					// push new continuation
					args = append(args, &continuation{
						stack: m.stack.copyStack(),
						arity: int(inst.immediate.(integer)),
					})
				}

				scope = proc.scope.push()
				m.assignFormals(proc.proc, scope, args)

				m.stack = &frame{
					caller:  m.stack,
					closure: proc,
				}
				body, stack, pc = proc.proc.body, nil, -1
			case *continuation:
				// replace the current stack with the continuation

				if len(args) != proc.arity {
					s := ""
					if proc.arity > 1 {
						s = "s"
					}
					panic(fmt.Errorf("continuation expects %v argument%s", proc.arity, s))
				}

				m.stack = proc.stack.copyStack()
				body, scope, stack, pc = m.stack.closure.proc.body, m.stack.scope, m.stack.stack, m.stack.pc

				stack = append(stack, args...)
			default:
				stack = append(stack, proc.Apply(args))
			}
		case opTail:
			// pop args, closure, pop frame, push frame, jump
			nargs := int(inst.immediate.(integer))
			args := Vector(stack[len(stack)-nargs:])
			stack = stack[:len(stack)-nargs]

			proc := stack[len(stack)-1].(Procedure)
			stack = stack[:len(stack)-1]

			switch proc := proc.(type) {
			case *compiledClosure:
				m.stack.stack, m.stack.pc = stack, pc

				if proc == callCC {
					// push new continuation
					args = append(args, &continuation{
						stack: m.stack.copyStack(),
						arity: int(inst.immediate.(integer)),
					})
				}

				scope = proc.scope.push()

				m.assignFormals(proc.proc, scope, args)

				m.stack = &frame{
					caller:  m.stack.caller,
					closure: proc,
				}
				body, stack, pc = proc.proc.body, nil, -1
			case *continuation:
				// replace the current stack with the continuation

				if len(args) != proc.arity {
					s := ""
					if proc.arity > 1 {
						s = "s"
					}
					panic(fmt.Errorf("continuation expects %v argument%s", proc.arity, s))
				}

				m.stack = proc.stack.copyStack()
				body, scope, stack, pc = m.stack.closure.proc.body, m.stack.scope, m.stack.stack, m.stack.pc

				stack = append(stack, args...)
			default:
				v := proc.Apply(args)
				caller := m.stack.caller

				if caller == nil {
					return v
				}

				m.stack = caller
				scope, body, stack, pc = caller.closure.scope, caller.closure.proc.body, caller.stack, caller.pc
				stack = append(stack, v)
			}
		case opReturn:
			// pop value, pop frame, continue
			v := stack[len(stack)-1]
			caller := m.stack.caller

			if caller == nil {
				return v
			}

			m.stack = caller
			scope, body, stack, pc = caller.closure.scope, caller.closure.proc.body, caller.stack, caller.pc
			stack = append(stack, v)
		default:
			panic(fmt.Errorf("unexpected opcode %v", inst.code))
		}

		pc++
	}
}

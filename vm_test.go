package loom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVM(t *testing.T) {
	root := &compiledClosure{
		scope: globalScope,
		proc: &compiledProcedure{
			name: "test",
			body: []instruction{
				{opQuote, NewInt(42)},
				{opDefine, Symbol("a")},
				{opQuote, NewInt(24)},
				{opDefine, Symbol("b")},
				{opLambda, &compiledProcedure{
					name: "<lambda>",
					body: []instruction{
						{opGet, Symbol("+")},
						{opGet, Symbol("a")},
						{opGet, Symbol("b")},
						{opTail, integer(2)},
					},
				}},
				{opTail, integer(0)},
			},
		},
	}

	v := root.Apply(nil)
	assert.True(t, eq(NewInt(66), v))

	scope := globalScope.push()
	scope.set("call/cc", callCC)

	root = &compiledClosure{
		scope: scope,
		proc: &compiledProcedure{
			name: "test2",
			body: []instruction{
				{opGet, Symbol("*")},
				{opQuote, NewInt(2)},
				{opGet, Symbol("call/cc")},
				{opLambda, &compiledProcedure{
					name:    "<lambda>",
					formals: []Symbol{"c"},
					body: []instruction{
						{opGet, Symbol("c")},
						{opQuote, NewInt(33)},
						{opCall, integer(1)},
						{opQuote, NewInt(21)},
						{opReturn, nil},
					},
				}},
				{opCall, integer(1)},
				{opTail, integer(2)},
			},
		},
	}

	v = root.Apply(nil)
	assert.True(t, eq(NewInt(66), v))
}

func TestCompile(t *testing.T) {
	expr, err := ParseString(`((lambda (a b) ((lambda () (+ a b))) ) 42 24)`)
	require.NoError(t, err)

	scope := globalScope.push()
	scope.set("call/cc", callCC)

	root := &compiledClosure{
		scope: scope,
		proc: &compiledProcedure{
			name: "<stdin>",
			body: compileBody([]Value{expr}),
		},
	}

	v := root.Apply(nil)
	assert.True(t, eq(NewInt(66), v))

	expr, err = ParseString(`(* 2 (call/cc (lambda (c) (c 33))))`)
	require.NoError(t, err)

	root.proc.body = compileBody([]Value{expr})

	v = root.Apply(nil)
	assert.True(t, eq(NewInt(66), v))

	expr, err = ParseString(`((lambda (x)
		(define (fac n acc)
			(if
				(= n 0) acc
				(fac (- n 1) (* n acc))))
		(fac x 1)) 4)`)
	require.NoError(t, err)

	root.proc.body = compileBody([]Value{expr})

	v = root.Apply(nil)
	assert.True(t, eq(NewInt(24), v))
}

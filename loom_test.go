package loom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testExpr(t *testing.T, expr, expectedExpr string, globalPairs ...interface{}) {
	defer func() {
		if x := recover(); x != nil {
			t.Fatalf("panic: %v", x)
		}
	}()

	e := NewEnv()

	globals := map[Symbol]Value{}
	require.Zero(t, len(globalPairs)%2, "len(globalPairs) must be even")

	for i := 0; i < len(globalPairs); i += 2 {
		key := globalPairs[i]
		switch k := key.(type) {
		case Symbol:
			// OK
		case string:
			key = Symbol(k)
		default:
			t.Fatalf("global names must be strings or symbols")
		}

		value := globalPairs[i+1]
		switch v := value.(type) {
		case Value:
			// OK
		case string:
			x, err := ParseString(v)
			require.NoError(t, err)
			value = e.Eval(x)
		default:
			t.Fatalf("global values must be Values or strings")
		}

		globals[key.(Symbol)] = value.(Value)
	}
	e = e.With(globals)

	actualx, err := ParseString(expr)
	require.NoError(t, err)
	expectedx, err := ParseString(expectedExpr)
	require.NoError(t, err)
	actual := e.Eval(actualx)
	expected := e.Eval(expectedx)
	if !assert.True(t, Truthy(Equal(Vector{actual, expected}))) {
		assert.Equal(t, EncodeToString(expected), EncodeToString(actual))
	}
}

func TestSmoke(t *testing.T) {
	cases := []struct{ name, expr, expected string }{
		{
			"pair",
			"'(1 . 2)",
			"'(1 . 2)",
		},
		{
			"identity",
			"((lambda (x) x) 42)",
			"42",
		},
		{
			"identity-2",
			"((lambda () ((lambda (x) x) 42)))",
			"42",
		},
		{
			"if-t",
			"(if #t 42)",
			"42",
		},
		{
			"define-x",
			"((lambda () (define x 42) x))",
			"42",
		},
		{
			"factorial",
			`((lambda (n)
								(define (factorial-loop n acc)
									(if (= n 0) acc
										(factorial-loop (- n 1) (* n acc))))
								(factorial-loop n 1))
							4)`,
			"24",
		},
		{
			"quasiquote",
			`(quasiquote (a ,((lambda (n)
								(define (factorial-loop n acc)
									(if (= n 0) acc
										(factorial-loop (- n 1) (* n acc))))
								(factorial-loop n 1))
							4) b))`,
			"'(a 24 b)",
		},
		{
			"let-cond-1",
			`(let ((x 24)) (cond ((= x 24) x) ((= x 42) 1) (else 0)))`,
			"24",
		},
		{
			"let-cond-2",
			`(let ((x 42)) (cond ((= x 24) x) ((= x 42) 1) (else 0)))`,
			"1",
		},
		{
			"let-cond-3",
			`(let ((x 42)) (cond ((= x 24) x) ((= x 43) 1) (else 0)))`,
			"0",
		},
		{
			"list-tail",
			`(list-tail (list 1 2 3 4 5) 2)`,
			"'(3 4 5)",
		},
		{
			"mergesort",
			`(begin
						(define sort #f)
						(define merge #f)
						(let ()
						  (define dosort
							(lambda (pred? ls n)
							  (if (= n 1)
								  (list (car ls))
								  (let ((i (quotient n 2)))
									(domerge pred?
											 (dosort pred? ls i)
											 (dosort pred? (list-tail ls i) (- n i)))))))
						  (define domerge
							(lambda (pred? l1 l2)
							  (cond
								((null? l1) l2)
								((null? l2) l1)
								((pred? (car l2) (car l1))
								 (cons (car l2) (domerge pred? l1 (cdr l2))))
								(else (cons (car l1) (domerge pred? (cdr l1) l2))))))
						  (set! sort
							(lambda (pred? l)
							  (if (null? l) l (dosort pred? l (length l)))))
						  (set! merge
							(lambda (pred? l1 l2)
							  (domerge pred? l1 l2))))
						(sort < '(5 4 3 2 1)))`,
			"'(1 2 3 4 5)",
		},
		{
			"interpret",
			`(begin
				(define interpret #f)
				(let ()
				  ;; primitive-environment contains a small number of primitive
				  ;; procedures; it can be extended easily with additional primitives.
				  (define primitive-environment
					(quasiquote ((apply . ,apply) (assq . ,assq)
						  (car . ,car) (cdr . ,cdr) (cons . ,cons)
						  (eq? . ,eq?) (list . ,list) (null? . ,null?)
						  (pair? . ,pair?) (set-car! . ,set-car!)
						  (set-cdr! . ,set-cdr!) (symbol? . ,symbol?))))

				  ;; new-env returns a new environment from a formal parameter
				  ;; specification, a list of actual parameters, and an outer
				  ;; environment.  The symbol? test identifies "improper"
				  ;; argument lists.  Environments are association lists,
				  ;; associating variables with values.
				  (define new-env
					(lambda (formals actuals env)
					  (cond
						((null? formals) env)
						((symbol? formals) (cons (cons formals actuals) env))
						(else
						 (cons (cons (car formals) (car actuals))
							   (new-env (cdr formals) (cdr actuals) env))))))

				  ;; lookup finds the value of the variable var in the environment
				  ;; env, using assq.  Assumes var is bound in env.
				  (define lookup
					(lambda (var env)
					  (cdr (assq var env))))

				  ;; assign is similar to lookup but alters the binding of the
				  ;; variable var by changing the cdr of the association pair
				  (define assign
					(lambda (var val env)
					  (set-cdr! (assq var env) val)))

				  ;; exec evaluates the expression, recognizing all core forms.
				  (define exec
					(lambda (exp env)
					  (cond
						((symbol? exp) (lookup exp env))
						((pair? exp)
						 (case (car exp)
						   ((quote) (car (cdr exp)))
						   ((lambda)
							(lambda vals
							  (let ((env (new-env (car (cdr exp)) vals env)))
								(let loop ((exps (cdr (cdr exp))))
								   (if (null? (cdr exps))
									   (exec (car exps) env)
									   (begin
										  (exec (car exps) env)
										  (loop (cdr exps))))))))
						   ((if)
							(if (exec (car (cdr exp)) env)
								(exec (car (cdr (cdr exp))) env)
								(exec (car (cdr (cdr (cdr exp)))) env)))
						   ((set!)
							(assign (car (cdr exp))
									(exec (car (cdr (cdr exp))) env)
									env))
						   (else
							(apply (exec (car exp) env)
								   (map (lambda (x) (exec x env))
										(cdr exp))))))
						(else exp))))

				  ;; interpret starts execution with the primitive environment.
				  (set! interpret
					(lambda (exp)
					  (exec exp primitive-environment))))

				(interpret
				  '((lambda (reverse)
					  (set! reverse
						(lambda (ls new)
						  (if (null? ls)
							  new
							  (reverse (cdr ls) (cons (car ls) new)))))
					  (reverse '(a b c d e) '()))
				 #f)))`,
			"'(e d c b a)",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testExpr(t, c.expr, c.expected)
		})
	}
}

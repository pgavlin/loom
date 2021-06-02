package loom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyntaxRules(t *testing.T) {
	// (define-syntax and
	//       (syntax-rules ()
	//         ((and) #t)
	//         ((and test) test)
	//         ((and test1 test2 ...)
	//          (if test1 (and test2 ...) #f))))

	x, err := ParseString(
		`(define-syntax and
			(syntax-rules ()
				((and) #t)
				((and test) test)
				((and test1 test2 ...)
					(if test1 (and test2 ...) #f))))`)
	require.NoError(t, err)

	scope := globalScope.push()

	_ = eval(x, scope, false)

	rules, ok := scope.syntax["and"]
	require.True(t, ok)

	parse := func(s string) *Pair {
		v, err := ParseString(s)
		require.NoError(t, err)
		return v.(*Pair)
	}

	v, ok := rules.match(parse("(and)"), globalScope)
	if assert.True(t, ok) {
		assert.True(t, equal(v, Boolean(true), map[Value]struct{}{}))
	}

	v, ok = rules.match(parse("(and #t)"), globalScope)
	if assert.True(t, ok) {
		assert.True(t, equal(v, Boolean(true), map[Value]struct{}{}))
	}

	v, ok = rules.match(parse("(and #t #f)"), globalScope)
	if assert.True(t, ok) {
		assert.True(t, equal(v, parse("(if #t #f #f)"), map[Value]struct{}{}))
		assert.Equal(t, "(if #t #f #f)", EncodeToString(v))
	}

	begin, err := ParseString(
		`(define-syntax begin
		  (syntax-rules ()
			((begin exp ...)
			 ((lambda () exp ...)))))`)
	require.NoError(t, err)

	_ = eval(begin, scope, false)

	rules, ok = scope.syntax["begin"]
	require.True(t, ok)

	cond, err := ParseString(
		`(define-syntax cond
			  (syntax-rules (else =>)
				((cond (else result1 result2 ...))
				 (begin result1 result2 ...))
				((cond (test => result))
				 (let ((temp test))
				   (if temp (result temp))))
				((cond (test => result) clause1 clause2 ...)
				 (let ((temp test))
				   (if temp
					   (result temp)
					   (cond clause1 clause2 ...))))
				((cond (test)) test)
				((cond (test) clause1 clause2 ...)
				 (let ((temp test))
                   (if temp
				       temp
					   (cond clause1 clause2 ...))))
				((cond (test result1 result2 ...))
				 (if test (begin result1 result2 ...)))
				((cond (test result1 result2 ...)
					   clause1 clause2 ...)
				 (if test
					 (begin result1 result2 ...)
					 (cond clause1 clause2 ...)))))`)
	require.NoError(t, err)

	_ = eval(cond, scope, false)

	rules, ok = scope.syntax["cond"]
	require.True(t, ok)

	v, ok = rules.match(parse("(cond ((> 3 2) 'greater) ((< 3 2) 'less))"), globalScope)
	if assert.True(t, ok) {
		assert.True(t, equal(v, parse("(if (> 3 2) (begin (quote greater)) (if (< 3 2) (begin (quote less))))"), map[Value]struct{}{}))
		assert.Equal(t, "(if (> 3 2) (begin (quote greater)) (if (< 3 2) (begin (quote less))))", EncodeToString(v))
	}
}

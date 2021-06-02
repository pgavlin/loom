package loom

import (
	"fmt"
	"strings"
)

func VectorPred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	_, ok := args[0].(Vector)
	return Boolean(ok)
}

func VectorRef(args Vector) Value {
	if len(args) != 2 {
		panic("vector-ref expects 2 arguments")
	}

	v, ok := args[0].(Vector)
	if !ok {
		panic("the first argument to vector-ref must be a vector")
	}
	n, ok := args[1].(Number)
	if !ok {
		panic("the second argument to vector-ref must be an integer")
	}
	i, ok := n.Int()
	if !ok {
		panic("the second argument to vector-ref must be an integer")
	}
	if i > int64(len(v)) {
		panic(fmt.Sprintf("%v is not a member of a vector of length %v", i, len(v)))
	}
	return v[i]
}

func VectorAppend(args Vector) Value {
	if len(args) == 0 {
		return Vector(nil)
	}

	result, ok := args[0].(Vector)
	if !ok {
		panic("arguments to vector-append must be vectors")
	}
	for _, a := range args[1:] {
		v, ok := a.(Vector)
		if !ok {
			panic("arguments to vector-append must be vectors")
		}
		result = append(result, v...)
	}
	return result
}

func VectorToString(args Vector) Value {
	if len(args) != 1 {
		panic("vector->string expects 1 argument")
	}

	v, ok := args[0].(Vector)
	if !ok {
		panic("the argument to vector->string must be a vector of characters")
	}
	var b strings.Builder
	b.Grow(len(v))
	for _, v := range v {
		c, ok := v.(Character)
		if !ok {
			panic("the argument to vector->string must be a vector of characters")
		}
		b.WriteRune(rune(c))
	}
	return String(b.String())
}

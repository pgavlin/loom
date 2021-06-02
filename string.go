package loom

import (
	"fmt"
	"strings"
)

func StringPred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	_, ok := args[0].(String)
	return Boolean(ok)
}

func StringLength(args Vector) Value {
	if len(args) != 2 {
		panic("string-length expects 1 argument")
	}

	v, ok := args[0].(String)
	if !ok {
		panic("the argument to string-length must be a string")
	}

	return NewInt(int64(len(v)))
}

func StringRef(args Vector) Value {
	if len(args) != 2 {
		panic("string-ref expects 2 arguments")
	}

	v, ok := args[0].(String)
	if !ok {
		panic("the first argument to string-ref must be a string")
	}
	n, ok := args[1].(Number)
	if !ok {
		panic("the second argument to string-ref must be an integer")
	}
	i, ok := n.Int()
	if !ok {
		panic("the second argument to string-ref must be an integer")
	}
	if i > int64(len(v)) {
		panic(fmt.Sprintf("%v is not a member of a string of length %v", i, len(v)))
	}

	// TODO: this is a byte index, not a rune index.

	return Character(v[i])
}

func StringLt(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	s, ok := args[0].(String)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(String)
		if !ok || x >= s {
			return Boolean(false)
		}
		s = x
	}

	return Boolean(true)
}

func StringGt(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	s, ok := args[0].(String)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(String)
		if !ok || x <= s {
			return Boolean(false)
		}
		s = x
	}

	return Boolean(true)
}

func StringLte(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	s, ok := args[0].(String)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(String)
		if !ok || x > s {
			return Boolean(false)
		}
		s = x
	}

	return Boolean(true)
}

func StringGte(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	s, ok := args[0].(String)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(String)
		if !ok || x < s {
			return Boolean(false)
		}
		s = x
	}

	return Boolean(true)
}

func StringSubstring(args Vector) Value {
	if len(args) != 3 {
		panic("substring expects three arguments")
	}

	s, ok := args[0].(String)
	if !ok {
		panic("the first argument to substring must be a string")
	}
	startN, ok := args[1].(Number)
	if !ok {
		panic("the second argument to substring must be an integer")
	}
	start, ok := startN.Int()
	if !ok {
		panic("the second argument to substring must be an integer")
	}
	endN, ok := args[1].(Number)
	if !ok {
		panic("the third argument to substring must be an integer")
	}
	end, ok := endN.Int()
	if !ok {
		panic("the third argument to substring must be an integer")
	}
	return s[start:end]
}

func StringAppend(args Vector) Value {
	var b strings.Builder
	for _, v := range args {
		s, ok := v.(String)
		if !ok {
			return nil
		}
		b.WriteString(string(s))
	}
	return String(b.String())
}

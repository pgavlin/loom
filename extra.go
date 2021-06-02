package loom

import "strings"

func Repr(args Vector) Value {
	if len(args) != 1 {
		panic("repr expects 1 argument")
	}
	return String(EncodeToString(args[0]))
}

func StringTrimSuffix(args Vector) Value {
	if len(args) != 2 {
		panic("string-trim-suffix expects 2 arguments")
	}
	s, ok := args[0].(String)
	if !ok {
		panic("the first argument to string-trim-suffix must be a string")
	}
	cut, ok := args[1].(String)
	if !ok {
		panic("the second argument to string-trim-suffix must be a string")
	}
	return String(strings.TrimSuffix(string(s), string(cut)))
}

func StringContains(args Vector) Value {
	if len(args) != 2 {
		panic("string-contains expects 2 arguments")
	}
	s, ok := args[0].(String)
	if !ok {
		panic("the first argument to string-contains must be a string")
	}
	needle, ok := args[1].(String)
	if !ok {
		panic("the second argument to string-contains must be a string")
	}
	return Boolean(strings.Contains(string(s), string(needle)))
}

func StringReplace(args Vector) Value {
	if len(args) != 3 {
		panic("string-replace expects 3 arguments")
	}
	s, ok := args[0].(String)
	if !ok {
		panic("the first argument to string-replace must be a string")
	}
	old, ok := args[1].(String)
	if !ok {
		panic("the second argument to string-replace must be a string")
	}
	new, ok := args[2].(String)
	if !ok {
		panic("the third argument to string-replace must be a string")
	}
	return String(strings.ReplaceAll(string(s), string(old), string(new)))
}

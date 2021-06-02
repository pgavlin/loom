package loom

func SymbolPred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	_, ok := args[0].(Symbol)
	return Boolean(ok)
}

func SymbolToString(args Vector) Value {
	if len(args) != 1 {
		panic("symbol->string expects one argument")
	}
	sym, ok := args[0].(Symbol)
	if !ok {
		panic("symbol->string expects a symbol")
	}
	return String(sym)
}

func StringToSymbol(args Vector) Value {
	if len(args) != 1 {
		panic("string->symbol expects one argument")
	}
	str, ok := args[0].(String)
	if !ok {
		panic("string->symbol expects a string")
	}
	return Symbol(str)
}

package loom

var globalScope = &scope{env: map[Symbol]Value{
	// equality predicates
	"eqv?":   ProcedureFunc(Eqv),
	"eq?":    ProcedureFunc(Eq),
	"equal?": ProcedureFunc(Equal),

	// numerics
	"number?":           ProcedureFunc(NumberPred),
	"=":                 ProcedureFunc(NumberEq),
	"<":                 ProcedureFunc(NumberLt),
	">":                 ProcedureFunc(NumberGt),
	"<=":                ProcedureFunc(NumberLte),
	">=":                ProcedureFunc(NumberGte),
	"+":                 ProcedureFunc(NumberAdd),
	"*":                 ProcedureFunc(NumberMul),
	"-":                 ProcedureFunc(NumberSub),
	"/":                 ProcedureFunc(NumberDiv),
	"truncate-quotient": ProcedureFunc(NumberTruncateQuotient),
	"quotient":          ProcedureFunc(NumberTruncateQuotient),

	// booleans
	"boolean?": ProcedureFunc(BooleanPred),
	"not":      ProcedureFunc(BooleanNot),

	// pairs and lists
	"pair?":     ProcedureFunc(PairPred),
	"cons":      ProcedureFunc(PairCons),
	"car":       ProcedureFunc(PairCar),
	"cdr":       ProcedureFunc(PairCdr),
	"set-car!":  ProcedureFunc(PairSetCar),
	"set-cdr!":  ProcedureFunc(PairSetCdr),
	"null?":     ProcedureFunc(NullPred),
	"list":      ProcedureFunc(ListConstructor),
	"length":    ProcedureFunc(ListLength),
	"append":    ProcedureFunc(ListAppend),
	"assq":      ProcedureFunc(ListAssq),
	"list-tail": ProcedureFunc(ListTail),
	"list-ref":  ProcedureFunc(ListRef),

	// symbols
	"symbol?":        ProcedureFunc(SymbolPred),
	"symbol->string": ProcedureFunc(SymbolToString),
	"string->symbol": ProcedureFunc(StringToSymbol),

	// strings
	"string?":       ProcedureFunc(StringPred),
	"string-length": ProcedureFunc(StringLength),
	"string-ref":    ProcedureFunc(StringRef),
	"string<?":      ProcedureFunc(StringLt),
	"string>?":      ProcedureFunc(StringGt),
	"string<=?":     ProcedureFunc(StringLte),
	"string>=?":     ProcedureFunc(StringGte),
	"string-append": ProcedureFunc(StringAppend),
	"substring":     ProcedureFunc(StringSubstring),

	// vectors
	"vector?":        ProcedureFunc(VectorPred),
	"vector-ref":     ProcedureFunc(VectorRef),
	"vector-append":  ProcedureFunc(VectorAppend),
	"vector->string": ProcedureFunc(VectorToString),

	// Control funcitons
	"apply": ProcedureFunc(ProcedureApply),
	"map":   ProcedureFunc(ProcedureMap),

	// extras
	"repr":               ProcedureFunc(Repr),
	"string-trim-suffix": ProcedureFunc(StringTrimSuffix),
	"string-contains":    ProcedureFunc(StringContains),
	"string-replace":     ProcedureFunc(StringReplace),
}}

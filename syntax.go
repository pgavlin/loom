package loom

type syntaxRules struct {
	scope    *scope
	literals map[Symbol]*scope
	rules    []syntaxRule
}

func (r *syntaxRules) match(form *Pair, scope *scope) (Value, bool) {
	m := syntaxMatcher{
		literals:  r.literals,
		ruleScope: r.scope,
		formScope: scope,
	}
	for i := range r.rules {
		m.rule = &r.rules[i]
		if v, ok := m.match(form); ok {
			return v, true
		}
	}
	return nil, false
}

type syntaxRule struct {
	pattern  *Pair
	template Value
}

type syntaxMatcher struct {
	literals  map[Symbol]*scope
	ruleScope *scope
	formScope *scope
	rule      *syntaxRule
}

func (m *syntaxMatcher) match(form *Pair) (Value, bool) {
	// ignore the first element of the pattern and the form
	pattern, _ := m.rule.pattern.cdr.(*Pair)
	form, _ = form.cdr.(*Pair)

	// create a new scope for the pattern's bindings
	bindings := &scope{env: map[Symbol]Value{}}

	if pattern == nil {
		if form != nil {
			return nil, false
		}
	} else {
		if !m.matchPattern(pattern, form, bindings) {
			return nil, false
		}
	}

	// emit the template
	return m.emitTemplate(m.rule.template, bindings), true
}

func (m *syntaxMatcher) emitTemplate(template Value, bindings *scope) Value {
	if template == nil {
		return nil
	}

	switch t := template.(type) {
	case Symbol:
		if where, ok := m.literals[t]; ok {
			if where != nil {
				return &binding{where: where, name: t}
			}
			return t
		}

		if v, ok := bindings.lookup(t); ok {
			return v
		}
		return t
	case *Pair:
		var head, tail *Pair
		for {
			// check for an ellipsis
			next, more := t.cdr.(*Pair)
			if more && next.car == Symbol("...") {
				sym, ok := t.car.(Symbol)
				if ok {
					// emit the matches
					matches, _ := bindings.lookup(sym)
					for _, m := range matches.(Vector) {
						p := &Pair{car: m}
						if head == nil {
							head, tail = p, p
						} else {
							tail.cdr, tail = p, p
						}
					}
				}
				t = next
				next, more = next.cdr.(*Pair)
			} else {
				p := &Pair{car: m.emitTemplate(t.car, bindings)}
				if head == nil {
					head, tail = p, p
				} else {
					tail.cdr, tail = p, p
				}
			}

			if more {
				t = next
				continue
			}

			if t.cdr != nil {
				tail.cdr = m.emitTemplate(t.cdr, bindings)
			}

			if sym, ok := head.car.(Symbol); ok {
				if syntax, ok := m.ruleScope.lookupKeyword(sym); ok {
					if v, ok := syntax.match(head, m.ruleScope); ok {
						return v
					}
				}
			}

			return head
		}
	case Vector:
		var result Vector
		for len(t) > 0 {
			// check for an ellipsis
			if len(t) > 1 && t[1] == Symbol("...") {
				ellipsis := t[0]
				t = t[2:]

				sym, ok := ellipsis.(Symbol)
				if ok {
					matches, _ := bindings.lookup(sym)
					result = append(result, matches.(Vector)...)
				}
			} else {
				result = append(result, m.emitTemplate(t[0], bindings))
				t = t[1:]
			}
		}
		return result
	default:
		return t
	}
}

func (m *syntaxMatcher) matchPattern(pattern, form Value, bindings *scope) bool {
	if pattern == nil {
		return form == nil
	}

	switch p := pattern.(type) {
	case Symbol:
		if p == "_" {
			return true
		}
		if where, ok := m.literals[p]; ok {
			form, ok := form.(Symbol)
			return ok && p == form && m.formScope.where(form) == where
		}
		bindings.set(p, form)
		return true
	case *Pair:
		form, ok := form.(*Pair)
		if !ok {
			return false
		}

		scope := bindings.push()
		for {
			// check for an ellipsis
			next, more := p.cdr.(*Pair)
			matchedEllipsis := false
			if more && next.car == Symbol("...") {
				ellipsis := p.car

				// determine how many matches we need
				matches := form.len() - next.len() + 1
				if matches < 0 {
					return false
				}

				var matched Vector
				if matches > 0 {
					for {
						if !m.matchPattern(ellipsis, form.car, scope) {
							return false
						}
						matched = append(matched, form.car)

						matches--
						if matches == 0 {
							break
						}
						form, _ = form.cdr.(*Pair)
					}
				}

				if sym, ok := ellipsis.(Symbol); ok {
					if _, ok := m.literals[sym]; !ok {
						scope.set(sym, matched)
					}
				}

				p, matchedEllipsis = next, true
				next, more = next.cdr.(*Pair)
			} else if form == nil || !m.matchPattern(p.car, form.car, scope) {
				return false
			}

			if more {
				nextForm, ok := form.cdr.(*Pair)
				if !ok && nextForm != nil {
					return false
				}
				p, form = next, nextForm
				continue
			}

			if form == nil {
				if !matchedEllipsis && p.cdr == nil {
					return false
				}
			} else if !m.matchPattern(p.cdr, form.cdr, scope) {
				return false
			}

			// copy out this pattern's bindings.
			for k, v := range scope.env {
				bindings.env[k] = v
			}
			return true
		}
	case Vector:
		form, ok := form.(Vector)
		if !ok {
			return false
		}

		scope := bindings.push()
		for {
			if len(p) == 0 {
				if len(form) != 0 {
					return false
				}

				// copy out this pattern's bindings.
				for k, v := range scope.env {
					bindings.env[k] = v
				}
				return true
			}

			// check for an ellipsis
			if len(p) > 1 && p[1] == Symbol("...") {
				ellipsis := p[0]
				p = p[2:]

				// determine how many matches we need
				matches := len(form) - len(p)
				if matches < 0 {
					return false
				}

				matched := form[:matches]
				for _, form := range matched {
					if !m.matchPattern(ellipsis, form, scope) {
						return false
					}
				}

				if sym, ok := ellipsis.(Symbol); ok {
					if _, ok := m.literals[sym]; !ok {
						scope.set(sym, matched)
					}
				}

				form = form[matches:]
				continue
			}

			if len(form) == 0 || !m.matchPattern(p[0], form[0], scope) {
				return false
			}

			p, form = p[1:], form[1:]
		}

	default:
		return equal(p, form, map[Value]struct{}{})
	}
}

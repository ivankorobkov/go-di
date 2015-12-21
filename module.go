package di

import (
	"fmt"
	"reflect"
	"runtime"
)

type ModuleFunc func(m *Module)

func (m ModuleFunc) Name() string {
	return runtime.FuncForPC(reflect.ValueOf(m).Pointer()).Name()
}

type Module struct {
	imports map[string]ModuleFunc
	types   map[reflect.Type]*typ
}

func newModule(f ModuleFunc) *Module {
	m := &Module{
		imports: map[string]ModuleFunc{},
		types:   map[reflect.Type]*typ{},
	}
	f(m)
	return m
}

func (m *Module) Import(f ModuleFunc) {
	if f == nil {
		panic("nil module")
	}

	name := f.Name()
	if _, ok := m.imports[name]; ok {
		return
	}

	m.imports[name] = f
}

func (m *Module) Add(f interface{}) {
	typ := newTypFromFunc(f)

	if _, ok := m.types[typ.typ]; ok {
		panic(fmt.Sprintf("modules: duplicate constructor for type %v", typ))
	}
	m.types[typ.typ] = typ
}

type typ struct {
	typ         reflect.Type
	deps        []reflect.Type
	constructor reflect.Value // function
}

func newTypFromFunc(f interface{}) *typ {
	constructor := reflect.ValueOf(f)
	if constructor.Kind() != reflect.Func {
		panic("init: constructor is not a function")
	}

	ctype := constructor.Type()
	if ctype.NumOut() != 1 {
		panic(fmt.Sprintf(`init: constructor must return one value "%v"`, getFuncName(constructor)))
	}

	rtype := ctype.Out(0)
	deps := []reflect.Type{}
	for i := 0; i < ctype.NumIn(); i++ {
		deps = append(deps, ctype.In(i))
	}

	return &typ{
		typ:         rtype,
		deps:        deps,
		constructor: constructor,
	}
}

func getFuncName(v reflect.Value) string {
	return runtime.FuncForPC(v.Pointer()).Name()
}

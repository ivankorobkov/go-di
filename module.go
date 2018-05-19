package di

import (
	"fmt"
	"reflect"
)

// ModuleFunc defines a module provider.
type ModuleFunc func(*Module)

func (m ModuleFunc) Name() string {
	return getFuncName(reflect.ValueOf(m))
}

// Module groups providers, dependencies and imports.
type Module struct {
	Name      string
	Imports   []ModuleFunc
	Providers []*Provider
	Deps      []reflect.Type
}

func newModule(f ModuleFunc) *Module {
	m := &Module{
		Name:      getFuncName(reflect.ValueOf(f)),
		Imports:   []ModuleFunc{},
		Providers: []*Provider{},
		Deps:      []reflect.Type{},
	}
	f(m)
	return m
}

// Add ands a new provider.
func (m *Module) Add(f interface{}) {
	p := newProvider(m, f)
	m.add(p)
}

// AddInstance adds a new instance provider.
func (m *Module) AddInstance(instance interface{}) {
	p := newInstanceProvider(m, instance)
	m.add(p)
}

func (m *Module) add(p *Provider) {
	for _, p0 := range m.Providers {
		if p0.Type == p.Type {
			panic(fmt.Errorf("di: duplicate provider, type=%v module=%v", p.Type, m.Name))
		}
	}
	m.Providers = append(m.Providers, p)
}

// Dep adds a dependency which will be provided at a context level, not via imported modules.
func (m *Module) Dep(dep interface{}) {
	typ := reflect.TypeOf(dep)
	for _, typ0 := range m.Deps {
		if typ == typ0 {
			panic(fmt.Errorf("di: duplicate dependency, type=%v module=%v", typ, m.Name))
		}
	}

	m.Deps = append(m.Deps, typ)
}

// Import adds another module to this module dependencies.
func (m *Module) Import(module ModuleFunc) {
	if module == nil {
		panic("di: nil module")
	}

	name := module.Name()
	for _, imp := range m.Imports {
		if imp.Name() == name {
			panic(fmt.Errorf("di: duplicate import, import=%v module=%v", name, m.Name))
		}
	}

	m.Imports = append(m.Imports, module)
}

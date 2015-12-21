package di

import (
	"fmt"
	"reflect"
)

type Package struct {
	modules map[string]*Module
	types   map[reflect.Type]*typ
}

func NewPackage(mm ...ModuleFunc) *Package {
	p := &Package{
		modules: map[string]*Module{},
		types:   map[reflect.Type]*typ{},
	}

	for _, m := range mm {
		p.initModule(m)
	}
	return p
}

func (p *Package) Build() Graph {
	g := Graph{}
	for _, typ := range p.types {
		g.instantiate(p, typ)
	}
	return g
}

func (p *Package) initModule(f ModuleFunc) {
	name := f.Name()
	if _, ok := p.modules[name]; ok {
		return
	}

	module := newModule(f)
	p.modules[name] = module

	// Imports
	for _, d := range module.imports {
		p.initModule(d)
	}

	// Types.
	for _, typ := range module.types {
		if _, ok := p.types[typ.typ]; ok {
			panic(fmt.Sprintf("modules: duplicate constructor for type %v", typ.typ))
		}

		p.types[typ.typ] = typ
	}
}

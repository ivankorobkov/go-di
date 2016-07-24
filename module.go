package di

import (
	"fmt"
	"reflect"
)

type ModuleFunc func(m *Module)

func (m ModuleFunc) Name() string {
	return getFuncName(reflect.ValueOf(m))
}

type Module struct {
	Name         string
	Imports      []ModuleFunc
	Constructors []*Constructor
	PackageDeps  []reflect.Type
}

func newModule(f ModuleFunc) *Module {
	m := &Module{
		Name:         getFuncName(reflect.ValueOf(f)),
		Imports:      []ModuleFunc{},
		Constructors: []*Constructor{},
		PackageDeps:  []reflect.Type{},
	}
	f(m)
	return m
}

// Add adds a new function as a type constructor.
func (m *Module) AddConstructor(f interface{}) {
	c := newConstructor(m, f)
	m.add(c)
}

// AddInstance adds an instance as a constructor.
func (m *Module) AddInstance(instance interface{}) {
	c := newConstructorFromInstance(m, instance)
	m.add(c)
}

// AddInstanceWithFields adds an instance and all its public fields as constructors.
func (m *Module) AddInstanceWithFields(instance interface{}) {
	m.AddInstance(instance)

	val := reflect.ValueOf(instance)
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		m.AddInstance(field.Interface())
	}
}

func (m *Module) add(c *Constructor) {
	for _, c0 := range m.Constructors {
		if c0.Type == c.Type {
			panic(fmt.Errorf("di: Duplicate constructor in a module: type=%v module=%v", c.Type, m.Name))
		}
	}
	m.Constructors = append(m.Constructors, c)
}

// MarkPackageDep adds a dependency which must be resolved at a package level, not via imported modules.
func (m *Module) MarkPackageDep(dep interface{}) {
	typ := reflect.TypeOf(dep)
	m.markPackageDep(typ)
}

// MarkPackageDeps adds all struct fields as dependencies which must be resolved at a package level.
func (m *Module) MarkPackageDeps(structWithDeps interface{}) {
	structType := reflect.TypeOf(structWithDeps)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		m.markPackageDep(field.Type)
	}
}

func (m *Module) markPackageDep(typ reflect.Type) {
	for _, typ0 := range m.PackageDeps {
		if typ == typ0 {
			panic(fmt.Errorf("di: Duplicate package-level dependency in a module: type=%v module=%v", typ, m.Name))
		}
	}

	m.PackageDeps = append(m.PackageDeps, typ)
}

// Import imports a module which provides the dependencies of this module.
func (m *Module) Import(f ModuleFunc) {
	if f == nil {
		panic("di: Tried to import a nil module")
	}

	name := f.Name()
	for _, imp := range m.Imports {
		if imp.Name() == name {
			panic(fmt.Errorf("di: Duplicate import in a module: import=%v module=%v", name, m.Name))
		}
	}

	m.Imports = append(m.Imports, f)
}

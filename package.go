package di

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

type Package struct {
	Modules      map[string]*Module
	Constructors map[reflect.Type]*Constructor
}

func NewPackage(moduleFuncs ...ModuleFunc) (*Package, error) {
	p := &Package{
		Modules:      map[string]*Module{},
		Constructors: map[reflect.Type]*Constructor{},
	}

	if err := p.initModules(moduleFuncs); err != nil {
		return nil, err
	}
	if err := p.initConstructors(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Package) initModules(moduleFuncs []ModuleFunc) error {
	for _, mfunc := range moduleFuncs {
		prevNames := []string{}
		if _, err := p.initModule(mfunc, prevNames); err != nil {
			return err
		}
	}
	return nil
}

func (p *Package) initModule(mfunc ModuleFunc, prevNames []string) (*Module, error) {
	name := mfunc.Name()
	if m, ok := p.Modules[name]; ok {
		return m, nil
	}

	// Prevent cyclic imports.
	{
		path := []string{name}
		for i := len(prevNames) - 1; i >= 0; i-- {
			prev := prevNames[i]
			path = append(path, prev)

			if prev == name {
				return nil, fmt.Errorf("di: Cyclic import in modules: %v", strings.Join(path, " -> "))
			}
		}
	}
	prevNames = append(prevNames, name)

	// Start module initialization.
	m := newModule(mfunc)

	// Resolve imported modules.
	for _, ifunc := range m.Imports {
		if _, err := p.initModule(ifunc, prevNames); err != nil {
			return nil, err
		}
	}

	// Add the initialized module to the package.
	p.Modules[name] = m
	return m, nil
}

func (p *Package) initConstructors() error {
	// Add constructors to the package, prevent duplicates.
	for _, m := range p.Modules {
		for _, c := range m.Constructors {
			if c1, ok := p.Constructors[c.Type]; ok {
				err := fmt.Errorf("di: Duplicate constructors: type=%v module0=%v module1=%v", c.Type, c.Module, c1.Module)
				return err
			}

			p.Constructors[c.Type] = c
		}
	}

	// Check constructor dependencies.
	for _, m := range p.Modules {
		availableDeps := map[reflect.Type]bool{}

		// Collect imported dependencies.
		for _, imp := range m.Imports {
			impModule := p.Modules[imp.Name()]
			for _, dep := range impModule.Constructors {
				availableDeps[dep.Type] = true
			}
		}

		// Collect this module constructors as dependencies.
		for _, c := range m.Constructors {
			availableDeps[c.Type] = true
		}

		// Collect package-level dependencies.
		for _, dep := range m.PackageDeps {
			if _, ok := p.Constructors[dep]; !ok {
				err := fmt.Errorf("di: Unresolved package-level dependency: dep=%v module=%v", dep, m.Name)
				return err
			}

			availableDeps[dep] = true
		}

		// Check constructors dependencies.
		for _, c := range m.Constructors {
			for _, dep := range c.Deps {
				if _, ok := availableDeps[dep]; !ok {
					err := fmt.Errorf("di: Unresolved dependency: dep=%v constructor=%v module=%v", dep, c, m.Name)
					return err
				}
			}
		}
	}

	return nil
}

func (p *Package) WriteDot(io.Writer) error {
	return nil
}

func (p *Package) MarshalDot() string {
	return ""
}

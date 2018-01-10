package di

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// Fill creates a new context and fills a struct public fields.
func Fill(dstPtr interface{}, modules ...ModuleFunc) error {
	ctx, err := NewContext(modules...)
	if err != nil {
		return err
	}
	ctx.Fill(dstPtr)
	return nil
}

// MustFill creates a new context and fills a struct public fields or panics on an error.
func MustFill(dstPtr interface{}, modules ...ModuleFunc) {
	if err := Fill(dstPtr, modules...); err != nil {
		panic(err)
	}
}

// Context is an initialized application context with modules, providers and instances.
type Context struct {
	Modules   map[string]*Module
	Providers map[reflect.Type]*Provider
	Instances map[reflect.Type]interface{}
	mu        sync.Mutex // Guards instances.
}

func NewContext(moduleFuncs ...ModuleFunc) (*Context, error) {
	ctx := &Context{
		Modules:   map[string]*Module{},
		Providers: map[reflect.Type]*Provider{},
		Instances: map[reflect.Type]interface{}{},
	}

	// Recursively initialize the modules and their providers.
	for _, mfunc := range moduleFuncs {
		if err := ctx.addModule(mfunc, []string{}); err != nil {
			return nil, err
		}
	}

	// Check provider dependencies.
	if err := ctx.checkProviderDependencies(); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (ctx *Context) addModule(mfunc ModuleFunc, importPath []string) error {
	name := mfunc.Name()
	if _, ok := ctx.Modules[name]; ok {
		return nil
	}

	// Prevent cyclic imports.
	{
		path := []string{name}
		for i := len(importPath) - 1; i >= 0; i-- {
			prev := importPath[i]
			path = append(path, prev)

			if prev == name {
				return fmt.Errorf("di: cyclic module imports: %v", strings.Join(path, " -> "))
			}
		}
	}
	importPath = append(importPath, name)

	// Initialize the module.
	module := mfunc()
	module.Name = name

	// Resolve the module imports.
	for _, ifunc := range module.Imports {
		if err := ctx.addModule(ifunc, importPath); err != nil {
			return err
		}
	}

	// Add the module to the context.
	ctx.Modules[name] = module

	// Initialize the module providers.
	module.ProviderMap = map[reflect.Type]*Provider{}
	for _, pfunc := range module.Providers {
		provider, err := ctx.addProvider(module, pfunc)
		if err != nil {
			return err
		}
		module.ProviderMap[provider.Type] = provider
	}

	// Initialize the module external dependencies.
	module.DependencyMap = map[reflect.Type]struct{}{}
	for _, dep := range module.Dependencies {
		module.DependencyMap[reflect.TypeOf(dep)] = struct{}{}
	}

	return nil
}

func (ctx *Context) addProvider(module *Module, pfunc interface{}) (*Provider, error) {
	// Initialize a provider.
	provider, err := newProvider(module, pfunc)
	if err != nil {
		return nil, err
	}

	// Prevent providers for duplicate types.
	provider0 := ctx.Providers[provider.Type]
	if provider0 != nil {
		return nil, fmt.Errorf("di: duplicate provider for type %v in modules %v and %v",
			provider.Type, provider.Module, provider0.Module)
	}

	ctx.Providers[provider.Type] = provider
	return provider, nil
}

func (ctx *Context) checkProviderDependencies() error {
	for _, module := range ctx.Modules {
		// Collect available types inside a module.
		available := map[reflect.Type]bool{}

		// Collect imported providers.
		for _, ifunc := range module.Imports {
			imodule := ctx.Modules[ifunc.Name()]
			for _, iprovider := range imodule.ProviderMap {
				available[iprovider.Type] = true
			}
		}

		// Collect this module providers.
		for _, provider := range module.ProviderMap {
			available[provider.Type] = true
		}

		// Collect external dependencies.
		for deptype, _ := range module.DependencyMap {
			if _, ok := ctx.Providers[deptype]; !ok {
				err := fmt.Errorf("di: unresolved external dependency %v in module %v", deptype, module)
				return err
			}

			available[deptype] = true
		}

		// Check provider dependencies.
		for _, provider := range module.ProviderMap {
			for _, deptype := range provider.Dependencies {
				if _, ok := available[deptype]; !ok {
					return fmt.Errorf("di: unresolved dependency %v for provider %v in module %v", deptype, provider, module)
				}
			}
		}
	}
	return nil
}

// Get returns an instance of a given type, or an error.
func (ctx *Context) Get(v interface{}) (interface{}, error) {
	type0 := reflect.TypeOf(v)
	return ctx.GetByType(type0)
}

// GetByType returns an instance by a type, or an error.
func (ctx *Context) GetByType(type0 reflect.Type) (interface{}, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.getOrInit(type0, make(map[reflect.Type]struct{}))
}

func (ctx *Context) getOrInit(type0 reflect.Type, path map[reflect.Type]struct{}) (interface{}, error) {
	instance, ok := ctx.Instances[type0]
	if ok {
		return instance, nil
	}

	provider, ok := ctx.Providers[type0]
	if !ok {
		return nil, fmt.Errorf("di: no provider for type %v", type0)
	}
	instance, err := provider.create(ctx, path)
	if err != nil {
		return nil, err
	}

	ctx.Instances[type0] = instance
	return instance, nil
}

// Fill fills public struct fields with instances from this context.
func (ctx *Context) Fill(dstPtr interface{}) {
	v := reflect.ValueOf(dstPtr).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		ftype := field.Type()
		instance, ok := ctx.Instances[ftype]
		if !ok {
			continue
		}

		field.Set(reflect.ValueOf(instance))
	}
}

type ModuleFunc func() *Module

func (fn ModuleFunc) Name() string {
	return getFuncName(fn)
}

type Module struct {
	Name         string
	Imports      []ModuleFunc
	Providers    []interface{}
	Dependencies []interface{}

	ProviderMap   map[reflect.Type]*Provider
	DependencyMap map[reflect.Type]struct{}
}

func (m *Module) String() string {
	return m.Name
}

func (m *Module) Import(mfunc ModuleFunc) {
	m.Imports = append(m.Imports, mfunc)
}

func (m *Module) Depend(v interface{}) {
	m.Dependencies = append(m.Dependencies, v)
}

func (m *Module) Add(provider interface{}) {
	m.Providers = append(m.Providers, provider)
}

type Provider struct {
	Module       *Module
	Type         reflect.Type
	Function     reflect.Value
	Dependencies []reflect.Type
}

func newProvider(module *Module, fn interface{}) (*Provider, error) {
	fval := reflect.ValueOf(fn)
	if fval.Kind() != reflect.Func {
		return nil, fmt.Errorf("di: provider must be a function, invalid provider %T in module %v", fn, module)
	}
	ftype := fval.Type()

	// Result
	var resultType reflect.Type
	numOut := ftype.NumOut()
	switch numOut {
	case 1:
		resultType = ftype.Out(0)
	case 2:
		resultType = ftype.Out(0)
		// TODO: Check error type
		// errorType := ftype.Out(1)
	default:
		return nil, fmt.Errorf("di: provider must be a function with (instance) or (instance, error) return signature, invalid provider %v in module %v, ", fn, module)
	}

	// Dependencies
	dependencies := []reflect.Type{}
	for i := 0; i < ftype.NumIn(); i++ {
		dependencies = append(dependencies, ftype.In(i))
	}

	return &Provider{
		Module:       module,
		Type:         resultType,
		Dependencies: dependencies,
		Function:     fval,
	}, nil
}

func (p *Provider) create(ctx *Context, path map[reflect.Type]struct{}) (interface{}, error) {
	// Get or init dependencies from the context.
	args := make([]reflect.Value, len(p.Dependencies))
	for i, dtype := range p.Dependencies {
		arg, err := ctx.getOrInit(dtype, path)
		if err != nil {
			return nil, err
		}

		args[i] = reflect.ValueOf(arg)
	}

	result := p.Function.Call(args)
	switch p.Function.Type().NumOut() {
	case 1:
		return result[0].Interface(), nil
	case 2:
		// TODO: Cast and check.
		var err error = nil
		if !result[1].IsNil() {
			err = result[1].Interface().(error)
		}
		return result[0].Interface(), err
	default:
		panic("di: unexpected number of results")
	}
}

// getFuncName returns a function name.
func getFuncName(fn interface{}) string {
	fval := reflect.ValueOf(fn)
	return runtime.FuncForPC(fval.Pointer()).Name()
}

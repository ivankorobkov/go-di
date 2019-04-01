package di

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// Context is a dependency injection context.
type Context struct {
	Modules       map[string]*Module
	Providers     map[reflect.Type]*Provider
	Instances     map[reflect.Type]interface{}
	InstanceSlice []interface{} // Ordered from dependencies to dependants.
}

// Inject creates a context and injects dependencies into public struct fields.
func Inject(dstPtr interface{}, depOrMods ...interface{}) error {
	ctx, err := NewContext(depOrMods...)
	if err != nil {
		return err
	}

	ctx.Inject(dstPtr)
	return nil
}

// MustInject creates a context and injects dependencies into public struct fields, or panics on an error.
func MustInject(dstPtr interface{}, depOrMods ...interface{}) {
	if err := Inject(dstPtr, depOrMods...); err != nil {
		panic(err)
	}
}

// NewContext creates a context and initializes all instances from its providers.
func NewContext(depOrMods ...interface{}) (*Context, error) {
	ctx := &Context{
		Modules:   make(map[string]*Module),
		Providers: make(map[reflect.Type]*Provider),
		Instances: make(map[reflect.Type]interface{}),
	}

	mods := make([]ModuleFunc, 0, len(depOrMods))
	for _, depOrMod := range depOrMods {
		mod, ok := depOrMod.(func(*Module))
		if ok {
			mods = append(mods, mod)
		} else {
			dep := depOrMod
			mod := func(m *Module) {
				m.AddInstance(dep)
			}
			mods = append(mods, mod)
		}
	}

	if err := ctx.initModules(mods); err != nil {
		return nil, err
	}
	if err := ctx.initProviders(); err != nil {
		return nil, err
	}
	if err := ctx.initInstances(); err != nil {
		return nil, err
	}
	return ctx, nil
}

// Get returns an instance from this context of a given type.
func (ctx *Context) Get(dstPtr interface{}) bool {
	t := reflect.TypeOf(dstPtr).Elem()
	instance, ok := ctx.Instances[t]
	if !ok {
		return false
	}

	v := reflect.ValueOf(instance)
	reflect.ValueOf(dstPtr).Elem().Set(v)
	return true
}

// GetMust returns an instance from this context of a given type or panics if absents.
func (ctx *Context) MustGet(dstPtr interface{}) {
	if !ctx.Get(dstPtr) {
		panic(fmt.Sprintf("di: no instance, type=%T", dstPtr))
	}
}

// Inject injects dependencies into public struct fields.
func (ctx *Context) Inject(structPtr interface{}) {
	v := reflect.ValueOf(structPtr).Elem()

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

func (ctx *Context) initModules(mfuncs []ModuleFunc) error {
	for _, mfunc := range mfuncs {
		prevNames := []string{}
		if _, err := ctx.initModule(mfunc, prevNames); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *Context) initModule(mfunc ModuleFunc, prevNames []string) (*Module, error) {
	name := mfunc.Name()
	if m, ok := ctx.Modules[name]; ok {
		return m, nil
	}

	// Prevent cyclic imports.
	{
		path := []string{name}
		for i := len(prevNames) - 1; i >= 0; i-- {
			prev := prevNames[i]
			path = append(path, prev)

			if prev == name {
				return nil, fmt.Errorf("di: cyclic import %v", strings.Join(path, " -> "))
			}
		}
	}
	prevNames = append(prevNames, name)

	// Start module initialization.
	m := newModule(mfunc)

	// Resolve imported modules.
	for _, impfunc := range m.Imports {
		if _, err := ctx.initModule(impfunc, prevNames); err != nil {
			return nil, err
		}
	}

	// Add the initialized module to the context.
	ctx.Modules[name] = m
	return m, nil
}

func (ctx *Context) initProviders() error {
	// Add providers to the package, prevent duplicates.
	for _, m := range ctx.Modules {
		for _, p := range m.Providers {
			if p1, ok := ctx.Providers[p.Type]; ok {
				return fmt.Errorf("di: duplicate provider, type=%v, module0=%v, module1=%v",
					p.Type, p.Module.Name, p1.Module.Name)
			}

			ctx.Providers[p.Type] = p
		}
	}

	// Check provider dependencies.
	for _, m := range ctx.Modules {
		availableDeps := map[reflect.Type]bool{}

		// Add providers from the imported modules.
		for _, imp := range m.Imports {
			impModule := ctx.Modules[imp.Name()]
			for _, dep := range impModule.Providers {
				availableDeps[dep.Type] = true
			}
		}

		// Add this module providers.
		for _, p := range m.Providers {
			availableDeps[p.Type] = true
		}

		// Add existing explicit dependencies.
		for _, dep := range m.Deps {
			_, ok := ctx.Providers[dep]
			if ok {
				availableDeps[dep] = true
			}
		}

		// Check provider dependencies.
		// for _, p := range m.Providers {
		// 	for _, dep := range p.Deps {
		// 		if _, ok := availableDeps[dep]; !ok {
		// 			return fmt.Errorf(
		// 				"di: unresolved provider dependency, dep=%v, provider=%v, module=%v",
		// 				dep, p, m.Name)
		// 		}
		// 	}
		// }
	}

	return nil
}

func (ctx *Context) initInstances() error {
	for _, p := range ctx.Providers {
		if _, err := ctx.initInstance(p.Type); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *Context) initInstance(typ reflect.Type) (interface{}, error) {
	instance, ok := ctx.Instances[typ]
	if ok {
		return instance, nil
	}

	p, ok := ctx.Providers[typ]
	if !ok {
		return nil, fmt.Errorf("di: no provider, type=%v", typ)
	}

	args := []interface{}{}
	for _, dep := range p.Deps {
		arg, err := ctx.initInstance(dep)
		if err != nil {
			return nil, err
		}

		args = append(args, arg)
	}

	instance, err := p.Func(args)
	if err != nil {
		return nil, err
	}

	ctx.Instances[typ] = instance
	ctx.InstanceSlice = append(ctx.InstanceSlice, instance)
	return instance, nil
}

func getFuncName(fval reflect.Value) string {
	return runtime.FuncForPC(fval.Pointer()).Name()
}

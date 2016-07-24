package di

import (
	"fmt"
	"reflect"
)

// Constructor is a single type specification in a module.
// It has a result type, dependencies, and a function which
// returns an instance using an object graph for dependencies.
type Constructor struct {
	Module *Module
	Name   string
	Type   reflect.Type
	Deps   []reflect.Type
	Func   func(args []interface{}) interface{}
}

func (c *Constructor) String() string {
	return c.Name
}

// NewConstructor creates a new constructor from a function with injected dependencies,
// for example, newServiceZ(ServiceA, ServiceB) ServiceZ.
func newConstructor(module *Module, f interface{}) *Constructor {
	fval := reflect.ValueOf(f)
	if fval.Kind() != reflect.Func {
		panic(fmt.Sprintf("di: Constructor must be a function: %T", f))
	}
	ftyp := fval.Type()

	// Result
	if ftyp.NumOut() != 1 {
		fname := getFuncName(fval)
		panic(fmt.Sprintf(`di: Constructor must return (result) or (result, error): %v`, fname))
	}
	rtype := ftyp.Out(0)

	// Deps
	deps := []reflect.Type{}
	for i := 0; i < ftyp.NumIn(); i++ {
		deps = append(deps, ftyp.In(i))
	}

	// Function
	function := func(args []interface{}) interface{} {
		argv := []reflect.Value{}
		for _, arg := range args {
			argv = append(argv, reflect.ValueOf(arg))
		}

		result := fval.Call(argv)[0]
		return result.Interface()
	}

	return &Constructor{
		Module: module,
		Name:   getFuncName(fval),
		Type:   rtype,
		Deps:   deps,
		Func:   function,
	}
}

// NewConstructorFromInstance creates a constructor which always returns the same instance.
func newConstructorFromInstance(module *Module, instance interface{}) *Constructor {
	typ := reflect.TypeOf(instance)
	return &Constructor{
		Module: module,
		Name:   typ.String(),
		Type:   typ,
		Deps:   []reflect.Type{},
		Func: func([]interface{}) interface{} {
			return instance
		},
	}
}

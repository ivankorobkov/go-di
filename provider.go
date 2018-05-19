package di

import (
	"fmt"
	"reflect"
)

// Provider creates a service instance.
type Provider struct {
	Module *Module
	Name   string
	Type   reflect.Type
	Deps   []reflect.Type
	Func   func(args []interface{}) (interface{}, error)
}

func (c *Provider) String() string {
	return c.Name
}

// newProvider creates a new constructor from a function with injected dependencies,
// for example, newServiceZ(ServiceA, ServiceB) ServiceZ.
func newProvider(module *Module, f interface{}) *Provider {
	fval := reflect.ValueOf(f)
	if fval.Kind() != reflect.Func {
		panic(fmt.Sprintf("di: provider must be a function: %T", f))
	}
	ftyp := fval.Type()

	// Result
	switch ftyp.NumOut() {
	case 1, 2:
	default:
		fname := getFuncName(fval)
		panic(fmt.Sprintf(`di: provider must return (instance) or (instance, error): %v`, fname))
	}
	rtype := ftyp.Out(0)

	// Deps
	deps := []reflect.Type{}
	for i := 0; i < ftyp.NumIn(); i++ {
		deps = append(deps, ftyp.In(i))
	}

	// Function
	function := func(args []interface{}) (interface{}, error) {
		argv := []reflect.Value{}
		for _, arg := range args {
			argv = append(argv, reflect.ValueOf(arg))
		}

		out := fval.Call(argv)
		if len(out) == 1 {
			return out[0].Interface(), nil
		}

		result := out[0].Interface()
		err := out[1].Interface().(error)
		return result, err
	}

	return &Provider{
		Module: module,
		Name:   getFuncName(fval),
		Type:   rtype,
		Deps:   deps,
		Func:   function,
	}
}

func newInstanceProvider(module *Module, instance interface{}) *Provider {
	typ := reflect.TypeOf(instance)
	return &Provider{
		Module: module,
		Name:   typ.String(),
		Type:   typ,
		Deps:   []reflect.Type{},
		Func: func([]interface{}) (interface{}, error) {
			return instance, nil
		},
	}
}

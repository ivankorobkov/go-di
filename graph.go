package di

import (
	"fmt"
	"reflect"
	"runtime"
)

// New creates a new object graph from module funcs.
func New(moduleFuncs ...ModuleFunc) (*Graph, error) {
	p, err := NewPackage(moduleFuncs...)
	if err != nil {
		return nil, err
	}

	return NewGraph(p)
}

// Fill creates a new object graph and fills dstPtr public fields.
func Fill(dstPtr interface{}, moduleFuncs ...ModuleFunc) error {
	graph, err := New(moduleFuncs...)
	if err != nil {
		return err
	}

	graph.Fill(dstPtr)
	return nil
}

// MustFill creates a new object graph and fills dstPtr public fields or panics on an error.
func MustFill(dstPtr interface{}, moduleFuncs ...ModuleFunc) {
	if err := Fill(dstPtr, moduleFuncs...); err != nil {
		panic(err)
	}
}

// Graph is an object graph initialized from a package of modules.
type Graph struct {
	Package   *Package
	Instances map[reflect.Type]interface{}
}

// NewGraph creates an object graph from a package.
func NewGraph(p *Package) (*Graph, error) {
	g := &Graph{
		Package:   p,
		Instances: make(map[reflect.Type]interface{}, len(p.Constructors)),
	}
	g.initInstances()
	return g, nil
}

func (g *Graph) initInstances() {
	for _, c := range g.Package.Constructors {
		g.initInstance(c.Type)
	}
}

func (g *Graph) initInstance(typ reflect.Type) (interface{}, error) {
	instance, ok := g.Instances[typ]
	if ok {
		return instance, nil
	}

	c, ok := g.Package.Constructors[typ]
	if !ok {
		return nil, fmt.Errorf("di: No constructor for type %v", typ)
	}

	args := []interface{}{}
	for _, dep := range c.Deps {
		arg, err := g.initInstance(dep)
		if err != nil {
			return nil, err
		}

		args = append(args, arg)
	}

	instance = c.Func(args)
	g.Instances[typ] = instance
	return instance, nil
}

// MustGet returns an instance from this graph of the same type as i.
func (g *Graph) MustGet(i interface{}) interface{} {
	return g.MustGetByType(reflect.TypeOf(i))
}

// MustGetByType returns an instance from this graph of the given type.
func (g *Graph) MustGetByType(typ reflect.Type) interface{} {
	obj, ok := g.Instances[typ]
	if !ok {
		panic(fmt.Sprintf("di: No constructor for type %v", typ))
	}
	return obj
}

// Fill fills public fields in a struct with instances from this graph.
func (g *Graph) Fill(structPtr interface{}) {
	v := reflect.ValueOf(structPtr).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		ftype := field.Type()
		instance, ok := g.Instances[ftype]
		if !ok {
			continue
		}

		field.Set(reflect.ValueOf(instance))
	}
}

func getFuncName(fval reflect.Value) string {
	return runtime.FuncForPC(fval.Pointer()).Name()
}

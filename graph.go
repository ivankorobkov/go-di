package di

import (
	"fmt"
	"reflect"
)

// Object graph, well, actually, it's a tree.
type Graph map[reflect.Type]interface{}

func (g Graph) instantiate(p *Package, typ *typ) interface{} {
	if instance, ok := g[typ.typ]; ok {
		return instance
	}

	in := []reflect.Value{}
	for _, depTyp := range typ.deps {
		dep, ok := p.types[depTyp]
		if !ok {
			panic(fmt.Sprintf("modules: no constructor for type %v", depTyp))
		}

		arg := g.instantiate(p, dep)
		in = append(in, reflect.ValueOf(arg))
	}

	result := typ.constructor.Call(in)
	g[typ.typ] = result[0].Interface()
	return g[typ.typ]
}

func (g Graph) Get(i interface{}) interface{} {
	return g.GetByType(reflect.TypeOf(i))
}

func (g Graph) GetByType(typ reflect.Type) interface{} {
	obj, ok := g[typ]
	if !ok {
		panic(fmt.Sprintf("di: no instance for type %v", typ))
	}
	return obj
}

func (g Graph) Fill(structPtr interface{}) {
	v := reflect.ValueOf(structPtr).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		ftype := field.Type()
		instance, ok := g[ftype]
		if !ok {
			continue
		}

		field.Set(reflect.ValueOf(instance))
	}
}

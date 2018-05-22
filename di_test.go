package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewContext__should_create_and_initialize_context_and_instances(t *testing.T) {
	type Service struct {
		Int32  int32
		String string
		Bool   bool
	}
	newService := func(i int32, str string, b bool) *Service {
		return &Service{
			Int32:  i,
			String: str,
			Bool:   b,
		}
	}

	module0 := func(m *Module) {
		m.Add(func() int32 { return 1 })
		m.AddInstance("Hello, world")
	}
	module1 := func(m *Module) {
		boolean := true
		m.Import(module0)
		m.Add(newService)
		m.Dep(boolean)
	}
	module2 := func(m *Module) {
		m.AddInstance(true)
	}

	context, err := NewContext(module1, module2)
	if err != nil {
		t.Fatal(err)
	}

	var service *Service
	context.MustGet(&service)
	assert.Equal(t, int32(1), service.Int32)
	assert.Equal(t, "Hello, world", service.String)
	assert.Equal(t, true, service.Bool)
}

func testCyclicImport0(m *Module) { m.Import(testCyclicImport1) }
func testCyclicImport1(m *Module) { m.Import(testCyclicImport0) }

func Test_NewContext__should_return_error_on_cyclic_module_imports(t *testing.T) {
	_, err := NewContext(testCyclicImport0, testCyclicImport1)
	assert.Contains(t, err.Error(), "cyclic import")
}

func Test_NewContext__should_return_error_on_duplicate_providers(t *testing.T) {
	_, err := NewContext(func(m *Module) {
		m.AddInstance("hello")
	}, func(m *Module) {
		m.AddInstance("world")
	})

	assert.Contains(t, err.Error(), "duplicate provider")
}

func Test_NewContext__should_return_error_on_unresolved_provider_dependency(t *testing.T) {
	newService := func(address string) int32 { return 0 }
	_, err := NewContext(func(m *Module) { m.Add(newService) })
	assert.Contains(t, err.Error(), "unresolved provider dependency")
}

func Test_NewContext__should_return_provider_error_if_any(t *testing.T) {
	testErr := errors.New("Test error")
	_, err := NewContext(func(m *Module) {
		m.Add(func() (string, error) { return "", testErr })
	})

	assert.Equal(t, testErr, err)
}

func Test_NewContext__should_return_nil_error_from_provider(t *testing.T) {
	ctx, err := NewContext(func(m *Module) {
		m.Add(func() (string, error) { return "hello", nil })
	})
	if err != nil {
		t.Fatal(err)
	}

	str := ""
	ctx.MustGet(&str)

	assert.Equal(t, "hello", str)
}

func Test_Context_Get__should_get_instance_from_context(t *testing.T) {
	ctx, err := NewContext(func(m *Module) {
		m.AddInstance("hello")
	})
	if err != nil {
		t.Fatal(err)
	}

	s := ""
	ok := ctx.Get(&s)

	assert.True(t, ok)
	assert.Equal(t, "hello", s)
}

func Test_Context_Get__should_return_false_when_instance_is_not_found(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}

	s := ""
	ok := ctx.Get(&s)
	assert.False(t, ok)
}

func Test_Context_Inject__should_inject_dependencies_into_struct_fields(t *testing.T) {
	ctx, err := NewContext(func(m *Module) {
		m.AddInstance("hello")
		m.AddInstance(123)
		m.AddInstance(true)
	})
	if err != nil {
		t.Fatal(err)
	}

	s := struct {
		String string
		Int    int
		Bool   bool
	}{}
	ctx.Inject(&s)

	assert.Equal(t, "hello", s.String)
	assert.Equal(t, 123, s.Int)
	assert.Equal(t, true, s.Bool)
}

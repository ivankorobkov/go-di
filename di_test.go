package di

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewContext__should_create_and_initialize_context(t *testing.T) {
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

	module0 := func() *Module {
		m := &Module{}
		m.Add(func() int32 { return 1 })
		m.Add(func() string { return "Hello, world" })
		return m
	}
	module1 := func() *Module {
		m := &Module{}
		m.Import(module0)
		m.Depend(true)
		m.Add(newService)
		return m
	}
	module2 := func() *Module {
		m := &Module{}
		m.Add(func() bool { return true })
		return m
	}

	ctx, err := NewContext(module1, module2)
	if err != nil {
		t.Fatal(err)
	}

	s, err := ctx.Get(&Service{})
	if err != nil {
		t.Fatal(err)
	}
	service := s.(*Service)

	assert.Equal(t, int32(1), service.Int32)
	assert.Equal(t, "Hello, world", service.String)
	assert.Equal(t, true, service.Bool)
}

func testCyclicImport0() *Module {
	m := &Module{}
	m.Import(testCyclicImport1)
	return m
}

func testCyclicImport1() *Module {
	m := &Module{}
	m.Import(testCyclicImport0)
	return m
}

func TestNewContext__should_return_error_on_cyclic_imports(t *testing.T) {
	_, err := NewContext(testCyclicImport0, testCyclicImport1)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cyclic module imports")
	}
}

func TestNewContext__should_return_error_on_unresolved_constructor_dep(t *testing.T) {
	newService := func(address string) int32 { return 0 }
	_, err := NewContext(func() *Module {
		m := &Module{}
		m.Add(newService)
		return m
	})
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "unresolved dependency")
	}
}

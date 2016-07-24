package di

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew__should_create_and_initialize_object_graph(t *testing.T) {
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
		m.AddConstructor(func() int32 { return 1 })
		m.AddInstance("Hello, world")
	}
	module1 := func(m *Module) {
		m.Import(module0)
		m.AddConstructor(newService)
		m.MarkPackageDeps(struct{ Bool bool }{})
	}
	module2 := func(m *Module) {
		m.AddInstance(true)
	}

	g, err := New(module1, module2)
	if err != nil {
		t.Fatal(err)
	}

	service := g.MustGet(&Service{}).(*Service)
	assert.Equal(t, int32(1), service.Int32)
	assert.Equal(t, "Hello, world", service.String)
	assert.Equal(t, true, service.Bool)
}

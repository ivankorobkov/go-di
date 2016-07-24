package di

import (
	"strings"
	"testing"
)

func testCyclicImport0(m *Module) { m.Import(testCyclicImport1) }
func testCyclicImport1(m *Module) { m.Import(testCyclicImport0) }

func TestNewPackage__should_return_error_on_cyclic_imports(t *testing.T) {
	_, err := NewPackage(testCyclicImport0, testCyclicImport1)
	if err == nil || !strings.Contains(err.Error(), "Cyclic import in modules") {
		t.Fatal("Expected a cyclic import error")
	}
}

func TestNewPackage__should_return_error_on_unresolved_constructor_dep(t *testing.T) {
	newService := func(address string) int32 { return 0 }
	_, err := NewPackage(func(m *Module) { m.AddConstructor(newService) })
	if err == nil || !strings.Contains(err.Error(), "Unresolved dependency") {
		t.Fatal("Expected an unresolved dependency error")
	}
}

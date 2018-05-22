package di

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testAppService struct {
	started bool
	stopped bool
}

func (s *testAppService) Start() error {
	s.started = true
	return nil
}

func (s *testAppService) Stop() error {
	s.stopped = true
	return nil
}

func Test_App_Start__should_start_services(t *testing.T) {
	service := &testAppService{}
	app, err := NewApp(func(m *Module) { m.AddInstance(service) })
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	assert.True(t, service.started)
}

func Test_App_Stop__should_stop_services(t *testing.T) {
	service := &testAppService{}
	app, err := NewApp(func(m *Module) { m.AddInstance(service) })
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	assert.True(t, service.stopped)
}

package di

import (
	"context"
	"os"
	"os/signal"
	"time"
)

const (
	StartTimeout = 30 * time.Second
	StopTimeout  = 30 * time.Second
)

// Starter is a service which should be started on an application startup.
// The opposite is Closer.
type Starter interface {
	Start() error
}

// Closer is a service which should be closed on an application shutdown.
type Closer interface {
	Close() error
}

// App provides a start/stop lifecycle and a graceful shutdown.
// Usually, users should call app.Run() which starts the services in toplogical order
// from dependencies to dependants. Then blocks until a SIGINT/SIGKILL signal arrives,
// and stops the services in reverse order.
type App struct {
	Context      *Context
	StartTimeout time.Duration
	StopTimeout  time.Duration
	started      []interface{}
}

// NewApp creates a new application from modules.
func NewApp(modules ...ModuleFunc) (*App, error) {
	ctx, err := NewContext(modules...)
	if err != nil {
		return nil, err
	}

	app := &App{
		Context:      ctx,
		StartTimeout: StartTimeout,
		StopTimeout:  StopTimeout,
	}
	return app, nil
}

// Run starts the application, awaits a stop signal and then stops the application.
func (app *App) Run() error {
	// Start the app.
	startCtx := context.Background()
	if app.StartTimeout > 0 {
		var cancel context.CancelFunc
		startCtx, cancel = context.WithTimeout(startCtx, app.StartTimeout)
		defer cancel()
	}
	if err := app.Start(startCtx); err != nil {
		return err
	}

	// Await SIGINT/SIGKILL.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch

	// Stop the app.
	stopCtx := context.Background()
	if app.StopTimeout > 0 {
		var cancel context.CancelFunc
		stopCtx, cancel = context.WithTimeout(stopCtx, app.StopTimeout)
		defer cancel()
	}
	return app.Stop(stopCtx)
}

// Start starts the services which implement the Starter interface.
func (app *App) Start(ctx context.Context) error {
	// Find the services which implement the Starter interface.
	services := []Starter{}
	for _, instance := range app.Context.InstanceSlice {
		service, ok := instance.(Starter)
		if ok {
			services = append(services, service)
		}
	}

	// Start the services. On error stop already started in reverse order.
	for _, service := range services {
		err := withTimeout(ctx, service.Start)
		if err == nil {
			app.started = append(app.started, service)
			continue
		}
		if len(app.started) == 0 {
			return err
		}

		// Stop the started services in reverse order.
		for i := len(app.started) - 1; i >= 0; i-- {
			service := app.started[i]
			closer, ok := service.(Closer)
			if !ok {
				continue
			}
			withTimeout(ctx, closer.Close)
		}
		return err
	}

	return nil
}

// Stop stops the services which implement the Closer interface.
func (app *App) Stop(ctx context.Context) error {
	// Find the services which implement the Closer interface.
	services := []Closer{}
	for _, instance := range app.Context.InstanceSlice {
		service, ok := instance.(Closer)
		if ok {
			services = append(services, service)
		}
	}

	// Close the services.
	for _, service := range services {
		withTimeout(ctx, service.Close)
	}
	return nil
}

func withTimeout(ctx context.Context, fn func() error) error {
	ch := make(chan error, 1)
	go func() {
		ch <- fn()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}

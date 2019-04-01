package di

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"
)

const (
	StartTimeout = 30 * time.Second
	StopTimeout  = 30 * time.Second
)

// Starter is a service which should be started on an application startup.
type Starter interface {
	Start() error
}

// Stopper is a service which should be stopped on an application shutdown.
type Stopper interface {
	Stop() error
}

// Logger is an application logger.
type Logger interface {
	Println(v ...interface{})
}

// App provides a start/stop lifecycle and a graceful shutdown.
// Usually, users should call app.Run() which starts the services in toplogical order
// from dependencies to dependants. Then blocks until a SIGINT/SIGKILL signal arrives,
// and stops the services in reverse order.
type App struct {
	Context      *Context
	Logger       Logger
	StartTimeout time.Duration
	StopTimeout  time.Duration
}

// NewApp creates a new application from modules.
func NewApp(depOrMods ...interface{}) (*App, error) {
	ctx, err := NewContext(depOrMods...)
	if err != nil {
		return nil, err
	}

	app := &App{
		Context:      ctx,
		Logger:       log.New(os.Stderr, "", log.LstdFlags),
		StartTimeout: StartTimeout,
		StopTimeout:  StopTimeout,
	}
	return app, nil
}

// Run starts the application, awaits a stop signal and then stops the application.
func (app *App) Run() error {
	if err := app.runStart(); err != nil {
		app.runStop()
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch

	return app.runStop()
}

func (app *App) runStart() error {
	startCtx := context.Background()
	if app.StartTimeout > 0 {
		var cancel context.CancelFunc
		startCtx, cancel = context.WithTimeout(startCtx, app.StartTimeout)
		defer cancel()
	}
	return app.Start(startCtx)
}

func (app *App) runStop() error {
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
	app.log("Starting...")

	// Find the services which implement the Starter interface.
	services := []Starter{}
	for _, instance := range app.Context.InstanceSlice {
		service, ok := instance.(Starter)
		if ok {
			services = append(services, service)
		}
	}

	// Start the services.
	var err error
	for _, service := range services {
		if err = withTimeout(ctx, service.Start); err != nil {
			break
		}
	}

	switch {
	case ctx.Err() == err && err == context.DeadlineExceeded:
		app.log("Start timed out.")
		return err

	case err != nil:
		app.log("Failed to start:", err)
		return err
	}

	app.log("Started.")
	return nil
}

// Stop stops the services which implement the Stopper interface.
func (app *App) Stop(ctx context.Context) error {
	app.log("Stopping...")

	// Find the services which implement the Stopper interface.
	services := []Stopper{}
	for _, instance := range app.Context.InstanceSlice {
		service, ok := instance.(Stopper)
		if ok {
			services = append(services, service)
		}
	}

	// Close the services.
	var err error = nil
	for _, service := range services {
		if stopErr := withTimeout(ctx, service.Stop); stopErr != nil {
			if err == nil {
				err = stopErr
			}
		}
	}

	switch {
	case ctx.Err() == err && err == context.DeadlineExceeded:
		app.log("Stop timed out.")
		return nil
	case err != nil:
		app.log("Failed to stop cleanly:", err)
		return err
	}

	app.log("Stopped.")
	return nil
}

func (app *App) log(v ...interface{}) {
	if app.Logger == nil {
		return
	}
	app.Logger.Println(v...)
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

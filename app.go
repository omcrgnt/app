package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/omcrgnt/runner"
)

const DefaultShutdownTimeout = 5 * time.Second

// ShutdownTimeout is the grace period for [App.Serve] after the run context is cancelled.
type ShutdownTimeout time.Duration

func (ShutdownTimeout) Usage() string {
	return "Grace period for resource shutdown after signal (0 = default 5s)"
}

func (d ShutdownTimeout) Validate() error {
	if d < 0 {
		return fmt.Errorf("shutdown timeout must be >= 0")
	}
	return nil
}

// App is the root application resource; [runner.Runner] is injected after [sdi.Resolve].
type App struct {
	runner          *runner.Runner
	shutdownTimeout time.Duration
}

func (a *App) BuildConfig() (Materializer, error) {
	return &Spec{}, nil
}

// Spec is the app config; [Spec.Build] returns [*App].
type Spec struct {
	ShutdownTimeout ShutdownTimeout
}

func (s Spec) Build() (any, error) {
	d := time.Duration(s.ShutdownTimeout)
	if d == 0 {
		d = DefaultShutdownTimeout
	}
	return &App{shutdownTimeout: d}, nil
}

// DefaultApp returns the system App resource for app/use registration.
func DefaultApp() any {
	return &App{}
}

func (a *App) Deps() []any {
	return []any{
		(*runner.Runner)(nil),
	}
}

func (a *App) Inject(args []any) {
	for _, arg := range args {
		if r, ok := arg.(*runner.Runner); ok {
			a.runner = r
		}
	}
}

func (a *App) GracePeriod() time.Duration {
	if a.shutdownTimeout != 0 {
		return a.shutdownTimeout
	}
	return DefaultShutdownTimeout
}

// Serve runs until ctx is cancelled, then stops resources via injected [runner.Runner].
// If a starter fails, Serve stops the run context, shuts down, and returns the error.
func (a *App) Serve(ctx context.Context) error {
	if a.runner == nil {
		return fmt.Errorf("app: runner not injected")
	}

	ctx, stop := context.WithCancel(ctx)
	defer stop()

	var (
		runErrMu sync.Mutex
		runErr   error
	)

	go func() {
		if err := a.runner.Run(ctx); err != nil {
			runErrMu.Lock()
			runErr = err
			runErrMu.Unlock()
			slog.Error("runner failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.GracePeriod())
	defer cancel()
	if err := a.runner.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("app: shutdown: %w", err)
	}

	runErrMu.Lock()
	err := runErr
	runErrMu.Unlock()
	if err != nil {
		return fmt.Errorf("app: runner: %w", err)
	}
	return nil
}

package app

import (
	"context"
	"fmt"
	"log/slog"
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
func (a *App) Serve(ctx context.Context) error {
	if a.runner == nil {
		return fmt.Errorf("app: runner not injected")
	}

	go func() {
		if err := a.runner.Run(ctx); err != nil {
			slog.Error("runner failed", "err", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.GracePeriod())
	defer cancel()
	if err := a.runner.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("app: shutdown: %w", err)
	}
	return nil
}

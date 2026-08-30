package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/omcrgnt/app"
	"github.com/omcrgnt/runner"
)

// slowThenCloseableStarter's Start returns well after a short ctx deadline
// would already have fired — reproducing the window where Stop's r.started
// scan could otherwise run before setStarted for this resource.
type slowThenCloseableStarter struct {
	startDelay time.Duration
	closed     bool
}

func (s *slowThenCloseableStarter) Start(context.Context) (func(context.Context) error, error) {
	time.Sleep(s.startDelay)
	return s.cleanup, nil
}

func (s *slowThenCloseableStarter) cleanup(context.Context) error {
	s.closed = true
	return nil
}

func TestServe_joinsRunBeforeStop(t *testing.T) {
	starter := &slowThenCloseableStarter{startDelay: 150 * time.Millisecond}

	r := &runner.Runner{}
	r.Inject([]any{
		[]runner.Starter{starter},
		nil,
	})
	a := &app.App{}
	a.Inject([]any{r})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if err := a.Serve(ctx); err != nil {
		t.Fatal(err)
	}

	if !starter.closed {
		t.Fatal("Serve returned Stop before the in-flight Starter's Close ran — Stop raced ahead of Run")
	}
}

type startFailer struct{}

func (startFailer) Start(context.Context) (func(context.Context) error, error) {
	return nil, errors.New("boom-start")
}

type closeFailer struct{}

func (closeFailer) Start(context.Context) (func(context.Context) error, error) {
	return func(context.Context) error { return errors.New("boom-close") }, nil
}

// TestServe_runFailureIncludesRollbackCleanupError: when a sibling Starter
// fails in the same wave, runner.Runner.Run rolls back cf's already-started
// cleanup itself — synchronously, before Run even returns — so
// "boom-close" ends up joined into wrappedRunErr, not wrappedStopErr. By
// the time app.go's own a.runner.Stop(shutdownCtx) call runs afterward,
// Run's rollback has already consumed and cleared cf's cleanup, so Stop
// finds nothing left to close and returns nil. This test only exercises
// that joined-Run-error path; see TestServe_stopCleanupError for the
// wrappedStopErr path (a real Stop-phase failure after a successful Run).
func TestServe_runFailureIncludesRollbackCleanupError(t *testing.T) {
	cf := closeFailer{}

	r := &runner.Runner{}
	r.Inject([]any{
		[]runner.Starter{startFailer{}, cf},
		nil,
	})
	a := &app.App{}
	a.Inject([]any{r})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := a.Serve(ctx)
	if err == nil {
		t.Fatal("expected a combined error")
	}
	if !strings.Contains(err.Error(), "boom-start") {
		t.Errorf("missing runner/start error in: %v", err)
	}
	if !strings.Contains(err.Error(), "boom-close") {
		t.Errorf("missing rollback cleanup error in: %v", err)
	}
	if !strings.Contains(err.Error(), "app: runner:") {
		t.Errorf("expected both errors wrapped under app: runner:, got: %v", err)
	}
	if strings.Contains(err.Error(), "app: shutdown:") {
		t.Errorf("boom-close was already consumed by Run's own rollback — must not also appear under app: shutdown:, got: %v", err)
	}
}

// startOKCleanupFailer starts successfully so its cleanup survives into the
// real a.runner.Stop(shutdownCtx) call in app.go — unlike closeFailer above,
// whose cleanup runner.Runner.Run's own rollback would otherwise consume
// first if paired with a failing sibling.
type startOKCleanupFailer struct{}

func (startOKCleanupFailer) Start(context.Context) (func(context.Context) error, error) {
	return func(context.Context) error { return errors.New("boom-close") }, nil
}

// TestServe_stopCleanupError covers wrappedStopErr in app.go's Serve: a
// Starter that starts cleanly, runs until ctx is cancelled, and whose
// cleanup then fails during the real a.runner.Stop(shutdownCtx) call — the
// one branch TestServe_runFailureIncludesRollbackCleanupError's rename made
// clear was not actually exercised (that test's "boom-close" is consumed
// by Run's own rollback before Stop ever runs).
func TestServe_stopCleanupError(t *testing.T) {
	r := &runner.Runner{}
	r.Inject([]any{
		[]runner.Starter{startOKCleanupFailer{}},
		nil,
	})
	a := &app.App{}
	a.Inject([]any{r})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := a.Serve(ctx)
	if err == nil {
		t.Fatal("expected a shutdown error")
	}
	if !strings.Contains(err.Error(), "app: shutdown:") {
		t.Errorf("missing app: shutdown: wrapper in: %v", err)
	}
	if !strings.Contains(err.Error(), "boom-close") {
		t.Errorf("missing close error in: %v", err)
	}
}

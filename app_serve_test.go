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

func (s *slowThenCloseableStarter) Start(context.Context) error {
	time.Sleep(s.startDelay)
	return nil
}

func (s *slowThenCloseableStarter) Close(context.Context) error {
	s.closed = true
	return nil
}

func TestServe_joinsRunBeforeStop(t *testing.T) {
	starter := &slowThenCloseableStarter{startDelay: 150 * time.Millisecond}

	r := &runner.Runner{}
	r.Inject([]any{
		[]runner.Starter{starter},
		[]runner.Closer{starter},
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

func (startFailer) Start(context.Context) error { return errors.New("boom-start") }

type closeFailer struct{}

func (closeFailer) Start(context.Context) error { return nil }
func (closeFailer) Close(context.Context) error { return errors.New("boom-close") }

func TestServe_combinedRunAndStopErrors(t *testing.T) {
	cf := closeFailer{}

	r := &runner.Runner{}
	r.Inject([]any{
		[]runner.Starter{startFailer{}, cf},
		[]runner.Closer{cf},
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
		t.Errorf("missing shutdown/close error in: %v", err)
	}
}

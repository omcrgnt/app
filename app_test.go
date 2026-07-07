package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/omcrgnt/app"
	"github.com/omcrgnt/runner"
)

type failingStarter struct{}

func (failingStarter) Start(context.Context) error {
	return errors.New("bind: address already in use")
}

func TestServe_runnerStartFailureReturnsError(t *testing.T) {
	r := &runner.Runner{}
	r.Inject([]any{
		[]runner.Starter{failingStarter{}},
		nil,
	})
	a := &app.App{}
	a.Inject([]any{r})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := a.Serve(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

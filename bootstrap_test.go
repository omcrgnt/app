package app_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/omcrgnt/app"
	"github.com/omcrgnt/res"
	"github.com/omcrgnt/runner"
)

// bootstrapResources is a minimal AppResources shape for pipeline tests.
type bootstrapResources struct {
	App    *app.App `ecfg:"APP"`
	Runner *runner.Runner
}

func TestBootstrap_injectsRunnerIntoApp(t *testing.T) {
	t.Setenv("FIX_APP_SHUTDOWN_TIMEOUT", "5s")

	var r bootstrapResources
	reg, err := app.Bootstrap(&r, app.Pipeline{
		Registry:  res.New(),
		EnvPrefix: "FIX",
	})
	if err != nil {
		t.Fatal(err)
	}

	appAny, err := reg.GetOneByType(reflect.TypeOf((*app.App)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	a := appAny.(*app.App)
	if a.GracePeriod() != 5*time.Second {
		t.Fatalf("shutdown: got %v", a.GracePeriod())
	}

	runAny, err := reg.GetOneByType(reflect.TypeOf((*runner.Runner)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	if runAny.(*runner.Runner) == nil {
		t.Fatal("expected runner")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Serve(ctx, reg); err != nil {
		t.Fatal(err)
	}
}

package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/omcrgnt/app"
	"github.com/omcrgnt/ecfg"
	"github.com/omcrgnt/res/unique"
	"github.com/omcrgnt/runner"
)

type bootstrapResources struct {
	App *app.App `ecfg:"APP"`
}

type bootstrapResourcesCfgTag struct {
	App *app.App `cfg:"APP"`
}

type bootstrapResourcesNoApp struct{}

// fakeGate satisfies runner.Runner's gateOpener dependency (Open(), no
// args) — every registry using Runner must supply one, since runner.Gate
// itself can only be constructed from inside its own package.
type fakeGate struct{}

func (*fakeGate) Open() {}

func TestBootstrap_defaultAppFromUse(t *testing.T) {
	reg := unique.New()
	reg.MustAddReplaceable(app.DefaultApp())
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})

	a, err := app.Bootstrap(&bootstrapResourcesNoApp{}, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	})
	if err != nil {
		t.Fatal(err)
	}

	if a.GracePeriod() != 5*time.Second {
		t.Fatalf("shutdown: got %v", a.GracePeriod())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Serve(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestBootstrap_injectsRunnerIntoApp(t *testing.T) {
	t.Setenv("FIX_APP_SHUTDOWN_TIMEOUT", "5s")

	reg := unique.New()
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})

	var r bootstrapResources
	a, err := app.Bootstrap(&r, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	})
	if err != nil {
		t.Fatal(err)
	}

	if a.GracePeriod() != 5*time.Second {
		t.Fatalf("shutdown: got %v", a.GracePeriod())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Serve(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestBootstrap_customCatalogTagKey(t *testing.T) {
	ecfg.SetTagKey("cfg")
	t.Cleanup(ecfg.ResetForTest)

	t.Setenv("FIX_APP_SHUTDOWN_TIMEOUT", "9s")

	reg := unique.New()
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})

	var r bootstrapResourcesCfgTag
	a, err := app.Bootstrap(&r, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	})
	if err != nil {
		t.Fatal(err)
	}

	if a.GracePeriod() != 9*time.Second {
		t.Fatalf("shutdown: got %v", a.GracePeriod())
	}
}

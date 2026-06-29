package app_test

import (
	"testing"
	"time"

	"github.com/omcrgnt/app"
)

func TestDefaultApp_gracePeriod(t *testing.T) {
	got := app.DefaultApp().(*app.App).GracePeriod()
	if got != 5*time.Second {
		t.Fatalf("shutdown: got %v", got)
	}
}

func TestSpec_Build_defaultShutdown(t *testing.T) {
	appAny, err := app.Spec{}.Build()
	if err != nil {
		t.Fatal(err)
	}
	got := appAny.(*app.App).GracePeriod()
	if got != 5*time.Second {
		t.Fatalf("shutdown: got %v", got)
	}
}

func TestSpec_Build_customShutdown(t *testing.T) {
	appAny, err := app.Spec{ShutdownTimeout: app.ShutdownTimeout(9 * time.Second)}.Build()
	if err != nil {
		t.Fatal(err)
	}
	got := appAny.(*app.App).GracePeriod()
	if got != 9*time.Second {
		t.Fatalf("shutdown: got %v", got)
	}
}

func TestShutdownTimeout_Validate(t *testing.T) {
	if err := app.ShutdownTimeout(-1).Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

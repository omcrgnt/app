package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/omcrgnt/app"
	"github.com/omcrgnt/res/unique"
	"github.com/omcrgnt/runner"
)

// standByProvider mimics client-http.Client: its own Inject computes state
// (ready) that a consumer registered earlier in the registry cannot see
// from inside its own Inject, only from a later phase.
type standByProvider struct {
	ready bool
}

func (*standByProvider) Deps() []any { return nil }

func (p *standByProvider) Inject([]any) {
	p.ready = true
}

// standByConsumer mimics client-ollama.Client: it stores the dependency
// pointer during Inject (registered, and thus injected, before the
// provider), then reads the provider's Inject-computed state from StandBy
// instead — proving StandBy runs after every resource's Inject has
// completed, regardless of registration order.
type standByConsumer struct {
	provider *standByProvider
	sawReady bool
}

func (c *standByConsumer) Deps() []any {
	return []any{(*standByProvider)(nil)}
}

func (c *standByConsumer) Inject(args []any) {
	for _, a := range args {
		if p, ok := a.(*standByProvider); ok {
			c.provider = p
		}
	}
}

func (c *standByConsumer) StandBy() error {
	c.sawReady = c.provider.ready
	return nil
}

func TestBootstrap_standByRunsAfterAllInject(t *testing.T) {
	reg := unique.New()
	reg.MustAddReplaceable(app.DefaultApp())
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})

	consumer := &standByConsumer{}
	provider := &standByProvider{}
	// Registration order matters: consumer before provider means
	// consumer.Inject runs while provider.ready is still false.
	reg.MustAddFixed(consumer)
	reg.MustAddFixed(provider)

	if _, err := app.Bootstrap(&bootstrapResourcesNoApp{}, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	}); err != nil {
		t.Fatal(err)
	}

	if consumer.provider == nil {
		t.Fatal("consumer.provider not injected")
	}
	if !consumer.sawReady {
		t.Fatal("StandBy observed provider.ready == false; want true (StandBy must run after every resource's Inject)")
	}
}

// standByOrderN are distinct concrete types (MustAddFixed allows only one
// resource per exact type) that each append a label to a shared slice —
// used to pin StandBy's documented "registration order" guarantee across
// multiple resolvers, not just the single-resolver case above.
type standByOrder1 struct{ order *[]string }

func (s *standByOrder1) StandBy() error { *s.order = append(*s.order, "first"); return nil }

type standByOrder2 struct{ order *[]string }

func (s *standByOrder2) StandBy() error { *s.order = append(*s.order, "second"); return nil }

type standByOrder3 struct{ order *[]string }

func (s *standByOrder3) StandBy() error { *s.order = append(*s.order, "third"); return nil }

func TestBootstrap_standByRunsInRegistrationOrder(t *testing.T) {
	reg := unique.New()
	reg.MustAddReplaceable(app.DefaultApp())
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})

	var order []string
	reg.MustAddFixed(&standByOrder1{order: &order})
	reg.MustAddFixed(&standByOrder2{order: &order})
	reg.MustAddFixed(&standByOrder3{order: &order})

	if _, err := app.Bootstrap(&bootstrapResourcesNoApp{}, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	}); err != nil {
		t.Fatal(err)
	}

	want := []string{"first", "second", "third"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

type standByFailer struct{}

func (*standByFailer) StandBy() error { return errStandBy }

var errStandBy = errors.New("standby: boom")

func TestBootstrap_standByErrorAbortsBootstrap(t *testing.T) {
	reg := unique.New()
	reg.MustAddReplaceable(app.DefaultApp())
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})
	reg.MustAddFixed(&standByFailer{})

	_, err := app.Bootstrap(&bootstrapResourcesNoApp{}, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

// standByCloserA/B mimic client-grpc.Client: StandBy succeeds (e.g. dials a
// connection), Close releases it. Distinct types (MustAddFixed) — A is
// meant to register before the failing resource, B after.
type standByCloserA struct {
	closeErr error
	closed   bool
}

func (s *standByCloserA) StandBy() error { return nil }
func (s *standByCloserA) Close(context.Context) error {
	s.closed = true
	return s.closeErr
}

type standByCloserB struct{ closed bool }

func (s *standByCloserB) StandBy() error { return nil }
func (s *standByCloserB) Close(context.Context) error {
	s.closed = true
	return nil
}

func TestBootstrap_standByFailureClosesEarlierSucceeded(t *testing.T) {
	reg := unique.New()
	reg.MustAddReplaceable(app.DefaultApp())
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})

	a := &standByCloserA{}
	b := &standByCloserB{}
	reg.MustAddFixed(a) // registered before the failure: must be closed
	reg.MustAddFixed(&standByFailer{})
	reg.MustAddFixed(b) // registered after the failure: StandBy never ran, must not be closed

	_, err := app.Bootstrap(&bootstrapResourcesNoApp{}, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	})
	if !errors.Is(err, errStandBy) {
		t.Fatalf("err = %v, want to wrap %v", err, errStandBy)
	}
	if !a.closed {
		t.Fatal("earlier-succeeded resource was not closed after a later StandBy failed")
	}
	if b.closed {
		t.Fatal("resource registered after the failure was closed, but its StandBy never ran")
	}
}

func TestBootstrap_standByFailureJoinsCloseError(t *testing.T) {
	reg := unique.New()
	reg.MustAddReplaceable(app.DefaultApp())
	reg.MustAddFixed(&runner.Runner{})
	reg.MustAddFixed(&fakeGate{})

	closeErr := errors.New("close: boom")
	reg.MustAddFixed(&standByCloserA{closeErr: closeErr})
	reg.MustAddFixed(&standByFailer{})

	_, err := app.Bootstrap(&bootstrapResourcesNoApp{}, app.Pipeline{
		Registry:  reg,
		EnvPrefix: "FIX",
	})
	if !errors.Is(err, errStandBy) {
		t.Fatalf("err = %v, want to wrap %v", err, errStandBy)
	}
	if !errors.Is(err, closeErr) {
		t.Fatalf("err = %v, want to also wrap %v", err, closeErr)
	}
}

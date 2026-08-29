package app

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/omcrgnt/res/unique"
)

type gSpec[T any] struct {
	N int
}

func (s gSpec[T]) Build() (any, error) {
	return &gServer[T]{n: s.N}, nil
}

type gServer[T any] struct {
	mu sync.Mutex
	n  int
}

func (*gServer[T]) BuildConfig() (Materializer, error) {
	return &gSpec[T]{}, nil
}

type gRepo[T any] struct {
	tag string
}

func (*gRepo[T]) NewResource() (any, error) {
	return &gRepo[T]{tag: "new"}, nil
}

type bothResource[T any] struct{}

func (*bothResource[T]) BuildConfig() (Materializer, error) {
	return &gSpec[T]{}, nil
}

func (*bothResource[T]) NewResource() (any, error) {
	return &gRepo[T]{}, nil
}

type valueSlot struct{}

func (*valueSlot) BuildConfig() (Materializer, error) {
	return &valueSpec{}, nil
}

type valueSpec struct{}

func (valueSpec) Build() (any, error) {
	return "built", nil
}

func TestFill_nilGenericConfigurable(t *testing.T) {
	type catalog struct {
		Slot *gServer[string] `ecfg:"SLOT"`
	}
	var c catalog

	specs := unique.New()
	resources := unique.New()

	if err := fill(&c, specs, resources); err != nil {
		t.Fatal(err)
	}

	entries := specs.GetByType(reflect.TypeFor[*gSpec[string]]())
	if len(entries) != 1 {
		t.Fatalf("spec entries = %d, want 1", len(entries))
	}
}

func TestFill_nilGenericFactory(t *testing.T) {
	type catalog struct {
		Repo *gRepo[int]
	}
	var c catalog

	specs := unique.New()
	resources := unique.New()

	if err := fill(&c, specs, resources); err != nil {
		t.Fatal(err)
	}

	got, err := resources.GetOneByType(reflect.TypeFor[*gRepo[int]]())
	if err != nil {
		t.Fatal(err)
	}
	repo := got.(*gRepo[int])
	if repo.tag != "new" {
		t.Fatalf("tag = %q, want new", repo.tag)
	}
}

func TestFill_nilAppConfigurable(t *testing.T) {
	type catalog struct {
		App *App `ecfg:"APP"`
	}
	var c catalog

	specs := unique.New()
	resources := unique.New()

	if err := fill(&c, specs, resources); err != nil {
		t.Fatal(err)
	}

	if _, err := specs.GetOneByType(reflect.TypeFor[*Spec]()); err != nil {
		t.Fatal(err)
	}
}

func TestFill_valueStructPointerMethods(t *testing.T) {
	type catalog struct {
		Slot valueSlot `ecfg:"SLOT"`
	}
	var c catalog

	specs := unique.New()
	resources := unique.New()

	if err := fill(&c, specs, resources); err != nil {
		t.Fatal(err)
	}

	if _, err := specs.GetOneByType(reflect.TypeFor[*valueSpec]()); err != nil {
		t.Fatal(err)
	}
}

// typedNilSpecSlot's BuildConfig returns a typed nil pointer as the spec —
// the buggy-implementation case isNilMaterializer exists to catch (a bare
// spec == nil comparison misses this: the interface value's type
// descriptor is non-nil even though the underlying pointer is).
type typedNilSpecSlot struct{}

func (*typedNilSpecSlot) BuildConfig() (Materializer, error) {
	var spec *gSpec[string]
	return spec, nil
}

// nilSpecSlot's BuildConfig returns a literal nil interface — the simpler
// case isNilMaterializer's spec == nil check alone already caught, kept
// covered separately from the typed-nil case above.
type nilSpecSlot struct{}

func (*nilSpecSlot) BuildConfig() (Materializer, error) {
	return nil, nil
}

func TestFill_literalNilConfigSpec(t *testing.T) {
	type catalog struct {
		Slot *nilSpecSlot `ecfg:"SLOT"`
	}
	var c catalog

	err := fill(&c, unique.New(), unique.New())
	if err == nil {
		t.Fatal("expected error for a nil config spec")
	}
	if !strings.Contains(err.Error(), "nil config spec") {
		t.Fatalf("error = %v, want mention of nil config spec", err)
	}
}

func TestFill_typedNilConfigSpec(t *testing.T) {
	type catalog struct {
		Slot *typedNilSpecSlot `ecfg:"SLOT"`
	}
	var c catalog

	err := fill(&c, unique.New(), unique.New())
	if err == nil {
		t.Fatal("expected error for a typed-nil config spec")
	}
	if !strings.Contains(err.Error(), "nil config spec") {
		t.Fatalf("error = %v, want mention of nil config spec", err)
	}
}

func TestFill_bothInterfaces(t *testing.T) {
	type catalog struct {
		Both *bothResource[string]
	}
	var c catalog

	err := fill(&c, unique.New(), unique.New())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "implements both") {
		t.Fatalf("error = %v", err)
	}
}

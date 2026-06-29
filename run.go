package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/omcrgnt/ecfg"
	"github.com/omcrgnt/res"
	"github.com/omcrgnt/res/unique"
	"github.com/omcrgnt/sdi"
)

// Pipeline configures fill → LoadEnv → materialize → merge → Transform → Resolve.
// Callers (typically main) must set every field explicitly; there are no package defaults.
type Pipeline struct {
	Registry   *unique.Registry
	EnvPrefix  string
	Transforms []res.TransformFunc
}

func (p Pipeline) validate() error {
	if p.Registry == nil {
		return fmt.Errorf("app: nil registry")
	}
	if p.EnvPrefix == "" {
		return fmt.Errorf("app: empty env prefix")
	}
	return nil
}

// Bootstrap runs fill → LoadEnv → materialize → merge → Transform → Resolve.
func Bootstrap(appResources any, p Pipeline) (*App, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}

	registrySpecs := unique.New()
	registryResources := unique.New()

	if err := fill(appResources, registrySpecs, registryResources); err != nil {
		return nil, err
	}

	ecfg.SetPrefix(p.EnvPrefix)
	if err := ecfg.LoadEnv(registrySpecs); err != nil {
		return nil, err
	}

	if err := materialize(registrySpecs); err != nil {
		return nil, err
	}

	if err := unique.Merge(registryResources, registrySpecs); err != nil {
		return nil, err
	}

	if err := unique.Merge(p.Registry, registryResources); err != nil {
		return nil, err
	}

	if len(p.Transforms) > 0 {
		if err := p.Registry.Transform(p.Transforms...); err != nil {
			return nil, err
		}
	}

	if err := sdi.Resolve(p.Registry); err != nil {
		return nil, err
	}

	return appFromReg(p.Registry)
}

func appFromReg(reg *unique.Registry) (*App, error) {
	appAny, err := reg.GetOneByType(reflect.TypeOf((*App)(nil)))
	if err != nil {
		return nil, fmt.Errorf("app: not found: %w", err)
	}
	return appAny.(*App), nil
}

// Run bootstraps appResources and blocks until shutdown via resolved [App].
func Run(appResources any, p Pipeline) error {
	a, err := Bootstrap(appResources, p)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return a.Serve(ctx)
}

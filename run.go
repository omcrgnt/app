package app

import (
	"context"
	"errors"
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

	standByType := reflect.TypeOf((*StandBy)(nil)).Elem()
	var succeeded []any
	for _, e := range p.Registry.GetByInterface(standByType) {
		v := e.Value()
		if err := v.(StandBy).StandBy(); err != nil {
			closeErr := closeStandBySucceeded(succeeded)
			return nil, errors.Join(fmt.Errorf("app: standby: %w", err), closeErr)
		}
		succeeded = append(succeeded, v)
	}

	return appFromReg(p.Registry)
}

// closer is a runner.Closer-shaped duck type (no runner import) — used only
// to close resources whose StandBy already succeeded when a later
// resource's StandBy fails and aborts Bootstrap. Without this, that
// failure path returns before runner.Run (and thus runner.Stop) is ever
// reached, so nothing else would close them.
type closer interface {
	Close(context.Context) error
}

func closeStandBySucceeded(values []any) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	var errs []error
	for _, v := range values {
		if c, ok := v.(closer); ok {
			if err := c.Close(ctx); err != nil {
				errs = append(errs, fmt.Errorf("app: standby cleanup: close %T: %w", v, err))
			}
		}
	}
	return errors.Join(errs...)
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

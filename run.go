package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/omcrgnt/builder"
	"github.com/omcrgnt/ecfg"
	"github.com/omcrgnt/res"
	"github.com/omcrgnt/sdi"
)

// Pipeline configures Seed → Apply → Build → Transform → Resolve.
// Callers (typically main) must set every field explicitly; there are no package defaults.
type Pipeline struct {
	Registry   res.Registry
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

// Bootstrap runs Seed → Apply → Build → Transform → Resolve.
func Bootstrap(appResources any, p Pipeline) (res.Registry, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}

	reg := p.Registry
	if err := builder.Seed(reg, appResources); err != nil {
		return nil, err
	}
	if err := ecfg.Apply(reg, appResources, ecfg.WithPrefix(p.EnvPrefix)); err != nil {
		return nil, err
	}
	if err := builder.Build(reg); err != nil {
		return nil, err
	}
	if len(p.Transforms) > 0 {
		if err := reg.Transform(p.Transforms...); err != nil {
			return nil, err
		}
	}
	if err := sdi.Resolve(reg); err != nil {
		return nil, err
	}

	return reg, nil
}

func appFromReg(reg res.Registry) (*App, error) {
	appAny, err := reg.GetOneByType(reflect.TypeOf((*App)(nil)))
	if err != nil {
		return nil, fmt.Errorf("app: not found: %w", err)
	}
	return appAny.(*App), nil
}

// Run bootstraps appResources and blocks until shutdown via resolved [App].
func Run(appResources any, p Pipeline) error {
	reg, err := Bootstrap(appResources, p)
	if err != nil {
		return err
	}

	a, err := appFromReg(reg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return a.Serve(ctx, reg)
}

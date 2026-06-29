package app

import (
	"fmt"

	"github.com/omcrgnt/res"
	"github.com/omcrgnt/res/unique"
)

func materialize(reg *unique.Registry) error {
	var specs []Materializer
	reg.WalkEntries(func(e res.Entry) bool {
		if m, ok := e.Value().(Materializer); ok {
			specs = append(specs, m)
		}
		return true
	})

	for _, spec := range specs {
		built, err := spec.Build()
		if err != nil {
			return fmt.Errorf("app: %T: %w", spec, err)
		}
		if err := reg.Add(built); err != nil {
			return fmt.Errorf("app: %T: %w", spec, err)
		}
		if err := reg.Remove(spec); err != nil {
			return fmt.Errorf("app: %T: remove spec: %w", spec, err)
		}
	}
	return nil
}

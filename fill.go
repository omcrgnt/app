package app

import (
	"fmt"
	"reflect"

	"github.com/omcrgnt/ecfg"
	"github.com/omcrgnt/res/unique"
)

// fill walks AppResources catalog fields and registers specs (Configurable) or
// resources (ResourceFactory) into staging registries before LoadEnv/materialize.
//
// Catalog *Resource fields are often nil: they declare type + ecfg segment only.
// BuildConfig/NewResource run on a zero *T via reflect (not on nil), including
// generic pointer receivers that panic when invoked on nil.
func fill(appResources any, registrySpecs, registryResources *unique.Registry) error {
	rv, err := structValue(appResources)
	if err != nil {
		return err
	}

	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		fieldVal := rv.Field(i)
		if !fieldVal.CanInterface() {
			continue
		}

		cfg, isConfigurable := catalogCallable[Configurable](fieldVal)
		factory, isFactory := catalogCallable[ResourceFactory](fieldVal)
		switch {
		case isConfigurable && isFactory:
			return fmt.Errorf("app: %s: implements both ResourceFactory and Configurable", sf.Name)
		case isConfigurable:
			spec, err := cfg.BuildConfig()
			if err != nil {
				return fmt.Errorf("app: %s: BuildConfig: %w", sf.Name, err)
			}
			if spec == nil {
				return fmt.Errorf("app: %s: nil config spec", sf.Name)
			}
			seg, err := ecfg.CatalogSegment(sf)
			if err != nil {
				return fmt.Errorf("app: %s: %w", sf.Name, err)
			}
			if err := registrySpecs.AddWithCustomTag(spec, ecfg.TagKey(), seg); err != nil {
				return fmt.Errorf("app: %s: %w", sf.Name, err)
			}
		case isFactory:
			resource, err := factory.NewResource()
			if err != nil {
				return fmt.Errorf("app: %s: NewResource: %w", sf.Name, err)
			}
			if err := registryResources.Add(resource); err != nil {
				return fmt.Errorf("app: %s: %w", sf.Name, err)
			}
		default:
			return fmt.Errorf("app: %s: must implement ResourceFactory or Configurable", sf.Name)
		}
	}
	return nil
}

func catalogCallable[T any](v reflect.Value) (T, bool) {
	var zero T
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			c, ok := reflect.New(v.Type().Elem()).Interface().(T)
			return c, ok
		}
		c, ok := v.Interface().(T)
		return c, ok
	default:
		if c, ok := v.Interface().(T); ok {
			return c, ok
		}
		if v.CanAddr() {
			c, ok := v.Addr().Interface().(T)
			return c, ok
		}
		return zero, false
	}
}

func structValue(v any) (reflect.Value, error) {
	if v == nil {
		return reflect.Value{}, fmt.Errorf("app: nil app resources")
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return reflect.Value{}, fmt.Errorf("app: nil app resources")
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("app: want struct, got %s", rv.Kind())
	}

	return rv, nil
}

package app

import (
	"fmt"
	"reflect"

	"github.com/omcrgnt/ecfg"
	"github.com/omcrgnt/res/unique"
)

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

		field := fieldVal.Interface()
		_, isFactory := field.(ResourceFactory)
		_, isConfigurable := field.(Configurable)
		switch {
		case isFactory && isConfigurable:
			return fmt.Errorf("app: %s: implements both ResourceFactory and Configurable", sf.Name)
		case isFactory:
			resource, err := field.(ResourceFactory).NewResource()
			if err != nil {
				return fmt.Errorf("app: %s: NewResource: %w", sf.Name, err)
			}
			if err := registryResources.Add(resource); err != nil {
				return fmt.Errorf("app: %s: %w", sf.Name, err)
			}
		case isConfigurable:
			spec, err := field.(Configurable).BuildConfig()
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
		default:
			return fmt.Errorf("app: %s: must implement ResourceFactory or Configurable", sf.Name)
		}
	}
	return nil
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

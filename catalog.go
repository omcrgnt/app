package app

// Configurable is a catalog resource slot (*Resource or value) that registers a spec for env + materialize.
// Nil *Resource is valid: fill calls BuildConfig on a zero instance of the field type.
type Configurable interface {
	BuildConfig() (Materializer, error)
}

// Materializer is a config spec in the registry before materialize.
type Materializer interface {
	Build() (any, error)
}

// ResourceFactory is a catalog resource slot whose runtime instance is created at fill time.
// Nil *Resource is valid: fill calls NewResource on a zero instance of the field type.
type ResourceFactory interface {
	NewResource() (any, error)
}

package app

// Configurable is a catalog wire type that registers a config spec for env + materialize.
type Configurable interface {
	BuildConfig() (Materializer, error)
}

// Materializer is a config spec in the registry before materialize.
type Materializer interface {
	Build() (any, error)
}

// ResourceFactory is a catalog wire type whose resource is created at fill time.
type ResourceFactory interface {
	NewResource() (any, error)
}

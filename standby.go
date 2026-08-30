package app

// StandBy is a sequential post-Resolve initialization hook: it runs once,
// in registration order, after every resource's Inject has completed and
// before runner.Run starts the concurrent Start phase. Use it only for
// zero-I/O finishing touches that read another resource's Inject-computed
// state (e.g. building an SDK client wrapper around a *clienthttp.Client
// whose own Inject ran later in the same sdi.Resolve pass). Anything that
// performs real I/O or may run long belongs in runner.Starter instead.
//
// If StandBy allocates something that needs undoing should a later
// resource's StandBy fail and abort Bootstrap, also implement
// [StandByCleaner] — most StandBy resources (e.g. client-ollama, which only
// wraps an already-owned *http.Client) allocate nothing and don't need to.
type StandBy interface {
	StandBy() error
}

// StandByCleaner is [StandBy] plus an explicit way to undo whatever StandBy
// set up. If a later resource's StandBy fails, Bootstrap calls CleanUp for
// every already-succeeded resource that implements this, in registration
// order, and joins any of those errors with the original failure.
//
// Deliberately a separate method from runner.Closer's Close(ctx) error:
// that one pairs with Start and the normal shutdown path (runner.Runner.Stop),
// which StandBy never participates in — a type may implement both, each
// undoing what its own Start/StandBy set up, without one path having to
// guess whether the other's Close is safe to call in its place.
type StandByCleaner interface {
	StandBy
	CleanUp() error
}

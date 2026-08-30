package app

// StandBy is a sequential post-Resolve initialization hook: it runs once,
// in registration order, after every resource's Inject has completed and
// before runner.Run starts the concurrent Start phase. Use it only for
// zero-I/O finishing touches that read another resource's Inject-computed
// state (e.g. building an SDK client wrapper around a *clienthttp.Client
// whose own Inject ran later in the same sdi.Resolve pass). Anything that
// performs real I/O or may run long belongs in runner.Starter instead.
//
// If StandBy returns an error, Bootstrap closes every already-succeeded
// StandBy resource that also implements Close(context.Context) error — see
// the package doc for the exact order and error-joining behavior.
type StandBy interface {
	StandBy() error
}

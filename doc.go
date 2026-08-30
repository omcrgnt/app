/*
Package app is the org bootstrap harness: catalog fill, env load, materialize, merge, sdi wire, and process [Run].

Import [github.com/omcrgnt/app/use] in main to register [DefaultApp] and [runner.Runner] on [unique.Global].
Add an [App] field to AppResources only to override app config from env (ecfg block APP).

Pipeline ([Pipeline], [Bootstrap], [Run]):

	fill → ecfg.LoadEnv → materialize → unique.Merge → Transform → sdi.Resolve → StandBy → (Run only) Serve → runner.Runner.Run

[Bootstrap] ends at StandBy and returns. [Run] additionally calls [App.Serve], which hands off to
runner.Runner.Run — this is where every resource's Start(ctx) runs, in two waves (see
[github.com/omcrgnt/runner]'s package doc for the normal-Starter/LastStarter split).

If any [StandBy] call fails, [Bootstrap] closes every already-succeeded StandBy resource that also
implements a Close(context.Context) error method (registration order, same convention as
runner.Runner.Stop), then returns the StandBy error joined with any close errors. This runs
regardless of what the caller does with the returned error — runner.Runner.Stop is never reached in
this path (Run is never called), so without it a resource like a dialed *grpc.ClientConn would
otherwise depend on the caller exiting the process for the OS to reclaim it.

Lifecycle safety rule, across Inject, StandBy, and Start:

  - Reading another resource's Inject-computed state from inside your own Inject is unsafe — sdi.Resolve
    calls Inject across all resources in one pass ordered by registration, not by the Deps() graph.
  - Reading another resource's Inject-computed state from inside your own [StandBy] is safe: StandBy runs
    once sdi.Resolve has finished entirely, i.e. after every resource's Inject has already run.
  - Reading another resource's Start-computed state from inside your own Start is unsafe within the same
    wave — runner.Runner.Run starts every Starter in a wave concurrently, so there is no registration-order
    fallback either, and two Starts race with no guaranteed winner. A [runner.LastStarter] is the one
    exception: it may safely read normal-Starter Start-computed state.
  - Reading anything — Inject-, StandBy-, or Start-computed — from inside an ordinary method invoked only
    after the whole pipeline above has finished (i.e. real request-handling code, never Inject, StandBy, or
    Start themselves) is always safe, regardless of whether the dependency is a [runner.Starter]. Prefer
    this when a dependency's real state is only available after its own Start.

Breaking v0.21: [Bootstrap] returns *App; [Pipeline.Registry] is *unique.Registry; [App.Serve] has no registry arg;
catalog fields are [Configurable] or [ResourceFactory] resources (nil *Resource ok for configurable slots).
*/
package app

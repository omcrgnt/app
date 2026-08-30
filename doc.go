/*
Package app is the org bootstrap harness: catalog fill, env load, materialize, merge, sdi wire, and process [Run].

Import [github.com/omcrgnt/app/use] in main to register [DefaultApp] and [runner.Runner] on [unique.Global].
Add an [App] field to AppResources only to override app config from env (ecfg block APP).

Pipeline ([Pipeline], [Bootstrap], [Run]):

	fill → ecfg.LoadEnv → materialize → unique.Merge → Transform → sdi.Resolve → (Run only) Serve → runner.Runner.Run

[Bootstrap] ends at sdi.Resolve and returns. [Run] additionally calls [App.Serve], which hands off to
runner.Runner.Run — this is where the StandBy phase and every resource's Start(ctx) run, in that
order (see [github.com/omcrgnt/runner]'s package doc for the full StandBy→Start ordering, the
normal-Starter/LastStarter split, and how each phase's cleanup ties back into runner.Runner.Stop).

Lifecycle safety rule, across Inject, StandBy, and Start — see
[github.com/omcrgnt/runner]'s package doc for what StandBy and Start are and how their cleanups
work; this rule is about what each may safely read of another resource's state:

  - Reading another resource's Inject-computed state from inside your own Inject is unsafe — sdi.Resolve
    calls Inject across all resources in one pass ordered by registration, not by the Deps() graph.
  - Reading another resource's Inject-computed state from inside your own [runner.StandBy] is safe:
    StandBy runs once sdi.Resolve has finished entirely, i.e. after every resource's Inject has already run.
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

Breaking v0.24: the StandBy phase moved to [github.com/omcrgnt/runner] ([runner.StandBy]) — [Bootstrap]
no longer runs or knows about it; [App.Serve] → runner.Runner.Run now covers it instead.
*/
package app

/*
Package app is the org bootstrap harness: catalog fill, env load, materialize, merge, sdi wire, and process [Run].

Import [github.com/omcrgnt/app/use] in main to register [DefaultApp] and [runner.Runner] on [unique.Global].
Add an [App] field to AppResources only to override app config from env (ecfg block APP).

Pipeline ([Pipeline], [Bootstrap], [Run]):

	fill → ecfg.LoadEnv → materialize → unique.Merge → Transform → sdi.Resolve → (Run only) Serve → runner.Runner.Run

[Bootstrap] ends at sdi.Resolve and returns. [Run] additionally calls [App.Serve], which hands off to
runner.Runner.Run — this is where every resource's Start(ctx) runs, concurrently, with no ordering between them.

Lifecycle safety rule, across Inject and Start:

  - Reading another resource's Inject-computed state from inside your own Inject is unsafe — sdi.Resolve
    calls Inject across all resources in one pass ordered by registration, not by the Deps() graph.
  - Reading another resource's Start-computed state from inside your own Start is unsafe for the same
    reason, and worse: runner.Runner.Run starts every Starter concurrently, so there is no registration-order
    fallback either — two Starts race with no guaranteed winner.
  - Reading anything — Inject- or Start-computed — from inside an ordinary method invoked only after the
    whole pipeline above has finished (i.e. real request-handling code, never Inject or Start themselves) is
    always safe, regardless of whether the dependency is a [runner.Starter]. Prefer this when a dependency's
    real state is only available after its own Start.

Breaking v0.21: [Bootstrap] returns *App; [Pipeline.Registry] is *unique.Registry; [App.Serve] has no registry arg;
catalog fields are [Configurable] or [ResourceFactory] resources (nil *Resource ok for configurable slots).
*/
package app

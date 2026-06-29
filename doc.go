/*
Package app is the org bootstrap harness: catalog fill, env load, materialize, merge, sdi wire, and process [Run].

Import [github.com/omcrgnt/app/use] in main to register [DefaultApp] and [runner.Runner] on [unique.Global].
Add an [App] field to AppResources only to override app config from env (ecfg block APP).

Pipeline ([Pipeline], [Bootstrap], [Run]):

	fill → ecfg.LoadEnv → materialize → unique.Merge → Transform → sdi.Resolve

Breaking v0.21: [Bootstrap] returns *App; [Pipeline.Registry] is *unique.Registry; [App.Serve] has no registry arg;
catalog wire types are [Configurable] and [ResourceFactory] (not builder).
*/
package app

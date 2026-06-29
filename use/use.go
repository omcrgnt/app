// Package use registers framework defaults on [unique.Global].
//
// Import for side effects at the app composition root (main or a meta use package):
//
//	import _ "github.com/omcrgnt/app/use"
//
// Registers [app.DefaultApp] (replaceable) and [runner.Runner] (fixed, via runner/use).
// Override app config via an [app.App] field with ecfg tag in AppResources.
package use

import (
	"github.com/omcrgnt/app"
	"github.com/omcrgnt/res/unique"

	_ "github.com/omcrgnt/runner/use"
)

func init() {
	unique.MustAddReplaceable(app.DefaultApp())
}

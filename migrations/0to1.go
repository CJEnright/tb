package migrations

import "github.com/cjenright/tb"

// Make names with "/" into hierarchy
func migrate0to1(tbw *tb.TBWrapper) {
	// We can assume projects are already sorted because TBWrapper.Save always
	// does it. This means we can just iterate over the list of projects in order
	// and create the hierarchy as we go.

	for _, p := range tbw.Projects {
	}

	tbw.Config.Version = 1
}

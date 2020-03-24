package migrations

import (
	"fmt"

	"github.com/cjenright/tb"
)

// Make names with "/" into hierarchy
func migrate0to1(tbw *tb.TBWrapper) {
	// We can assume projects are already sorted because TBWrapper.Save always
	// does it. This means we can just iterate over the list of projects in order
	// and create the hierarchy as we go.
	newRoot := &tb.Project{}

	for _, p := range tbw.Projects {
		newRoot.New("/" + p.Name)
		projs := newRoot.FindProjects("/" + p.Name)
		if len(projs) != 1 {
			fmt.Println("error while migrating, multiple projects match", p.Name)
			return
		}
		newName := projs[0].Name
		*projs[0] = *p
		projs[0].Name = newName
	}

	tbw.Root = newRoot

	tbw.Conf.Version = 1
}

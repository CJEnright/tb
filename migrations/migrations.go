package migrations

import (
	"github.com/cjenright/tb"
)

// The function at migrationMap[n] converts a TBWrapper of version
// n to the next version
var migrations = [...]func(*tb.TBWrapper){
	migrate0to1,
}

func Migrate(tbw *tb.TBWrapper) {
	for tbw.Conf.Version != tb.CurrentVersion {
		migrations[tbw.Conf.Version](tbw)
	}
}

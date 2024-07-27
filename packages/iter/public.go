package iter

import "github.com/hikitani/easylang/packages"

var Package = packages.
	New("iter").
	AddFunc("from", Iter).
	AddFunc("range", Range).
	Build()

package ed

import "github.com/tcrain/cons/consensus/auth/sig"

var EDDSAStats = sig.SigStats{
	PubSize:       32,
	SigSize:       64,
	SigVerifyTime: 342579,
	SignTime:      105099,
}

var SchnorrStats = sig.SigStats{
	PubSize:          33,
	SigSize:          64,
	ThrshShareSize:   192,
	SigVerifyTime:    359155,
	SignTime:         199822,
	ShareVerifyTime:  1208729,
	ShareGenTime:     1038380,
	ShareCombineTime: 1814843 / 7,
	AllowsCoin:       true,
}

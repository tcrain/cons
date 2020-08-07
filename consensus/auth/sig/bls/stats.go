package bls

import "github.com/tcrain/cons/consensus/auth/sig"

var Stats = sig.SigStats{
	PubSize:          128,
	SigSize:          64,
	ThrshShareSize:   64,
	VRFSize:          64,
	SigVerifyTime:    3329283,
	SignTime:         439761,
	ShareVerifyTime:  3329283,
	ShareGenTime:     439761,
	MultiCombineTime: 3352,
	VRFGenTime:       439761,
	VRFVerifyTime:    3329283,
	ShareCombineTime: 481735 / 7,
	AllowsMulti:      true,
	AllowsThresh:     true,
	AllowsVRF:        true,
}

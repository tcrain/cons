package ec

import "github.com/tcrain/cons/consensus/auth/sig"

var Stats = sig.SigStats{
	PubSize:       65,
	SigSize:       73,
	VRFSize:       129,
	SigVerifyTime: 86693,
	SignTime:      34126,
	VRFGenTime:    256468,
	VRFVerifyTime: 324876,
	AllowsVRF:     true,
}

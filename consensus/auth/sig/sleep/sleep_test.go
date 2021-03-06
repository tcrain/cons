package sleep

import (
	"github.com/tcrain/cons/consensus/auth/bitid"
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/types"
	"testing"
)

func getUseMultiSig(stats sig.SigStats) types.BoolSetting {
	if stats.AllowsMulti {
		return types.WithBothBool
	}
	return types.WithFalse
}

func TestSleepEncode(t *testing.T) {
	privFunc, nxtStat := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for i, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestEncode(nxt, t) },
			types.WithFalse, getUseMultiSig(nxtStat[i]), types.WithFalse, types.WithFalse)
	}
}

func TestSleepSharedSecret(t *testing.T) {
	privFunc, nxtStat := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for i, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestComputeSharedSecret(nxt, t) },
			types.WithFalse, getUseMultiSig(nxtStat[i]), types.WithFalse, types.WithFalse)
	}
}

func TestSleepFromBytes(t *testing.T) {
	privFunc, nxtStat := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for i, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestFromBytes(nxt, t) },
			types.WithBothBool, getUseMultiSig(nxtStat[i]), types.WithFalse, types.WithFalse)
	}
}

func TestSleepSort(t *testing.T) {
	privFunc, nxtStat := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for i, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSort(nxt, t) },
			types.WithBothBool, getUseMultiSig(nxtStat[i]), types.WithFalse, types.WithFalse)
	}
}

func TestSleepSign(t *testing.T) {
	privFunc, nxtStat := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for i, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSign(nxt, types.NormalSignature, t) },
			types.WithBothBool, getUseMultiSig(nxtStat[i]), types.WithFalse, types.WithFalse)
	}
}

func TestSleepGetRand(t *testing.T) {
	privFunc, _ := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for _, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestRand(nxt, t) },
			types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
	}
}

func TestSleepSerialize(t *testing.T) {
	privFunc, _ := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for _, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSerialize(nxt, types.NormalSignature, t) },
			types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
	}
}

func TestSleepVRF(t *testing.T) {
	privFunc, _ := GetAllSigStatsNewPriv(true, false, bitid.NewBitIDFuncs)
	for _, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestVRF(nxt, t) },
			types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
	}
}

func TestSleepMultiSignTestMsgSerialize(t *testing.T) {
	privFunc, _ := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for _, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestMultiSignTestMsgSerialize(nxt, t) },
			types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
	}
}

func TestSleepSignTestMsgSerialize(t *testing.T) {
	privFunc, _ := GetAllSigStatsNewPriv(false, false, bitid.NewBitIDFuncs)
	for _, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.SigTestSignTestMsgSerialize(nxt, t) },
			types.WithBothBool, types.WithFalse, types.WithFalse, types.WithFalse)
	}
}

func TestBlsSignMerge(t *testing.T) {
	privFunc, _ := GetAllSigStatsNewPriv(false, true, bitid.NewBitIDFuncs)
	for _, nxt := range privFunc {
		sig.RunFuncWithConfigSetting(func() { sig.TestSigMerge(nxt, t) },
			types.WithBothBool, types.WithTrue, types.WithFalse, types.WithFalse)
	}
}

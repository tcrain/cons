/*
github.com/tcrain/cons - Experimental project for testing and scaling consensus algorithms.
Copyright (C) 2020 The project authors - tcrain

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/
package cons

import "github.com/tcrain/cons/consensus/types"

// TestOptionIterator is an interface for iterating TestOptions.
type TestOptionIterator interface {
	// Next returns the next test options, and if any more remain.
	Next() (itm types.TestOptions, hasNext bool)
}

// NewTestOptIter takes as input a TestOptIterator, and a consensus config options.
// It creates an iterator that iterates toIter, but only returns the TestOptions that are valid for the config options.
// It implements TestOpIterator.
func NewTestOptIter(opt GetOptionType, consConfigs ConfigOptions,
	toIter TestOptionIterator) (*TestOptIter, error) {

	ret := &TestOptIter{
		configIter:  toIter,
		opt:         opt,
		consConfigs: consConfigs,
	}
	// We have to first generate the first valid config
	if _, ok := ret.Next(); !ok {
		return nil, noValidConfigs
	}
	return ret, nil
}

// Next returns the next test options, and if any more remain.
func (toi *TestOptIter) Next() (to types.TestOptions, hasNext bool) {
	if toi.done {
		return toi.next, false
	}
	to = toi.next
	nxt, hasNext := toi.configIter.Next()
	for ; hasNext; nxt, hasNext = toi.configIter.Next() {
		if checkConflict(toi.opt, toi.consConfigs, nxt) == nil {
			toi.next = nxt
			return to, true
		}
	}
	toi.done = true
	if checkConflict(toi.opt, toi.consConfigs, nxt) == nil {
		toi.next = nxt
		return to, true
	}
	return to, false
}

// TestOptIter implements an iterator that only returns TestOptions that are valid for a given config options.
// It implements TestOpIterator.
type TestOptIter struct {
	next        types.TestOptions
	configIter  TestOptionIterator
	opt         GetOptionType
	consConfigs ConfigOptions
	done        bool
}

// NewAllConfigIter creates an iterator starting at to, then going through all combinations
// of testConfigs.
func NewAllConfigIter(testConfigs OptionStruct, to types.TestOptions) TestOptionIterator {
	ret := &ConfigIter{
		testConfigs: testConfigs,
		to:          to,
	}
	ret.items = ret.genItems()
	return ret
}

// Next returns the next test options, and if any more remain.
func (ci *ConfigIter) Next() (itm types.TestOptions, hasNext bool) {
	itm = ci.items[ci.id]
	ci.id++
	return itm, ci.id < len(ci.items)
}

// ConfigIter is an iterates over all combinations (exponential) of an OptionStruct starting
// from a base testOptions. It implements TestOpIterator.
// Why did I make this???
type ConfigIter struct {
	id          int
	testConfigs OptionStruct
	to          types.TestOptions
	done        bool
	items       []types.TestOptions
}

// Next returns the next test options, and if any more remain.
func (ci *ConfigIter) genItems() (ret []types.TestOptions) {
	for _, ot := range ci.testConfigs.OrderingTypes {
		ci.to.OrderingType = ot
		for _, smt := range ci.testConfigs.StateMachineTypes {
			ci.to.StateMachineType = smt
			for _, byz := range ci.testConfigs.ByzTypes {
				ci.to.ByzType = byz
				for _, mct := range ci.testConfigs.MemberCheckerTypes {
					ci.to.MCType = mct
					for _, ipt := range ci.testConfigs.IncludeProofsTypes {
						ci.to.IncludeProofs = ipt
						for _, asct := range ci.testConfigs.AllowSupportCoinTypes {
							ci.to.AllowSupportCoin = asct
							for _, act := range ci.testConfigs.AllowConcurrentTypes {
								ci.to.AllowConcurrent = uint64(act)
								for _, rmct := range ci.testConfigs.RandMemberCheckerTypes {
									ci.to.RndMemberType = rmct
									for _, rct := range ci.testConfigs.RotateCoordTypes {
										ci.to.RotateCord = rct
										for _, st := range ci.testConfigs.SigTypes {
											ci.to.SigType = st
											for _, cb := range ci.testConfigs.CollectBroadcast {
												ci.to.CollectBroadcast = cb
												for _, pit := range ci.testConfigs.UsePubIndexTypes {
													ci.to.UsePubIndex = pit
													ret = append(ret, ci.to)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return
}

// NewSingleIter returns a iterator through the testConfigs with to used as the base config.
// It implements TestOpIterator.
func NewSingleIter(testConfigs OptionStruct, to types.TestOptions) TestOptionIterator {
	ret := &SingleIter{
		testConfigs: testConfigs,
		to:          to,
	}
	ret.genItems()
	return ret
}

// SingleIter starts with a base test options, then iterates through each field of the test configs, changing
// one value then going back to the original test options.
// I.e. if the number of elements in all the slices of testConfigs is 10, then 10 tests are run.
type SingleIter struct {
	id          int
	testConfigs OptionStruct
	to          types.TestOptions
	items       []types.TestOptions
}

// Next returns the next test options, and if any more remain.
func (ci *SingleIter) Next() (itm types.TestOptions, hasNext bool) {
	itm = ci.items[ci.id]
	ci.id++
	return itm, ci.id < len(ci.items)
}

func (ci *SingleIter) genItems() {
	// TODO do this better
	itms := make(map[types.TestOptions]bool)
	var ret []types.TestOptions

	nxt := ci.to
	for _, ot := range ci.testConfigs.OrderingTypes {
		nxt.OrderingType = ot
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, fp := range ci.testConfigs.UseFixedCoinPresets {
		nxt.UseFixedCoinPresets = fp
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, smt := range ci.testConfigs.StateMachineTypes {
		nxt.StateMachineType = smt
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, bt := range ci.testConfigs.ByzTypes {
		nxt.ByzType = bt
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, ct := range ci.testConfigs.CoinTypes {
		nxt.CoinType = ct
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, mct := range ci.testConfigs.MemberCheckerTypes {
		nxt.MCType = mct
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, sct := range ci.testConfigs.StopOnCommitTypes {
		nxt.StopOnCommit = sct
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, ans := range ci.testConfigs.AllowNoSignaturesTypes {
		nxt.NoSignatures = ans
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, ipt := range ci.testConfigs.IncludeProofsTypes {
		nxt.IncludeProofs = ipt
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, asct := range ci.testConfigs.AllowSupportCoinTypes {
		nxt.AllowSupportCoin = asct
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, act := range ci.testConfigs.AllowConcurrentTypes {
		nxt.AllowConcurrent = uint64(act)
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, ect := range ci.testConfigs.EncryptChannelsTypes {
		nxt.EncryptChannels = ect
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, rmct := range ci.testConfigs.RandMemberCheckerTypes {
		nxt.RndMemberType = rmct
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, rct := range ci.testConfigs.RotateCoordTypes {
		nxt.RotateCord = rct
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, st := range ci.testConfigs.SigTypes {
		nxt.SigType = st
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, pit := range ci.testConfigs.UsePubIndexTypes {
		nxt.UsePubIndex = pit
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to
	for _, cb := range ci.testConfigs.CollectBroadcast {
		nxt.CollectBroadcast = cb
		if !itms[nxt] {
			ret = append(ret, nxt)
		}
		itms[nxt] = true
	}
	nxt = ci.to

	ci.items = ret
}

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
/*
Package for keeping statistics about the execution of consensus.
*/
package stats

import (
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/consensus/utils"
)

func GetStatsObject(ct types.ConsType, encryptChannels bool) StatsInterface {
	return &BasicStats{globalStats: &globalStats{BasicNwStats: BasicNwStats{EncryptChannels: encryptChannels}}}
}

///////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////

type StatsPrintInterface interface {
	GetTotalTime() time.Duration // Sum of all executed time
	GetAvgTime() time.Duration
	GetMaxTime() time.Duration
	GetAvgRoundDecide() float32
	GetMaxRoundDecide() uint64
	GetMinRoundDecide() uint64
	GetSummedTime() time.Duration
	GetTotalTimeDivNumCons() time.Duration
	GetRecordedCount() int
}

// StatsInterface is the interface for objects that keep statistics about the consensus.
type StatsInterface interface {
	StatsPrintInterface

	New(index types.ConsensusIndex) StatsInterface
	StartRecording(profileCPU, profileMem bool, testIndex int, testID uint64) // StartRecording starts recording the stats.
	DoneRecording()                                                           // DoneRecording stops recording the stats.
	AddStartTime()                                                            // AddStartTime is called each time a new consesnsus instance is started.
	// AddFinishTime()                                                           // AddFinishTime is called time a consensus instance decides.
	SignedItem()                                            // SignedItem is called each time consensus signs a message.
	ValidatedItem()                                         // ValidatedItem is called each time consensus validates a signature.
	ValidatedCoinShare()                                    // ValidatedCoinShare is call each time a coin share is validated
	ComputedCoin()                                          // ComputedCoin is called each time a coin is computed.
	ValidatedVRF()                                          // ValidatedVRF is called each time a VRF is validated
	CreatedVRF()                                            // CreatedVRF is called each time a VRF is created.
	String() string                                         // String outputs the statistics in a human readable format.
	AddParticipationRound(r types.ConsensusRound)           // Called when the node participates in a round r
	AddFinishRound(r types.ConsensusRound, decidedNil bool) // Called when the node decides at round r
	CombinedThresholdSig()                                  // Called when a threshold signature is combined.
	DiskStore(bytes int)                                    // Called when storage to disk is done with the number of bytes stored.
	Restart()                                               // Called on restart of failure so it knows to start recording again
	IsRecordIndex() bool                                    // IsRecordIndex returns true if the consensus index of this stats is being recorded.

	MergeLocalStats(numCons int) (total MergedStats)
	// Merge all stats is a static function that merges the list of stats, and returns the average stats per process, and the total summed stats.
	MergeAllStats(items []MergedStats) (perProc, merge MergedStats)

	NwStatsInterface
}

// MergeStats calls `items[0].MergeAllStats(numCons, items)` if len(items) > 0
func MergeStats(items []MergedStats) (perProc, merge MergedStats) {
	switch len(items) {
	case 0:
		return
	default:
		perProc, merge = (&BasicStats{}).MergeAllStats(items)
		return
	}
}

///////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////

type globalStats struct {
	BasicNwStats
	stats            []StatsInterface
	recording        bool // True means to track the stats
	profileCPU       bool
	profileMem       bool
	pFile            *os.File
	pFileName        string
	startMemFileName string
	endMemFileName   string
	doneProf         sync.WaitGroup
	doneRecording    bool
}

var DivStats = map[string]bool{"RoundParticipation": true, "RoundDecide": true, "DiskStorage": true,
	"Signed": true, "ThrshCreated": true, "Validated": true, "VRFCreated": true, "VRFValidated": true,
	"CoinValidated": true, "CoinCreated": true,
	"MsgsSent": true, "BytesSent": true, "MaxMsgsSent": true, "MaxBytesSent": true, "MinMsgsSent": true, "MinBytesSent": true}

type StatsObj struct {
	StartTime          time.Time // List of the times when consensus instances were started.
	FinishTime         time.Time // List of the times where consensus instances finished.
	Signed             uint64    // Number of signatures this node has made.
	Validated          uint64    // Number of signature validations this node has performed.
	CoinValidated      uint64    // Number of coin validations
	CoinCreated        uint64    // Number of coins created
	VRFCreated         uint64    // Number of VRFs created
	VRFValidated       uint64    // Number of VRFs validated
	ThrshCreated       uint64    // Number of threshold signatures created
	RoundDecide        uint64    // Round where decision took place
	RoundParticipation uint64    // Last round participated in
	DiskStorage        uint64    // Number of bytes written to disk
	DecidedNil         uint64    //  True if nil was decided
}

type MergedStats struct {
	MergedNwStats
	StatsObj
	StartTimes                                                                                                                                                                       []time.Time
	FinishTimes                                                                                                                                                                      []time.Time
	ConsTimes                                                                                                                                                                        []time.Duration
	SinceTimes                                                                                                                                                                       []time.Duration // Time since the previous decided
	MaxConsTime, MinConsTime                                                                                                                                                         time.Duration
	ConsTime                                                                                                                                                                         time.Duration
	MaxValidatedCoin, MaxCoinCreated, MaxDecidedNil, MaxDiskStorage, MaxSigned, MaxValidated, MaxVRFValidated, MaxVRFCreated, MaxRoundDecide, MaxRoundParticipation, MaxThrshCreated uint64
	MinValidatedCoin, MinCoinCreated, MinDecidedNil, MinDiskStorage, MinSigned, MinValidated, MinVRFValidated, MinVRFCreated, MinRoundDecide, MinRoundParticipation, MinThrshCreated uint64
	RecordCount                                                                                                                                                                      int
	CpuProfile                                                                                                                                                                       []byte
	StartMemProfile                                                                                                                                                                  []byte
	EndMemProfile                                                                                                                                                                    []byte
}

// BasicStats track basic statistics about the consensus.
type BasicStats struct {
	*globalStats
	StatsObj
	Decided     bool
	RecordIndex bool // true if we are keeping stats for this index
	Index       types.ConsensusID
}

// IsRecordIndex returns true if the consensus index of this stats is being recorded.
func (bs *BasicStats) IsRecordIndex() bool {
	return bs.RecordIndex
}

// CombinedThresholdSig is called when a threshold signature is combined.
func (bs *BasicStats) CombinedThresholdSig() {
	bs.ThrshCreated++
}

// Restart is called on restart of failure so it knows to start recording again
func (bs *BasicStats) Restart() {
	bs.doneRecording = false
}

// DiskStore is called when storage to disk is done with the number of bytes stored.
func (bs *BasicStats) DiskStore(bytes int) {
	bs.DiskStorage += uint64(bytes)
}

const ProfileOut = "testprofile"

// StartRecording starts recording the stats.
func (bs *BasicStats) StartRecording(profileCPU, profileMem bool, testIndex int, testID uint64) {
	if bs.recording || bs.doneRecording {
		return
	}

	bs.RecordIndex = true
	if profileCPU {
		var err error
		if err = os.MkdirAll(ProfileOut, os.ModePerm); err != nil {
			panic(err)
		}

		bs.pFileName = filepath.Join(ProfileOut, fmt.Sprintf("cpuprof%v_proc%v.out", testID, testIndex))
		bs.pFile, err = os.Create(bs.pFileName)
		if err != nil {
			panic(err)
		}
		if err = pprof.StartCPUProfile(bs.pFile); err != nil {
			logging.Error(err)
			if err = bs.pFile.Close(); err != nil {
				logging.Error(err)
			}
			bs.profileCPU = false
		} else {
			bs.profileCPU = true
		}
	}
	if profileMem {
		var err error
		if err = os.MkdirAll(ProfileOut, os.ModePerm); err != nil {
			panic(err)
		}

		bs.profileMem = true

		bs.startMemFileName = filepath.Join(ProfileOut, fmt.Sprintf("memprof_start%v_proc%v.out", testID, testIndex))

		bs.doneProf.Add(1)
		go func() {
			memfile, err := os.Create(bs.startMemFileName)
			if err != nil {
				panic(err)
			}
			defer func() {
				if err := memfile.Close(); err != nil {
					logging.Error(err)
				}
			}()

			if err = pprof.Lookup("allocs").WriteTo(memfile, 0); err != nil {
				panic(err)
			}
			bs.doneProf.Done()
		}()
		// for the end of the test
		bs.endMemFileName = filepath.Join(ProfileOut, fmt.Sprintf("memprof_finish%v_proc%v.out", testID, testIndex))
	}

	bs.recording = true
	//atomic.StoreInt32(&bs.ValidatedCountAtomic, 0)
	//atomic.StoreInt32(&bs.VRFValidatedAtomic, 0)
}

// DoneRecording stops recording the stats.
func (bs *BasicStats) DoneRecording() {
	bs.recording = false
	bs.doneRecording = true
	if bs.profileCPU {
		pprof.StopCPUProfile()
		if err := bs.pFile.Close(); err != nil {
			logging.Error(err)
		}
	}
	if bs.profileMem {
		bs.doneProf.Add(1)
		go func() {
			memfile, err := os.Create(bs.endMemFileName)
			if err != nil {
				panic(err)
			}
			defer func() {
				if err := memfile.Close(); err != nil {
					logging.Error(err)
				}
			}()
			if err = pprof.Lookup("allocs").WriteTo(memfile, 0); err != nil {
				panic(err)
			}
			bs.doneProf.Done()
		}()
	}
	// bs.VRFValidated = int(atomic.LoadInt32(&bs.VRFValidatedAtomic))
	// bs.ValidatedItem = int(atomic.LoadInt32(&bs.ValidatedCountAtomic))
	// bs.DoneRecordingNW()
}

func (bs *BasicStats) AddParticipationRound(r types.ConsensusRound) {
	if bs.RoundParticipation < uint64(r) {
		bs.RoundParticipation = uint64(r)
	}
}

func (bs *BasicStats) AddFinishRound(r types.ConsensusRound, decidedNil bool) {
	bs.Decided = true
	if decidedNil {
		bs.DecidedNil++
	}
	bs.FinishTime = time.Now()
	bs.RoundDecide = uint64(r)
	bs.AddParticipationRound(r)
}

func mergeInternal(items []StatsObj) (reTotal MergedStats) {
	var setStart bool
	var coinCreatedCounts, validateCoinCounts, decidedNil, diskStorage, roundDecides, roundParticipations, signedCounts, validatedCounts, VRFCreatedCounts, VRFValidatedCouts, thrshCreated []uint64
	var times []time.Duration

	prevTime := items[0].StartTime
	for _, item := range items {

		if !setStart || reTotal.StartTime.After(item.StartTime) {
			reTotal.StartTime = item.StartTime
			// reTotal.StartTime = item.StartTime
			setStart = true
		}
		if reTotal.FinishTime.Before(item.FinishTime) {
			reTotal.FinishTime = item.FinishTime
		}
		reTotal.StartTimes = append(reTotal.StartTimes, item.StartTime)
		reTotal.FinishTimes = append(reTotal.FinishTimes, item.FinishTime)
		reTotal.SinceTimes = append(reTotal.SinceTimes, item.FinishTime.Sub(prevTime))
		prevTime = item.FinishTime

		reTotal.ConsTime += item.FinishTime.Sub(item.StartTime)
		times = append(times, item.FinishTime.Sub(item.StartTime))

		reTotal.DecidedNil += item.DecidedNil
		decidedNil = append(decidedNil, item.DecidedNil)

		reTotal.RoundDecide += item.RoundDecide
		roundDecides = append(roundDecides, item.RoundDecide)

		reTotal.RoundParticipation += item.RoundParticipation
		roundParticipations = append(roundParticipations, item.RoundParticipation)

		reTotal.Signed += item.Signed
		signedCounts = append(signedCounts, item.Signed)

		reTotal.Validated += item.Validated
		validatedCounts = append(validatedCounts, item.Validated)

		reTotal.CoinCreated += item.CoinCreated
		coinCreatedCounts = append(coinCreatedCounts, item.CoinCreated)

		reTotal.CoinValidated += item.CoinValidated
		validateCoinCounts = append(validateCoinCounts, item.CoinValidated)

		reTotal.VRFCreated += item.VRFCreated
		VRFCreatedCounts = append(VRFCreatedCounts, item.VRFCreated)

		reTotal.VRFValidated += item.VRFValidated
		VRFValidatedCouts = append(VRFValidatedCouts, item.VRFValidated)

		reTotal.ThrshCreated += item.ThrshCreated
		thrshCreated = append(thrshCreated, item.ThrshCreated)

		reTotal.DiskStorage += item.DiskStorage
		diskStorage = append(diskStorage, item.DiskStorage)
	}

	reTotal.ConsTimes = times
	reTotal.MinConsTime = utils.MinDuration(times...)
	reTotal.MaxConsTime = utils.MaxDuration(times...)
	reTotal.MinRoundDecide = utils.MinU64Slice(roundDecides...)
	reTotal.MaxRoundDecide = utils.MaxU64Slice(roundDecides...)
	reTotal.MinRoundParticipation = utils.MinU64Slice(roundParticipations...)
	reTotal.MaxRoundParticipation = utils.MaxU64Slice(roundParticipations...)
	reTotal.MinSigned = utils.MinU64Slice(signedCounts...)
	reTotal.MaxSigned = utils.MaxU64Slice(signedCounts...)
	reTotal.MinValidated = utils.MinU64Slice(validatedCounts...)
	reTotal.MaxValidated = utils.MaxU64Slice(validatedCounts...)
	reTotal.MinCoinCreated = utils.MinU64Slice(coinCreatedCounts...)
	reTotal.MaxCoinCreated = utils.MaxU64Slice(coinCreatedCounts...)
	reTotal.MinValidatedCoin = utils.MinU64Slice(validateCoinCounts...)
	reTotal.MaxValidatedCoin = utils.MaxU64Slice(validateCoinCounts...)
	reTotal.MinVRFCreated = utils.MinU64Slice(VRFCreatedCounts...)
	reTotal.MaxVRFCreated = utils.MaxU64Slice(VRFCreatedCounts...)
	reTotal.MinVRFValidated = utils.MinU64Slice(VRFValidatedCouts...)
	reTotal.MaxVRFValidated = utils.MaxU64Slice(VRFValidatedCouts...)
	reTotal.MinThrshCreated = utils.MinU64Slice(thrshCreated...)
	reTotal.MaxThrshCreated = utils.MaxU64Slice(thrshCreated...)
	reTotal.MinDiskStorage = utils.MinU64Slice(diskStorage...)
	reTotal.MaxDiskStorage = utils.MaxU64Slice(diskStorage...)
	reTotal.MinDecidedNil = utils.MinU64Slice(decidedNil...)
	reTotal.MaxDecidedNil = utils.MaxU64Slice(decidedNil...)

	return
}

func mergeStatsObj(a, b StatsObj, includeTime bool) StatsObj {
	if includeTime {
		if !b.StartTime.IsZero() && !b.FinishTime.IsZero() {
			if !a.StartTime.IsZero() && !a.FinishTime.IsZero() {
				// Take the one that finished later
				if b.FinishTime.After(a.FinishTime) {
					a.StartTime = b.StartTime
					a.FinishTime = b.FinishTime
				}
			} else {
				a.StartTime = b.StartTime
				a.FinishTime = b.FinishTime
			}
		}
	}
	a.DecidedNil += b.DecidedNil
	a.DiskStorage += b.DiskStorage
	a.ThrshCreated += b.ThrshCreated
	a.VRFValidated += b.VRFValidated
	a.VRFCreated += b.VRFCreated
	a.RoundDecide += b.RoundDecide
	a.RoundParticipation += b.RoundParticipation
	a.Validated += b.Validated
	a.Signed += b.Signed
	a.CoinValidated += b.CoinValidated
	a.CoinCreated += a.CoinCreated
	return a
}

func (bs *BasicStats) MergeLocalStats(numCons int) (total MergedStats) {

	// Remove duplicates (can happen on fail and restart)
	for i, nxt := range bs.stats {
		if nxt == nil {
			continue
		}
		idx := nxt.(*BasicStats).Index
		if !nxt.(*BasicStats).FinishTime.IsZero() && nxt.(*BasicStats).FinishTime.Sub(nxt.(*BasicStats).StartTime) < 0 {
			panic("negative time")
		}

		// If this instance didn't decide then we add its results to the following
		if !nxt.(*BasicStats).Decided {
			for j, nxtNxt := range bs.stats[i+1:] {
				if nxtNxt != nil {
					// add the results to the next non nil object and update that object
					// Non-decided counts as nil-decision so increment decidedNil
					nxt.(*BasicStats).DecidedNil++
					bs.stats[i+j+1].(*BasicStats).StatsObj = mergeStatsObj(nxtNxt.(*BasicStats).StatsObj,
						nxt.(*BasicStats).StatsObj, false)
					bs.stats[i] = nil // we no longer use this one
					break             // exit for loop
				}
			}
			continue // Go to the next
		}
		for j, nxtJ := range bs.stats[i+1:] {
			if nxtJ != nil && nxtJ.(*BasicStats).Index == idx {
				// merge the duplicates
				bs.stats[i].(*BasicStats).StatsObj = mergeStatsObj(nxt.(*BasicStats).StatsObj,
					nxtJ.(*BasicStats).StatsObj, true)
				bs.stats[i+j+1] = nil
				// if either of them are to be recorded then we set recorded to true
				if nxt.(*BasicStats).RecordIndex || nxtJ.(*BasicStats).RecordIndex {
					bs.stats[i].(*BasicStats).RecordIndex = true
				}
			}
		}
	}

	var items []StatsObj
	for _, nxt := range bs.stats {
		if nxt != nil && nxt.(*BasicStats).RecordIndex && len(items) < numCons {
			items = append(items, nxt.(*BasicStats).StatsObj)
		}
	}
	if len(items) != numCons {
		panic("should have stats for each cons")
	}
	total = mergeInternal(items)
	total.RecordCount = numCons
	total.BasicNwStats = bs.BasicNwStats

	// Add the profile file as bytes
	var err error
	if bs.profileCPU {
		if total.CpuProfile, err = ioutil.ReadFile(bs.pFileName); err != nil {
			panic(err)
		}
	}
	if bs.profileMem {
		// be sure we have finished writing the files
		bs.doneProf.Wait()
		// Load the files
		if total.StartMemProfile, err = ioutil.ReadFile(bs.startMemFileName); err != nil {
			panic(err)
		}
		if total.EndMemProfile, err = ioutil.ReadFile(bs.endMemFileName); err != nil {
			panic(err)
		}
	}

	return
}

func (bs *BasicStats) MergeAllStats(sList []MergedStats) (perProc, totals MergedStats) {
	if len(sList) == 0 {
		return
	}
	perProc = sList[0]

	var items []StatsObj
	var nwItems []BasicNwStats
	var summedTime time.Duration

	// slice is [#cons][#participants]
	startTimes := make([][]time.Time, sList[0].RecordCount)
	endTimes := make([][]time.Time, sList[0].RecordCount)
	for _, nxt := range sList {

		for i := range nxt.StartTimes {
			startTimes[i] = append(startTimes[i], nxt.StartTimes[i])
			endTimes[i] = append(endTimes[i], nxt.FinishTimes[i])
		}

		nwItems = append(nwItems, nxt.BasicNwStats)
		items = append(items, nxt.StatsObj)
		perProc.MinConsTime = utils.MinDuration(perProc.MinConsTime, nxt.MinConsTime)
		perProc.MaxConsTime = utils.MaxDuration(perProc.MaxConsTime, nxt.MaxConsTime)
		perProc.MinRoundDecide = utils.MinU64Slice(perProc.MinRoundDecide, nxt.MinRoundDecide)
		perProc.MaxRoundDecide = utils.MaxU64Slice(perProc.MaxRoundDecide, nxt.MaxRoundDecide)
		perProc.MinRoundParticipation = utils.MinU64Slice(perProc.MinRoundParticipation, nxt.MinRoundParticipation)
		perProc.MaxRoundParticipation = utils.MaxU64Slice(perProc.MaxRoundParticipation, nxt.MaxRoundParticipation)
		perProc.MinSigned = utils.MinU64Slice(perProc.MinSigned, nxt.MinSigned)
		perProc.MaxSigned = utils.MaxU64Slice(perProc.MaxSigned, nxt.MaxSigned)
		perProc.MinValidated = utils.MinU64Slice(perProc.MinValidated, nxt.MinValidated)
		perProc.MaxValidated = utils.MaxU64Slice(perProc.MaxValidated, nxt.MaxValidated)
		perProc.MinCoinCreated = utils.MinU64Slice(perProc.MinCoinCreated, nxt.MinCoinCreated)
		perProc.MaxCoinCreated = utils.MaxU64Slice(perProc.MaxCoinCreated, nxt.MaxCoinCreated)
		perProc.MinValidatedCoin = utils.MinU64Slice(perProc.MinValidatedCoin, nxt.MinValidatedCoin)
		perProc.MaxValidatedCoin = utils.MaxU64Slice(perProc.MaxValidatedCoin, nxt.MaxValidatedCoin)
		perProc.MinVRFCreated = utils.MinU64Slice(perProc.MinVRFCreated, nxt.MinVRFCreated)
		perProc.MaxVRFCreated = utils.MaxU64Slice(perProc.MaxVRFCreated, nxt.MaxVRFCreated)
		perProc.MinVRFValidated = utils.MinU64Slice(perProc.MinVRFValidated, nxt.MinVRFValidated)
		perProc.MaxVRFValidated = utils.MaxU64Slice(perProc.MaxVRFValidated, nxt.MaxVRFValidated)
		perProc.MinThrshCreated = utils.MinU64Slice(perProc.MinThrshCreated, nxt.MinThrshCreated)
		perProc.MaxThrshCreated = utils.MaxU64Slice(perProc.MaxThrshCreated, nxt.MaxThrshCreated)
		perProc.MinDiskStorage = utils.MinU64Slice(perProc.MinDiskStorage, nxt.MinDiskStorage)
		perProc.MaxDiskStorage = utils.MaxU64Slice(perProc.MaxDiskStorage, nxt.MaxDiskStorage)
		perProc.MinDecidedNil = utils.MinU64Slice(perProc.MinDecidedNil, nxt.MinDecidedNil)
		perProc.MaxDecidedNil = utils.MaxU64Slice(perProc.MaxDecidedNil, nxt.MaxDecidedNil)

		summedTime += nxt.ConsTime
	}
	totals = mergeInternal(items)
	totals.ConsTime = summedTime
	totals.RecordCount = perProc.RecordCount
	perProc.StatsObj = totals.StatsObj

	perProc.EndMemProfile = nil
	perProc.StartMemProfile = nil
	perProc.CpuProfile = nil

	// Compute the average start and end times
	totals.StartTimes = nil
	totals.FinishTimes = nil
	totals.ConsTimes = nil
	totals.SinceTimes = nil
	for i := range startTimes {
		firstStartTime := startTimes[i][0]
		firstEndTime := endTimes[i][0]
		var sumStartTimes, sumFinishTimes time.Duration
		for j := range startTimes[i] {
			sumStartTimes += firstStartTime.Sub(startTimes[i][j])
			sumFinishTimes += firstEndTime.Sub(endTimes[i][j])
		}
		// divide for the avg
		sumStartTimes /= time.Duration(len(startTimes[i]))
		sumFinishTimes /= time.Duration(len(endTimes[i]))
		// add the negative duration back to the first start time
		startTime := firstStartTime.Add(-sumStartTimes)
		endTime := firstEndTime.Add(-sumFinishTimes)
		// add to the list
		totals.StartTimes = append(totals.StartTimes, startTime)
		totals.FinishTimes = append(totals.FinishTimes, endTime)
		// the avg duration
		totals.ConsTimes = append(totals.ConsTimes, endTime.Sub(startTime))
	}
	prevTime := totals.StartTimes[0]
	for _, nxt := range totals.FinishTimes {
		totals.SinceTimes = append(totals.SinceTimes, nxt.Sub(prevTime))
		prevTime = nxt
	}

	perProc.StartTimes = totals.StartTimes
	perProc.FinishTimes = totals.FinishTimes
	perProc.ConsTimes = totals.ConsTimes
	perProc.SinceTimes = totals.SinceTimes

	perProc.StartTime = totals.StartTimes[0]
	perProc.FinishTime = totals.FinishTimes[len(totals.FinishTimes)-1]
	totals.StartTime = perProc.StartTime
	totals.FinishTime = perProc.FinishTime

	perProc.ConsTime = summedTime / time.Duration(len(items))
	perProc.Signed /= uint64(len(items))
	perProc.Validated /= uint64(len(items))
	perProc.CoinCreated /= uint64(len(items))
	perProc.CoinValidated /= uint64(len(items))
	perProc.VRFCreated /= uint64(len(items))
	perProc.VRFValidated /= uint64(len(items))
	perProc.RoundDecide /= uint64(len(items))
	perProc.RoundParticipation /= uint64(len(items))
	perProc.ThrshCreated /= uint64(len(items))
	perProc.DiskStorage /= uint64(len(items))
	perProc.DecidedNil /= uint64(len(items))

	perProc.MergedNwStats, totals.MergedNwStats = (&BasicNwStats{}).MergeAllNWStats(perProc.RecordCount, nwItems)

	return
}

/*func mergeTimes(allTimes [][]time.Time) []time.Time {
	var mergedTimes []time.Time
	hasMore := true
	for i := 0; hasMore; i++ {
		hasMore = false
		var summed int64
		var count int64
		for j := 0; j < len(allTimes); j++ {
			if i < len(allTimes[j]) {
				hasMore = true
				count++
				summed += allTimes[j][i].UnixNano()
			}
		}
		if hasMore {
			mergedTimes = append(mergedTimes, time.Unix(0, summed/count))
		}
	}
	return mergedTimes
}
*/

func (bs *BasicStats) GetStartAndEndTimes() (startTimes, finishTimes []time.Time) {
	for _, nxt := range bs.stats {
		if nxt != nil && nxt.(*BasicStats).RecordIndex {
			if !nxt.(*BasicStats).FinishTime.IsZero() {
				startTimes = append(startTimes, nxt.(*BasicStats).StartTime)
				finishTimes = append(finishTimes, nxt.(*BasicStats).FinishTime)
			}
		}
	}
	return
}

func (bs *BasicStats) New(index types.ConsensusIndex) StatsInterface {
	ret := &BasicStats{globalStats: bs.globalStats,
		RecordIndex: bs.globalStats.recording,
		Index:       index.Index}
	bs.globalStats.stats = append(bs.globalStats.stats, ret)
	return ret
}

// AddStartTime is called each time a new consesnsus instance is started.
func (bs *BasicStats) AddStartTime() {
	if bs.StartTime.IsZero() {
		bs.StartTime = time.Now()
	}
	//if bs != nil && bs.Recording {
	//	bs.StartTimes = append(bs.StartTimes, time.Now())
	//}
}

// SignedItem is called each time consensus signs a message.
func (bs *BasicStats) SignedItem() {
	//if bs.Recording {
	bs.Signed++
	//}
}

// ValidatedItem is called each time consensus validates a signature.
func (bs *BasicStats) ValidatedItem() {
	atomic.AddUint64(&bs.Validated, 1)
}

// ValidatedCoinShare is call each time a coin share is validated
func (bs *BasicStats) ValidatedCoinShare() {
	atomic.AddUint64(&bs.CoinValidated, 1)
}

// ComputedCoin is called each time a coin is computed.
func (bs *BasicStats) ComputedCoin() {
	atomic.AddUint64(&bs.CoinCreated, 1)
}

// AddFinishTime is called time a consensus instance decides.
// func (bs *BasicStats) AddFinishTime() {
//	bs.FinishTime = time.Now()
//if bs != nil && bs.Recording {
// We might not have started this round because of receiving messages from others
// So just use the previous finish time
//	if len(bs.StartTimes) == len(bs.FinishTimes) {
//		bs.StartTimes = append(bs.StartTimes, bs.FinishTimes[len(bs.FinishTimes)-1])
//	}
//	bs.FinishTimes = append(bs.FinishTimes, time.Now())
//}
// }

func (bs *BasicStats) GetRecordedCount() (count int) {
	for _, nxt := range bs.stats {
		if nxt.(*BasicStats).RecordIndex && !nxt.(*BasicStats).FinishTime.IsZero() {
			count++
		}
	}
	return
}

// GetAvgTime returns the average duration for a consensus instance.
func (bs *BasicStats) GetAvgTime() time.Duration {

	return bs.GetSummedTime() / time.Duration(bs.GetRecordedCount())
}

// GetTotalTimeDivNumCons returns the total time divided by the number of consensus instances.
func (bs *BasicStats) GetTotalTimeDivNumCons() time.Duration {
	return bs.GetTotalTime() / time.Duration(bs.GetRecordedCount())
}

// GetMaxTime returns the longest duration consensus instance.
func (bs *BasicStats) GetMaxTime() time.Duration {
	return utils.MaxDuration(bs.getAllTimes()...)
}

// GetMinTime returns the shortest duration consensu instance.
func (bs *BasicStats) GetMinTime() time.Duration {
	return utils.MinDuration(bs.getAllTimes()...)
}

// ValidatedVRF is called each time a VRF is validated
func (bs *BasicStats) ValidatedVRF() {
	atomic.AddUint64(&bs.VRFValidated, 1)
}

// CreatedVRF is called each time a VRF is created.
func (bs *BasicStats) CreatedVRF() {
	bs.VRFCreated++
}

func (bs *BasicStats) getAllTimes() []time.Duration {
	startTimes, finishTimes := bs.GetStartAndEndTimes()
	var allTimes []time.Duration
	for i, s := range finishTimes {
		allTimes = append(allTimes, s.Sub(startTimes[i]))
	}
	return allTimes
}

// GetSummedTime returns the summed time of all consensus instances.
func (bs *BasicStats) GetSummedTime() time.Duration {
	return utils.SumDuration(bs.getAllTimes())
}

func (bs *BasicStats) getRoundDecides() (ret []uint64) {
	for _, nxt := range bs.stats {
		item := nxt.(*BasicStats)
		if item.RecordIndex {
			ret = append(ret, item.RoundDecide)
		}
	}
	return ret
}

func (bs *BasicStats) getRoundParticipation() (ret []uint64) {
	for _, nxt := range bs.stats {
		item := nxt.(*BasicStats)
		if item.RecordIndex {
			ret = append(ret, item.RoundParticipation)
		}
	}
	return ret
}

func (bs *BasicStats) GetAvgRoundDecide() float32 {
	return float32(utils.SumUint64(bs.getRoundDecides()...)) / float32(bs.GetRecordedCount())
}

func (bs *BasicStats) GetAvgRoundParticipation() float32 {
	return float32(utils.SumUint64(bs.getRoundParticipation()...)) / float32(bs.GetRecordedCount())
}

func (bs *BasicStats) GetMaxRoundDecide() uint64 {
	return utils.MaxU64Slice(bs.getRoundDecides()...)
}

func (bs *BasicStats) GetMinRoundDecide() uint64 {
	return utils.MinU64Slice(bs.getRoundDecides()...)
}

func (bs *BasicStats) GetMaxRoundParticipation() uint64 {
	return utils.MaxU64Slice(bs.getRoundParticipation()...)
}

func (bs *BasicStats) GetMinRoundParticipation() uint64 {
	return utils.MinU64Slice(bs.getRoundParticipation()...)
}

// GetTotalTime returns the duration from the start of the first consensus to the finish of the last consensu.
func (bs *BasicStats) GetTotalTime() time.Duration {
	startTimes, finishTimes := bs.GetStartAndEndTimes()
	if finishTimes[len(finishTimes)-1].UnixNano() < startTimes[0].UnixNano() {
		panic("invalid times")
	}
	return finishTimes[len(finishTimes)-1].Sub(startTimes[0])
}

// String outputs the statistics in a human readable format.
func (bs *BasicStats) String() string {
	return fmt.Sprintf("{#Cons: %v, AllTime: %v, TotalTimeDivNumCons: %v,\n\tTotalConsTime: %v, AvgConsTime: %v, MinTime: %v, MaxTime: %v, AvgRound: %v,\n\tMaxRound: %v, MinRound: %v, SigCount: %v, ValidatedItem: %v,\n\tVRFCreated: %v, VRFValidated: %v, CoinCreated: %v, CoinValidated: %v}",
		bs.GetRecordedCount(), float64(bs.GetTotalTime())/float64(time.Millisecond),
		float64(bs.GetTotalTimeDivNumCons())/float64(time.Millisecond),
		float64(bs.GetSummedTime())/float64(time.Millisecond), float64(bs.GetAvgTime())/float64(time.Millisecond),
		float64(bs.GetMinTime())/float64(time.Millisecond), float64(bs.GetMaxTime())/float64(time.Millisecond),
		bs.GetAvgRoundDecide(), bs.GetMaxRoundDecide(), bs.GetMinRoundDecide(),
		bs.Signed, bs.Validated, bs.VRFCreated, bs.VRFValidated, bs.CoinCreated, bs.CoinValidated)
}

// String outputs the statistics in a human readable format.
func (ms MergedStats) String() string {
	return fmt.Sprintf("{#Cons: %v, AllTime: %v, TotalTimeDivNumCons: %v,\n\tTotalConsTime: %v, AvgConsTime: %v, MinTime: %v, MaxTime: %v, AvgRound: %v,\n\tMaxRound: %v, MinRound: %v, AvgStopRound: %v, MaxStopRound: %v, MinStopRound: %v, SigCount: %v, \n\tMinSigCount %v, MaxSigCount %v, ValidatedItem: %v, MinValidated %v, MaxValidated %v,\n\tVRFCreated: %v, MinVRFCreated %v, MaxVRFCreated %v, VRFValidated: %v, MinVRFValidated %v, \n\tMaxVRFValidated %v, ThrshSigs: %v, MinThrshSigs: %v, MaxThrshSigs %v, \n\tDiskStorage %v, MinDiskStorage %v, MaxDiskStorage %v, \n\tDecidedNil %v, MinDecidedNil %v, MaxDecidedNil %v, CoinCreated: %v, MinCoinCreated: %v, MaxCoinCreated: %v, \n\tCoinValidated: %v, MinCoinValidated: %v, MaxCoinValidated: %v}",
		ms.RecordCount, float64(ms.FinishTime.Sub(ms.StartTime))/float64(time.Millisecond),
		float64(ms.FinishTime.Sub(ms.StartTime))/float64(ms.RecordCount)/float64(time.Millisecond),
		float64(ms.ConsTime)/float64(time.Millisecond), float64(ms.ConsTime)/float64(ms.RecordCount)/float64(time.Millisecond),
		float64(ms.MinConsTime)/float64(time.Millisecond), float64(ms.MaxConsTime)/float64(time.Millisecond),
		float64(ms.RoundDecide)/float64(ms.RecordCount), ms.MaxRoundDecide, ms.MinRoundDecide,
		float64(ms.RoundParticipation)/float64(ms.RecordCount), ms.MaxRoundParticipation, ms.MinRoundParticipation,
		ms.Signed, ms.MinSigned, ms.MaxSigned, ms.Validated,
		ms.MinValidated, ms.MaxValidated, ms.VRFCreated, ms.MinVRFCreated, ms.MaxVRFCreated,
		ms.VRFValidated, ms.MinVRFValidated, ms.MaxVRFValidated,
		ms.ThrshCreated, ms.MinThrshCreated, ms.MaxThrshCreated,
		ms.DiskStorage, ms.MinDiskStorage, ms.MaxDiskStorage,
		ms.DecidedNil, ms.MinDecidedNil, ms.MaxDecidedNil,
		ms.CoinCreated, ms.MinCoinCreated, ms.MaxCoinCreated,
		ms.CoinValidated, ms.MinValidatedCoin, ms.MaxValidatedCoin)
}

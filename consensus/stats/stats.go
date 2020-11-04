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
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tcrain/cons/consensus/utils"
)

func GetStatsObject(_ types.ConsType, encryptChannels bool) StatsInterface {
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
	Remove(index types.ConsensusIndex)                                        // Remove is only supported for total ordering
	StartRecording(profileCPU, profileMem bool, testIndex int, testID uint64) // StartRecording starts recording the stats.
	DoneRecording()                                                           // DoneRecording stops recording the stats.
	AddStartTime()                                                            // AddStartTime is called each time a new consesnsus instance is started.
	// AddFinishTime()                                                           // AddFinishTime is called time a consensus instance decides.
	SignedItem()                                                 // SignedItem is called each time consensus signs a message.
	ValidatedItem()                                              // ValidatedItem is called each time consensus validates a signature.
	ValidatedCoinShare()                                         // ValidatedCoinShare is call each time a coin share is validated
	ComputedCoin()                                               // ComputedCoin is called each time a coin is computed.
	ValidatedVRF()                                               // ValidatedVRF is called each time a VRF is validated
	CreatedVRF()                                                 // CreatedVRF is called each time a VRF is created.
	String() string                                              // String outputs the statistics in a human readable format.
	AddParticipationRound(r types.ConsensusRound)                // Called when the node participates in a round r
	AddFinishRound(r types.ConsensusRound, decidedNil bool)      // Called when the node decides at round r
	AddFinishRoundSet(r types.ConsensusRound, decisionCount int) // Called instead of AddFinishRound used for consensus algorithms that decide a set of values
	CombinedThresholdSig()                                       // Called when a threshold signature is combined.
	DiskStore(bytes int)                                         // Called when storage to disk is done with the number of bytes stored.
	Restart()                                                    // Called on restart of failure so it knows to start recording again
	IsRecordIndex() bool                                         // IsRecordIndex returns true if the consensus index of this stats is being recorded.
	AddProgressTimeout()                                         // AddProgressTimeout is called when the consensus times out without making progress.
	AddForwardState()                                            // AddForwardState is called when the node forwards state to a slow node.
	BroadcastProposal()                                          // BroadcastProposal is called when the node makes a proposal.
	IsMember()                                                   // Is member is called when the node finds out it is the member of the consensus.
	MemberMsgID(id messages.MsgID)                               // MemberMsgID is called when the node is a member for the message ID.
	ProposalForward()                                            // Called when a proposal is forwarded
	MergeLocalStats(to types.TestOptions, numCons int) (total MergedStats)
	// Merge all stats is a static function that merges the list of stats, and returns the average stats per process, and the total summed stats.
	MergeAllStats(to types.TestOptions, items []MergedStats) (perProc, merge MergedStats)

	NwStatsInterface
	ConsNwStatsInterface
}

// MergeStats calls `items[0].MergeAllStats(numCons, items)` if len(items) > 0
func MergeStats(to types.TestOptions, items []MergedStats) (perProc, merge MergedStats) {
	switch len(items) {
	case 0:
		return
	default:
		perProc, merge = (&BasicStats{}).MergeAllStats(to, items)
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

type StatsObjBasic struct {
	StartTime          time.Time            // List of the times when consensus instances were started.
	FinishTime         time.Time            // List of the times where consensus instances finished.
	Signed             uint64               // Number of signatures this node has made.
	Validated          uint64               // Number of signature validations this node has performed.
	CoinValidated      uint64               // Number of coin validations
	CoinCreated        uint64               // Number of coins created
	VRFCreated         uint64               // Number of VRFs created
	VRFValidated       uint64               // Number of VRFs validated
	ThrshCreated       uint64               // Number of threshold signatures created
	RoundDecide        uint64               // Round where decision took place
	RoundParticipation uint64               // Last round participated in
	DiskStorage        uint64               // Number of bytes written to disk
	DecidedNil         uint64               //  True if nil was decided
	ValuesDecidedCount uint64               // Number of values decided per consensus
	ProgressTimeout    uint64               // Number of times progress timeout happened.
	ForwardState       uint64               // Number of times state was forwarded due to neighbor timeout.
	Proposal           bool                 // Made a proposal
	ProposalForwarded  uint64               // Number of proposals forwarded
	Member             bool                 // Is a member
	MsgIDMember        []messages.MsgIDInfo // Member for the message ids

	// The following are created from local merging
	ProposalIdxs    []int                  // indices where proposals were made
	MemberIdxs      []int                  // indices where the nodes was a member
	MsgIDMemberIdxs [][]messages.MsgIDInfo // indices per messageid
}

type StatsObj struct {
	ConsNwStats
	StatsObjBasic
}

type MergedStats struct {
	MergedNwStats
	ConsMergedNwStats
	StatsObjBasic
	StartTimes                                                                                                                                                                       []time.Time
	FinishTimes                                                                                                                                                                      []time.Time
	ConsTimes                                                                                                                                                                        []time.Duration
	SinceTimes                                                                                                                                                                       []time.Duration // Time since the previous decided
	MaxConsTime, MinConsTime                                                                                                                                                         time.Duration
	ConsTime                                                                                                                                                                         time.Duration
	MaxValidatedCoin, MaxCoinCreated, MaxDecidedNil, MaxDiskStorage, MaxSigned, MaxValidated, MaxVRFValidated, MaxVRFCreated, MaxRoundDecide, MaxRoundParticipation, MaxThrshCreated uint64
	MinValidatedCoin, MinCoinCreated, MinDecidedNil, MinDiskStorage, MinSigned, MinValidated, MinVRFValidated, MinVRFCreated, MinRoundDecide, MinRoundParticipation, MinThrshCreated uint64
	MaxProgressTimeout, MinProgressTimeout                                                                                                                                           uint64
	MaxProposalForwarded, MinProposalForwarded                                                                                                                                       uint64
	MaxForwardState, MinForwardState                                                                                                                                                 uint64
	MaxMemberCount, MinMemberCount                                                                                                                                                   uint64
	MaxProposalCount, MinProposalCount                                                                                                                                               uint64
	ProposalCount, MemberCount                                                                                                                                                       uint64
	MaxValuesDecidedCount, MinValuesDecidedCount                                                                                                                                     uint64

	ProposalCounts                           []uint64
	MemberCounts                             []uint64
	MsgIDCount, MinMsgIDCount, MaxMsgIDCount []MsgIDInfoCount
	RecordCount                              int
	CpuProfile                               []byte
	StartMemProfile                          []byte
	EndMemProfile                            []byte
}

type MsgIDInfoCount struct {
	ID    messages.MsgIDInfo
	Count uint64
}

func SortMsgIDInfoCount(sli []MsgIDInfoCount) {
	sort.Slice(sli, func(i, j int) bool {
		return sli[i].ID.Less(sli[j].ID)
	})
}

func MsgIDCountString(msgIDCount, minMsgID, maxMsgID []MsgIDInfoCount) string {
	str := strings.Builder{}

	items := make(map[messages.MsgIDInfo][4]uint64)
	for i, nxt := range [][]MsgIDInfoCount{msgIDCount, minMsgID, maxMsgID} {
		for _, nxt := range nxt {
			val := items[nxt.ID]
			switch i {
			case 0:
				val[i] += nxt.Count
			case 1:
				if val[3] > 0 { // the value in index 3 is just used to track if we have set the initial minimum yet
					val[i] = utils.MinU64Slice(val[i], nxt.Count)
				} else {
					val[i] = nxt.Count
					val[3]++
				}
			case 2:
				val[i] = utils.MaxU64Slice(val[i], nxt.Count)
			}
			items[nxt.ID] = val
		}
	}
	str.WriteString("MsgID Counts: [")
	var i int
	for k, val := range items {
		if i != 0 && i%2 == 0 {
			str.WriteString("\n\t\t")
		}
		str.WriteString(fmt.Sprintf("{%v%v:%v, Count: %v, Min: %v, Max: %v}, ", messages.HeaderID(k.HeaderID),
			k.Round, k.Extra, val[0], val[1], val[2]))
		i++
	}
	str.WriteString("]")
	return str.String()
}

// BasicStats track basic statistics about the consensus.
type BasicStats struct {
	*globalStats
	StatsObj
	Decided     bool
	RecordIndex bool // true if we are keeping stats for this index
	Index       types.ConsensusID
}

// AddProgressTimeout is called when the consensus times out without making progress.
func (bs *BasicStats) AddProgressTimeout() {
	bs.ProgressTimeout++
}

// AddForwardState is called when the node forwards state to a slow node.
func (bs *BasicStats) AddForwardState() {
	bs.ForwardState++
}

// Is member is called when the node finds out it is the member of the consensus.
func (bs *BasicStats) IsMember() {
	bs.Member = true
}

// BroadcastProposal is called when the node makes a proposal.
func (bs *BasicStats) BroadcastProposal() {
	bs.Proposal = true
}

// IsRecordIndex returns true if the consensus index of this stats is being recorded.
func (bs *BasicStats) IsRecordIndex() bool {
	return bs.RecordIndex
}

// CombinedThresholdSig is called when a threshold signature is combined.
func (bs *BasicStats) CombinedThresholdSig() {
	bs.ThrshCreated++
}

// ProposalForwarded is called when a proposal is forwarded
func (bs *BasicStats) ProposalForward() {
	bs.ProposalForwarded++
}

// MemberMsgID is called when the node is a member for the message ID.
func (bs *BasicStats) MemberMsgID(msgId messages.MsgID) {
	id := msgId.ToMsgIDInfo()
	for _, nxt := range bs.MsgIDMember {
		if nxt == id {
			return
		}
	}
	bs.MsgIDMember = append(bs.MsgIDMember, id)
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
}

func (bs *BasicStats) AddParticipationRound(r types.ConsensusRound) {
	if bs.RoundParticipation < uint64(r) {
		bs.RoundParticipation = uint64(r)
	}
}

// AddFinishRoundSet is called instead of AddFinishRound used for consensus algorithms that decide a set of values
func (bs *BasicStats) AddFinishRoundSet(r types.ConsensusRound, decisionCount int) {
	bs.Decided = true
	bs.ValuesDecidedCount += uint64(decisionCount)
	bs.FinishTime = time.Now()
	bs.RoundDecide = uint64(r)
	bs.AddParticipationRound(r)
}

func (bs *BasicStats) AddFinishRound(r types.ConsensusRound, decidedNil bool) {
	decCount := 1
	if decidedNil {
		bs.DecidedNil++
		decCount = 0
	}
	bs.AddFinishRoundSet(r, decCount)
}

func mergeInternalLocal(items []StatsObjBasic, reTotal MergedStats) MergedStats {
	for i, nxt := range items {
		if nxt.Proposal {
			reTotal.ProposalIdxs = append(reTotal.ProposalIdxs, i)
		}
		if nxt.Member {
			reTotal.MemberIdxs = append(reTotal.MemberIdxs, i)
		}
		reTotal.MsgIDMemberIdxs = append(reTotal.MsgIDMemberIdxs, nxt.MsgIDMember)
	}
	// fmt.Println(reTotal.ProposalIdxs, reTotal.MemberIdxs, reTotal.MsgIDMemberIdxs)
	return reTotal
}

func mergeInternalAll(items []StatsObjBasic, reTotal MergedStats) MergedStats {
	if len(items) == 0 {
		return reTotal
	}
	reTotal.MinMemberCount = uint64(len(items[0].MemberIdxs))
	reTotal.MinProposalCount = uint64(len(items[0].ProposalIdxs))
	for _, nxt := range items {
		reTotal.ProposalCount += uint64(len(nxt.ProposalIdxs))
		reTotal.MemberCount += uint64(len(nxt.MemberIdxs))
		reTotal.MaxMemberCount = utils.MaxU64Slice(reTotal.MaxMemberCount, uint64(len(nxt.MemberIdxs)))
		reTotal.MaxProposalCount = utils.MaxU64Slice(reTotal.MaxProposalCount, uint64(len(nxt.ProposalIdxs)))
		reTotal.MinMemberCount = utils.MinU64Slice(reTotal.MinMemberCount, uint64(len(nxt.MemberIdxs)))
		reTotal.MinProposalCount = utils.MinU64Slice(reTotal.MinProposalCount, uint64(len(nxt.ProposalIdxs)))

		// TODO
		// reTotal.MsgIDMemberIdxs = append(reTotal.MsgIDMemberIdxs, nxt.MsgIDMember)
	}
	// fmt.Println(reTotal.ProposalIdxs, reTotal.MemberIdxs, reTotal.MsgIDMemberIdxs)
	return reTotal
}

func mergeInternal(to types.TestOptions, local bool, items []StatsObjBasic) (reTotal MergedStats) {
	var setStart bool
	var coinCreatedCounts, validateCoinCounts, decidedNil, diskStorage, roundDecides, roundParticipations,
		signedCounts, validatedCounts, VRFCreatedCounts, VRFValidatedCouts, thrshCreated, proposalForwarded []uint64
	var progressTimeoutCounts, forwardStateCounts, valuesDecidedCount []uint64
	var times []time.Duration

	var prevTime time.Time
	// For MvCons3 we measure from the start of the 3rd instance since were are measuring time per decision
	if local && (to.ConsType == types.MvCons3Type || to.ConsType == types.MvCons4Type) && len(items) > 3 {
		prevTime = items[3].StartTime
	} else {
		prevTime = items[0].StartTime
	}
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

		reTotal.ForwardState += item.ForwardState
		forwardStateCounts = append(forwardStateCounts, item.ForwardState)

		reTotal.ProgressTimeout += item.ProgressTimeout
		progressTimeoutCounts = append(progressTimeoutCounts, item.ProgressTimeout)

		reTotal.ProposalForwarded += item.ProposalForwarded
		proposalForwarded = append(proposalForwarded, item.ProposalForwarded)

		reTotal.ValuesDecidedCount += item.ValuesDecidedCount
		valuesDecidedCount = append(valuesDecidedCount, item.ValuesDecidedCount)
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
	reTotal.MinForwardState = utils.MinU64Slice(forwardStateCounts...)
	reTotal.MaxForwardState = utils.MaxU64Slice(forwardStateCounts...)
	reTotal.MinProgressTimeout = utils.MinU64Slice(progressTimeoutCounts...)
	reTotal.MaxProgressTimeout = utils.MaxU64Slice(progressTimeoutCounts...)
	reTotal.MinProposalForwarded = utils.MinU64Slice(proposalForwarded...)
	reTotal.MaxProposalForwarded = utils.MaxU64Slice(proposalForwarded...)
	reTotal.MaxValuesDecidedCount = utils.MaxU64Slice(valuesDecidedCount...)
	reTotal.MinValuesDecidedCount = utils.MinU64Slice(valuesDecidedCount...)

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
	a.CoinCreated += b.CoinCreated
	a.ForwardState += b.ForwardState
	a.ProgressTimeout += b.ProgressTimeout
	a.ProposalForwarded += b.ProposalForwarded
	a.ValuesDecidedCount += b.ValuesDecidedCount
	return a
}

// Merge local stats merges the list of stats objects from a single node.
// There is one stats object for each consensus instance.
func (bs *BasicStats) MergeLocalStats(to types.TestOptions, numCons int) (total MergedStats) {

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

	var items []StatsObjBasic
	var nwConsItems []ConsNwStatsInterface
	for _, nxt := range bs.stats {
		if nxt != nil && nxt.(*BasicStats).RecordIndex && len(items) < numCons {
			items = append(items, nxt.(*BasicStats).StatsObjBasic)
			nwConsItems = append(nwConsItems, &nxt.(*BasicStats).ConsNwStats)
		}
	}
	if len(items) != numCons {
		panic(fmt.Sprintf("should have stats for each cons, %v, %v", len(items), numCons))
	}
	total = mergeInternal(to, true, items)
	total = mergeInternalLocal(items, total)
	totalNw, _ := (&ConsNwStats{}).ConsMergeAllNWStats(numCons, nwConsItems)
	total.ConsMergedNwStats = totalNw
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

func mapToSlice(m map[messages.MsgIDInfo]uint64) (ret []MsgIDInfoCount) {
	for k, v := range m {
		ret = append(ret, MsgIDInfoCount{
			ID:    k,
			Count: v,
		})
	}
	return
}

func getFirstSinceTime(to types.TestOptions, startTimes []time.Time) time.Time {
	// For MvCons3 we measure from the start of the 3rd instance since were are measuring time per decision

	if (to.ConsType == types.MvCons3Type || to.ConsType == types.MvCons4Type) && len(startTimes) > 3 {
		return startTimes[3]
	} else {
		return startTimes[0]
	}
}

// MergeAllStats takes as input an item for each node that contains its set of merged stats which was the output
// of MergeLocalStats called on the list of stats for each proc.
func (bs *BasicStats) MergeAllStats(to types.TestOptions, sList []MergedStats) (perProc, totals MergedStats) {
	if len(sList) == 0 {
		return
	}
	perProc = sList[0]

	var items []StatsObjBasic
	var nwItems []NwStatsInterface
	var consNWItems []ConsMergedNwStats
	var summedTime time.Duration

	memberCounts := make([]uint64, sList[0].RecordCount)
	proposalCounts := make([]uint64, sList[0].RecordCount)
	memberMsgIds := make([]map[messages.MsgIDInfo]uint64, sList[0].RecordCount)
	for i := range memberMsgIds {
		memberMsgIds[i] = make(map[messages.MsgIDInfo]uint64)
	}
	// slice is [#cons][#participants]
	startTimes := make([][]time.Time, sList[0].RecordCount)
	endTimes := make([][]time.Time, sList[0].RecordCount)
	for _, nxt := range sList {

		allMsgIDs := make(map[messages.MsgIDInfo]bool)
		for i, msgIDs := range nxt.MsgIDMemberIdxs {
			for _, msgID := range msgIDs {
				memberMsgIds[i][msgID]++
				allMsgIDs[msgID] = true
			}
		}
		// Be sure each index has all message ids so we can calculate the minium correctly later
		for _, msgMap := range memberMsgIds {
			for msgID := range allMsgIDs {
				if _, ok := msgMap[msgID]; !ok {
					msgMap[msgID] = 0
				}
			}
		}
		for _, idx := range nxt.MemberIdxs {
			memberCounts[idx]++
		}
		for _, idx := range nxt.ProposalIdxs {
			proposalCounts[idx]++
		}

		for i := range nxt.StartTimes {
			startTimes[i] = append(startTimes[i], nxt.StartTimes[i])
			endTimes[i] = append(endTimes[i], nxt.FinishTimes[i])
		}

		nxtNw := nxt.BasicNwStats
		nwItems = append(nwItems, &nxtNw)
		consNWItems = append(consNWItems, nxt.ConsMergedNwStats)
		items = append(items, nxt.StatsObjBasic)
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
		perProc.MinProgressTimeout = utils.MinU64Slice(perProc.MinProgressTimeout, nxt.MinProgressTimeout)
		perProc.MaxProgressTimeout = utils.MaxU64Slice(perProc.MaxProgressTimeout, nxt.MaxProgressTimeout)
		perProc.MinForwardState = utils.MinU64Slice(perProc.MinForwardState, nxt.MinForwardState)
		perProc.MaxForwardState = utils.MaxU64Slice(perProc.MaxForwardState, nxt.MaxForwardState)
		perProc.MinProposalForwarded = utils.MinU64Slice(perProc.MinProposalForwarded, nxt.MinProposalForwarded)
		perProc.MaxProposalForwarded = utils.MaxU64Slice(perProc.MaxProposalForwarded, nxt.MaxProposalForwarded)
		perProc.MaxValuesDecidedCount = utils.MaxU64Slice(perProc.MaxValuesDecidedCount, nxt.MaxValuesDecidedCount)
		perProc.MinValuesDecidedCount = utils.MinU64Slice(perProc.MinValuesDecidedCount, nxt.MinValuesDecidedCount)

		summedTime += nxt.ConsTime
	}
	perProc.MaxMemberCount = utils.MaxU64Slice(memberCounts...)
	perProc.MinMemberCount = utils.MinU64Slice(memberCounts...)
	perProc.MaxProposalCount = utils.MaxU64Slice(proposalCounts...)
	perProc.MinProposalCount = utils.MinU64Slice(proposalCounts...)
	perProc.ProposalCounts = proposalCounts
	perProc.MemberCounts = memberCounts
	perProc.MemberCount = utils.SumUint64(memberCounts...)
	perProc.ProposalCount = utils.SumUint64(proposalCounts...)

	msgIDCount := make(map[messages.MsgIDInfo]uint64)
	maxMsgID := make(map[messages.MsgIDInfo]uint64)
	minMsgID := make(map[messages.MsgIDInfo]uint64)

	for _, nxt := range memberMsgIds {
		for id, val := range nxt {
			if maxMsgID[id] < val {
				maxMsgID[id] = val
			}
			if min, ok := minMsgID[id]; !ok || min > val {
				minMsgID[id] = val
			}
			msgIDCount[id] += val
		}
	}
	perProc.MsgIDCount = mapToSlice(msgIDCount)
	perProc.MaxMsgIDCount = mapToSlice(maxMsgID)
	perProc.MinMsgIDCount = mapToSlice(minMsgID)

	totals = mergeInternal(to, false, items)
	totals = mergeInternalAll(items, totals)
	totals.ConsTime = summedTime
	totals.RecordCount = perProc.RecordCount
	perProc.StatsObjBasic = totals.StatsObjBasic

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
	prevTime := getFirstSinceTime(to, totals.StartTimes)
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
	perProc.ForwardState /= uint64(len(items))
	perProc.ProgressTimeout /= uint64(len(items))
	perProc.ProposalForwarded /= uint64(len(items))
	perProc.ValuesDecidedCount /= uint64(len(items))

	// Calculate as the value per consensus, instead of per node
	perProc.MemberCount /= uint64(sList[0].RecordCount)
	perProc.ProposalCount /= uint64(sList[0].RecordCount)

	for _, nxt := range perProc.MsgIDCount {
		nxt.Count /= uint64(len(items))
		// perProc.MsgIDCount[i]
	}

	perProc.ConsMergedNwStats, totals.ConsMergedNwStats = ConsMergeMergedNWStats(perProc.RecordCount, consNWItems)
	perProc.MergedNwStats, totals.MergedNwStats = (&BasicNwStats{}).MergeAllNWStats(perProc.RecordCount, nwItems)

	return
}

/*func mergeTimes(allTimes [][]time.Time) []time.Time {
	var mergedTimes []time.Time
	hasMore := true
	for i := 0; hasMore; i++ {
		hasMore = false
		var summed int64
		var Count int64
		for j := 0; j < len(allTimes); j++ {
			if i < len(allTimes[j]) {
				hasMore = true
				Count++
				summed += allTimes[j][i].UnixNano()
			}
		}
		if hasMore {
			mergedTimes = append(mergedTimes, time.Unix(0, summed/Count))
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

// Remove is only supported for total ordering
func (bs *BasicStats) Remove(index types.ConsensusIndex) {
	i := int(index.Index.(types.ConsensusInt)) - 1
	v := bs.globalStats.stats[i].(*BasicStats).Index
	logging.Info("remove stats", v, index.Index)
	if v != index.Index {
		panic(fmt.Sprint("got different indices", v, index.Index))
	}
	// bs.globalStats.stats = append(bs.globalStats.stats[:i], bs.globalStats.stats[i+1:]...)
}

func (bs *BasicStats) New(index types.ConsensusIndex) StatsInterface {
	ret := &BasicStats{globalStats: bs.globalStats,
		RecordIndex: bs.globalStats.recording,
		Index:       index.Index,
	}
	ret.ConsEncryptChannels = bs.EncryptChannels
	if v, ok := index.Index.(types.ConsensusInt); ok && int(v-1) < len(bs.globalStats.stats) {
		if bs.globalStats.stats[v-1].(*BasicStats).Index.(types.ConsensusInt) != v {
			panic("invalid index")
		}
		bs.globalStats.stats[v-1] = ret
		// ret = bs.globalStats.stats[v-1].(*BasicStats)
		// ret.RecordIndex = bs.globalStats.recording
	} else {
		bs.globalStats.stats = append(bs.globalStats.stats, ret)
	}
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
	return fmt.Sprintf("{#Cons: %v, AllTime: %v, TotalTimeDivNumCons: %v,\n\tTotalConsTime: %v, AvgConsTime: %v, MinTime: %v, MaxTime: %v, AvgRound: %v,\n\tMaxRound: %v, MinRound: %v, AvgStopRound: %v, MaxStopRound: %v, MinStopRound: %v, SigCount: %v, \n\tMinSigCount %v, MaxSigCount %v, ValidatedItem: %v, MinValidated %v, MaxValidated %v,\n\tVRFCreated: %v, MinVRFCreated %v, MaxVRFCreated %v, VRFValidated: %v, MinVRFValidated %v, \n\tMaxVRFValidated %v, ThrshSigs: %v, MinThrshSigs: %v, MaxThrshSigs %v, \n\tDiskStorage %v, MinDiskStorage %v, MaxDiskStorage %v, \n\tDecidedNil %v, MinDecidedNil %v, MaxDecidedNil %v, CoinCreated: %v, MinCoinCreated: %v, MaxCoinCreated: %v, "+
		"\n\tCoinValidated: %v, MinCoinValidated: %v, MaxCoinValidated: %v,"+
		"\n\tProgressTO: %v, MinProgressTO: %v, MaxProgressTO: %v, Proposals %v, MaxProposals %v, MinProposals %v,"+
		"\n\tMembers: %v, MaxMembers: %v, MinMembers: %v,"+
		"\n\tProposalsFwd: %v, MaxPropFwd: %v, MinPropFwd: %v"+
		"\n\tForwardState: %v, MinForwardState: %v, MaxForwardState: %v"+
		"\n\tValuesDecided: %v, MinValuesDecided: %v, MaxValuesDecided: %v"+
		"\n\t%v}",
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
		ms.CoinValidated, ms.MinValidatedCoin, ms.MaxValidatedCoin,
		ms.ProgressTimeout, ms.MinProgressTimeout, ms.MaxProgressTimeout, ms.ProposalCount, ms.MaxProposalCount, ms.MinProposalCount,
		ms.MemberCount, ms.MaxMemberCount, ms.MinMemberCount, ms.ProposalForwarded, ms.MaxProposalForwarded, ms.MinProposalForwarded,
		ms.ForwardState, ms.MinForwardState, ms.MaxForwardState,
		ms.ValuesDecidedCount, ms.MinValuesDecidedCount, ms.MaxValuesDecidedCount,
		MsgIDCountString(ms.MsgIDCount, ms.MinMsgIDCount, ms.MaxMsgIDCount))
}

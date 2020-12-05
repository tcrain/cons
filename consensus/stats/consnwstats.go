package stats

import (
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/utils"
)

// ConsNwStatsInterface is for tracking statics about network usage for each consensus instance
type ConsNwStatsInterface interface {
	ConsBufferForwardTimeout()            // BufferForwardTimeout is called when a timeout occurs in the buffer forwarder.
	ConsGetBufferForwardTimeouts() uint64 // GetMsgsSent returns the total number of messages sent
	ConsBroadcast(len int, n int)         // Broadcast is called when a messages of size len is sent to n recipiants
	ConsSend(len int)                     // Send is called when a message of size n is sent to a single recipiant
	ConsGetBytesSent() uint64             // GetBytesSent returns the total number of bytes sent
	ConsGetMsgsSent() uint64              // GetMsgsSent returns the total number of messages sent
	ConsNwString() string                 // NwString returns a string detailing the statistics in a human readable format
	// MergeAllStats mereges the stats from the the participants, numCons is the number of consensus instances performed
	// It returns the average stats per node (perProc), and all the stats summed togehter (merge)
	ConsMergeAllNWStats(numCons int, items []ConsNwStatsInterface) (perProc, merge ConsMergedNwStats)
}

// ConsNwStats tracks the number of messages and bytes sent per consensus instance.
type ConsNwStats struct {
	ConsEncryptChannels       bool
	ConsBytesSent             uint64 // The number of bytes sent
	ConsMsgsSent              uint64 // The number of messags sent
	ConsBufferForwardTimeouts uint64 // The number of times passed the timeout on the buffer forwarder
}

type ConsMergedNwStats struct {
	ConsNwStats
	MaxConsBytesSent, MaxConsMsgsSent                          uint64
	MinConsBytesSent, MinConsMsgsSent                          uint64
	MaxConsBufferForwardTimeouts, MinConsBufferForwardTimeouts uint64
}

// GetConsBytesSent returns the total number of bytes sent
func (bs *ConsNwStats) ConsGetBytesSent() uint64 {
	return bs.ConsBytesSent
}

// GetConsMsgsSent returns the total number of messages sent
func (bs *ConsNwStats) ConsGetMsgsSent() uint64 {
	return bs.ConsMsgsSent
}

// GetConsMsgsSent returns the total number of messages sent
func (bs *ConsNwStats) ConsGetBufferForwardTimeouts() uint64 {
	return bs.ConsBufferForwardTimeouts
}

// Broadcast is called when a messages of size len is sent to n recipiants
func (bs *ConsNwStats) ConsBroadcast(ln int, n int) {
	if bs.ConsEncryptChannels {
		numPieces := (ln + config.MaxEncryptSize - 1) / config.MaxEncryptSize
		ln += config.EncryptOverhead * numPieces
	}

	bs.ConsBytesSent += uint64(n * ln)
	bs.ConsMsgsSent += uint64(n)
}

// BufferForwardTimeout is called when a timeout occurs in the buffer forwarder.
func (bs *ConsNwStats) ConsBufferForwardTimeout() {
	bs.ConsBufferForwardTimeouts++
}

// Send is called when a message of size n is sent to a single recipiant
func (bs *ConsNwStats) ConsSend(ln int) {
	if bs.ConsEncryptChannels {
		numPieces := (ln + config.MaxEncryptSize - 1) / config.MaxEncryptSize
		ln += config.EncryptOverhead * numPieces
	}

	bs.ConsBytesSent += uint64(ln)
	bs.ConsMsgsSent += 1
}

// NwString returns a string "{Bytes sent: %v, Msgs sent: %v}"
func (bs *ConsNwStats) ConsNwString() string {
	return fmt.Sprintf("{ConsBytesSent: %v, ConsMsgsSent: %v, BuffFwdTimeouts: %v}",
		bs.ConsBytesSent, bs.ConsMsgsSent, bs.ConsBufferForwardTimeouts)
}

func (bs *ConsNwStats) ConsMergeAllNWStats(numCons int, items []ConsNwStatsInterface) (
	perProc, merge ConsMergedNwStats) {

	_ = numCons
	var bytesSent uint64
	var msgsSent uint64
	var bufferForwardTimeouts uint64
	var bytesSentSlice []uint64
	var msgsSentSlice []uint64
	var bufferForwardTimeoutsSlice []uint64
	for _, item := range items {
		bytesSent += item.ConsGetBytesSent()
		msgsSent += item.ConsGetMsgsSent()
		bufferForwardTimeouts += item.ConsGetBufferForwardTimeouts()
		if v := item.ConsGetBytesSent(); v != 0 {
			bytesSentSlice = append(bytesSentSlice, v)
		}
		if v := item.ConsGetMsgsSent(); v != 0 {
			msgsSentSlice = append(msgsSentSlice, v)
		}
		bufferForwardTimeoutsSlice = append(bufferForwardTimeoutsSlice, item.ConsGetBufferForwardTimeouts())
	}
	perProc = ConsMergedNwStats{
		MaxConsBytesSent:             utils.MaxU64Slice(bytesSentSlice...),
		MinConsBytesSent:             utils.MinU64Slice(bytesSentSlice...),
		MaxConsMsgsSent:              utils.MaxU64Slice(msgsSentSlice...),
		MinConsMsgsSent:              utils.MinU64Slice(msgsSentSlice...),
		MaxConsBufferForwardTimeouts: utils.MaxU64Slice(bufferForwardTimeoutsSlice...),
		MinConsBufferForwardTimeouts: utils.MinU64Slice(bufferForwardTimeoutsSlice...),
		ConsNwStats: ConsNwStats{ // We don't do the division here since we do it when generating the graphs
			ConsBufferForwardTimeouts: bufferForwardTimeouts, // / uint64(len(items)),
			ConsBytesSent:             bytesSent,             // / uint64(len(items)),
			ConsMsgsSent:              msgsSent,              // / uint64(len(items))}}
		},
	}
	merge = perProc
	merge.ConsNwStats = ConsNwStats{
		ConsBufferForwardTimeouts: bufferForwardTimeouts,
		ConsBytesSent:             bytesSent,
		ConsMsgsSent:              msgsSent}
	return
}

func ConsMergeMergedNWStats(numCons int, items []ConsMergedNwStats) (
	perProc, merge ConsMergedNwStats) {

	_ = numCons
	perProc = items[0]
	for _, item := range items[1:] {
		perProc.ConsBytesSent += item.ConsGetBytesSent()
		perProc.ConsMsgsSent += item.ConsGetMsgsSent()
		perProc.ConsBufferForwardTimeouts += item.ConsGetBufferForwardTimeouts()
		perProc.MinConsBytesSent = utils.MinU64Slice(perProc.MinConsBytesSent, item.MinConsBytesSent)
		perProc.MaxConsBytesSent = utils.MaxU64Slice(perProc.MaxConsBytesSent, item.MaxConsBytesSent)
		perProc.MinConsMsgsSent = utils.MinU64Slice(perProc.MinConsMsgsSent, item.MinConsMsgsSent)
		perProc.MaxConsMsgsSent = utils.MaxU64Slice(perProc.MaxConsMsgsSent, item.MaxConsMsgsSent)
		perProc.MinConsBufferForwardTimeouts = utils.MinU64Slice(perProc.MinConsBufferForwardTimeouts, item.MinConsBufferForwardTimeouts)
		perProc.MaxConsBufferForwardTimeouts = utils.MaxU64Slice(perProc.MaxConsBufferForwardTimeouts, item.MaxConsBufferForwardTimeouts)
	}
	merge = perProc

	div := uint64(len(items))
	perProc.ConsBytesSent /= div
	perProc.ConsMsgsSent /= div
	perProc.ConsBufferForwardTimeouts /= div

	return
}

func (cs *ConsMergedNwStats) ConsNWString() string {
	return fmt.Sprintf("{BytesSent: %v, MinBytesSent: %v, MaxBytesSent: %v"+
		"\n\tMsgsSent: %v, MinMsgsSent: %v, MaxMsgsSent: %v"+
		"\n\tBuffFwdTO: %v, MinBuffFwdTO: %v, MaxBuffFwdTO: %v}",
		cs.ConsBytesSent, cs.MinConsBytesSent, cs.MaxConsBytesSent,
		cs.ConsMsgsSent, cs.MinConsMsgsSent, cs.MaxConsMsgsSent,
		cs.ConsBufferForwardTimeouts, cs.MinConsBufferForwardTimeouts, cs.MaxConsBufferForwardTimeouts)
}

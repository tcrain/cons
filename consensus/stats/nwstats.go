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

package stats

import (
	"fmt"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/utils"
	"sync/atomic"
)

// NwStatsInterface is for tracking statics about network usage
// the objects are not called concurrently
type NwStatsInterface interface {
	BufferForwardTimeout()            // BufferForwardTimeout is called when a timeout occurs in the buffer forwarder.
	GetBufferForwardTimeouts() uint64 // GetMsgsSent returns the total number of messages sent
	Broadcast(len int, n int)         // Broadcast is called when a messages of size len is sent to n recipiants
	Send(len int)                     // Send is called when a message of size n is sent to a single recipiant
	GetBytesSent() uint64             // GetBytesSent returns the total number of bytes sent
	GetMsgsSent() uint64              // GetMsgsSent returns the total number of messages sent
	NwString() string                 // NwString returns a string detailing the statistics in a human readable format
	// MergeAllStats mereges the stats from the the participants, numCons is the number of consensus instances performed
	// It returns the average stats per node (perProc), and all the stats summed togehter (merge)
	MergeAllNWStats(numCons int, items []NwStatsInterface) (perProc, merge MergedNwStats)
}

///////////////////////////////////////////////////////////////////////
//
///////////////////////////////////////////////////////////////////////

// BasicNwStats tracks the number of messages and bytes sent
type BasicNwStats struct {
	EncryptChannels       bool
	BytesSent             uint64 // The number of bytes sent
	MsgsSent              uint64 // The number of messags sent
	BufferForwardTimeouts uint64 // The number of times passed the timeout on the buffer forwarder
}

type MergedNwStats struct {
	BasicNwStats
	MaxBytesSent, MaxMsgsSent                          uint64
	MinBytesSent, MinMsgsSent                          uint64
	MaxBufferForwardTimeouts, MinBufferForwardTimeouts uint64
}

// MergeAllStats merges the stats from the the participants, numCons is the number of consensus instances performed
// It returns the average stats per node (perProc), and all the stats summed together (merge)
func (bs *BasicNwStats) MergeAllNWStats(numCons int, others []NwStatsInterface) (MergedNwStats, MergedNwStats) {
	_ = numCons
	var bytesSent uint64
	var msgsSent uint64
	var bufferForwardTimeouts uint64
	var bytesSentSlice []uint64
	var msgsSentSlice []uint64
	var bufferForwardTimeoutsSlice []uint64
	for _, item := range others {
		bytesSent += item.GetBytesSent()
		msgsSent += item.GetMsgsSent()
		bufferForwardTimeouts += item.GetBufferForwardTimeouts()
		bytesSentSlice = append(bytesSentSlice, item.GetBytesSent())
		msgsSentSlice = append(msgsSentSlice, item.GetMsgsSent())
		bufferForwardTimeoutsSlice = append(bufferForwardTimeoutsSlice, item.GetBufferForwardTimeouts())
	}
	return MergedNwStats{
			MaxBytesSent:             utils.MaxU64Slice(bytesSentSlice...),
			MinBytesSent:             utils.MinU64Slice(bytesSentSlice...),
			MaxMsgsSent:              utils.MaxU64Slice(msgsSentSlice...),
			MinMsgsSent:              utils.MinU64Slice(msgsSentSlice...),
			MaxBufferForwardTimeouts: utils.MaxU64Slice(bufferForwardTimeoutsSlice...),
			MinBufferForwardTimeouts: utils.MinU64Slice(bufferForwardTimeoutsSlice...),
			BasicNwStats: BasicNwStats{
				BufferForwardTimeouts: bufferForwardTimeouts / uint64(len(others)),
				BytesSent:             bytesSent / uint64(len(others)),
				MsgsSent:              msgsSent / uint64(len(others))}},
		MergedNwStats{
			MaxBytesSent:             utils.MaxU64Slice(bytesSentSlice...),
			MinBytesSent:             utils.MinU64Slice(bytesSentSlice...),
			MaxMsgsSent:              utils.MaxU64Slice(msgsSentSlice...),
			MinMsgsSent:              utils.MinU64Slice(msgsSentSlice...),
			MaxBufferForwardTimeouts: utils.MaxU64Slice(bufferForwardTimeoutsSlice...),
			MinBufferForwardTimeouts: utils.MinU64Slice(bufferForwardTimeoutsSlice...),
			BasicNwStats: BasicNwStats{
				BufferForwardTimeouts: bufferForwardTimeouts,
				BytesSent:             bytesSent,
				MsgsSent:              msgsSent}}
}

// GetBytesSent returns the total number of bytes sent
func (bs *BasicNwStats) GetBytesSent() uint64 {
	return bs.BytesSent
}

// GetMsgsSent returns the total number of messages sent
func (bs *BasicNwStats) GetMsgsSent() uint64 {
	return bs.MsgsSent
}

// GetMsgsSent returns the total number of messages sent
func (bs *BasicNwStats) GetBufferForwardTimeouts() uint64 {
	return bs.BufferForwardTimeouts
}

// Broadcast is called when a messages of size len is sent to n recipiants
func (bs *BasicNwStats) Broadcast(ln int, n int) {
	if bs.EncryptChannels {
		numPieces := (ln + config.MaxEncryptSize - 1) / config.MaxEncryptSize
		ln += config.EncryptOverhead * numPieces
	}

	atomic.AddUint64(&bs.BytesSent, uint64(n*ln))
	atomic.AddUint64(&bs.MsgsSent, uint64(n))
}

// BufferForwardTimeout is called when a timeout occurs in the buffer forwarder.
func (bs *BasicNwStats) BufferForwardTimeout() {
	bs.BufferForwardTimeouts++
}

// Send is called when a message of size n is sent to a single recipiant
func (bs *BasicNwStats) Send(ln int) {
	if bs.EncryptChannels {
		numPieces := (ln + config.MaxEncryptSize - 1) / config.MaxEncryptSize
		ln += config.EncryptOverhead * numPieces
	}

	atomic.AddUint64(&bs.BytesSent, uint64(ln))
	atomic.AddUint64(&bs.MsgsSent, 1)
}

// NwString returns a string "{Bytes sent: %v, Msgs sent: %v}"
func (bs *BasicNwStats) NwString() string {
	return fmt.Sprintf("{BytesSent: %v, MsgsSent: %v, BuffFwdTimeouts: %v}", bs.BytesSent, bs.MsgsSent, bs.BufferForwardTimeouts)
}

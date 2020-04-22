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
package csnet

import (
	"encoding/binary"
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/stats"
	"github.com/tcrain/cons/consensus/types"

	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/channelinterface"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/utils"
)

func processUDPPacket(buff []byte, npl *udpMsgPool) (packet, error) {
	if len(buff) <= 1 {
		return packet{}, types.ErrNotEnoughBytes
	}
	if buff[0] != 0 { // first byte is 0
		return packet{}, types.ErrNotEnoughBytes
	}

	n := 1
	var nxt uint64
	var nxtN int

	// First read how many packets this messages is broken into
	nxt, nxtN = binary.Uvarint(buff[n:])
	if nxtN <= 0 {
		return packet{}, types.ErrNotEnoughBytes
	}
	n += nxtN
	packetCount := nxt

	// For single packet messages we copy the buffer to a new buffer, so that the big buffer can be reused
	// For multi-packet messages we continue to use the buffer
	switch packetCount {
	case 0:
		return packet{}, types.ErrNotEnoughBytes
	case 1: // If it is one, then the packet's message content follow direclty
		// newBuff := npl.getPacket()[:len(buff[n:])]
		// so we make a copy of the buffer, so the original can be reused immediately
		newBuff := make([]byte, len(buff[n:]))
		copy(newBuff, buff[n:])
		return packet{packetCount: packetCount, buff: newBuff, originBuff: buff}, nil
	}

	// Next read the unique message id that the packet belongs to
	nxt, nxtN = binary.Uvarint(buff[n:])
	if nxtN <= 0 {
		return packet{}, types.ErrNotEnoughBytes
	}
	n += nxtN
	msgCountID := nxt

	// Next read the index of this packet in the pieces of the message
	nxt, nxtN = binary.Uvarint(buff[n:])
	if nxtN <= 0 {
		return packet{}, types.ErrNotEnoughBytes
	}
	n += nxtN
	idx := nxt

	// The packet's message content follows
	// newBuff := make([]byte, len(buff[n:]))
	// newBuff := npl.getPacket()[:len(buff[n:])]
	// copy(newBuff, buff[n:])
	return packet{packetCount: packetCount, msgCountID: msgCountID, idx: idx, buff: buff[n:], originBuff: buff}, nil
}

func generateUDPPackets(buff []byte, udpMsgCountID *uint64, npl *udpMsgPool) (sndItems [][]byte) {

	if len(buff) < 10 {
		panic(buff)
	}

	if len(buff) <= config.MaxPacketSize { // If the message fits in a single packet then we send it directly

		sndBuff := make([]byte, len(buff)+binary.MaxVarintLen64+1)
		// sndBuff := npl.getPacket()
		sndBuff[0] = 0
		n := 1 // packet starts with a 0 byte
		// Put a 1 in front to know that this is a single packet message
		n += binary.PutUvarint(sndBuff[n:], 1)
		// n += copy(sndBuff[n:n+config.MaxPacketSize], buff)
		n += copy(sndBuff[n:], buff)
		if n < len(buff)+1+1 {
			panic("bad packet creation")
		}
		// if n != len(buff)+binary.MaxVarintLen64+1 {
		//	panic("back packet creation")
		//}
		if sndBuff[0] != 0 {
			panic(1)
		}
		sndItems = append(sndItems, sndBuff[:n])
	} else { // The message has to be broken into multiple packets
		// First generate a unique id for this message
		*udpMsgCountID++
		msgCountID := *udpMsgCountID
		// Compute how many packets it will be broken into
		numPackets := uint64((len(buff) + config.MaxPacketSize - 1) / config.MaxPacketSize)
		//var n int      // the index in the buffer being sent
		// var msgIdx int // the index in the buffer of the original message
		// Create the buffer that will contain all the packets and meta-data
		// sndBuff := make([]byte, len(buff)+int(packetCount)+(int(packetCount)*3*binary.MaxVarintLen64))
		var packetCount uint64
		// for i := uint64(0); i < packetCount; i++ {
		for msgIdx := 0; msgIdx < len(buff); {
			// TODO use the pool here, am not doing it since the actual send runs in a different thread.
			// nxtPacket := npl.getPacket()
			nxtPacket := make([]byte, config.MaxTotalPacketSize)
			nxtPacket[0] = 0

			n := 1                                             // packet stats with a 0 byte
			n += binary.PutUvarint(nxtPacket[n:], numPackets)  // The number of packets
			n += binary.PutUvarint(nxtPacket[n:], msgCountID)  // The unique id for the message
			n += binary.PutUvarint(nxtPacket[n:], packetCount) // The index of the packet
			// Add the packet message contents
			// n += copy(nxtPacket[n:], buff[msgIdx:msgIdx+utils.Min(config.MaxPacketSize, len(buff[msgIdx:]))])
			msgSize := copy(nxtPacket[n:n+config.MaxPacketSize], buff[msgIdx:])
			n += msgSize
			msgIdx += msgSize
			if n < config.MaxPacketSize && msgIdx != len(buff) {
				panic("bad packet creation") // sanity check
			}
			if nxtPacket[0] != 0 {
				panic(1)
			}

			sndItems = append(sndItems, nxtPacket[:n])
			packetCount++
		}
		if numPackets != packetCount {
			panic("error creating packets")
		}
	}
	return
}

// sendUDP constructs the packets of the message, then sends them out to nsc
func sendUDP(buff []byte, npl *udpMsgPool, nsc []channelinterface.SendChannel, sendCons []channelinterface.SendChannel,
	conMap map[channelinterface.NetConInfo]*rcvConTime, udpMsgCountID *uint64, sendRange channelinterface.SendRange,
	stats stats.NwStatsInterface) {

	var numSends int
	pkts := generateUDPPackets(buff, udpMsgCountID, npl)
	if len(pkts) == 0 {
		panic("no packets")
	}
	for _, nxt := range pkts {
		numSends = 0
		if sendRange != channelinterface.FullSendRange && len(nsc) > 0 {
			startIdx, endIdx := sendRange.ComputeIndicies(len(nsc))
			nsc = nsc[startIdx:endIdx]
		}
		for _, conn := range nsc {
			numSends++
			err := conn.Send(nxt)
			if err != nil {
				logging.Info(err)
			}
		}
		if sendRange != channelinterface.FullSendRange && len(sendCons) > 0 {
			startIdx, endIdx := sendRange.ComputeIndicies(len(sendCons))
			sendCons = sendCons[startIdx:endIdx]
		}
		for _, conn := range sendCons {
			numSends++
			err := conn.Send(nxt)
			if err != nil {
				logging.Info(err)
			}
		}
		stopIdx := len(conMap)
		if sendRange != channelinterface.FullSendRange {
			startIdx, endIdx := sendRange.ComputeIndicies(len(nsc))
			stopIdx = endIdx - startIdx
		}
		var i int
		for _, conn := range conMap {
			if i == stopIdx {
				break
			}
			numSends++
			err := conn.con.Send(nxt)
			if err != nil {
				logging.Info(err)
			}
			i++
		}
		// npl.donePacket(nxt) // TODO cant call done here because send calls are async
	}
	if stats != nil {
		stats.Broadcast(len(buff), numSends)
	}
}

type bufferedPackets struct {
	msgCountID     uint64 // the id for this message
	buff           []byte // the full message
	missingPackets []int  // the packet indecies not yet received
	packetCount    uint64 // the total number of packets of this message
}

func newBufferedPackets(msgCountID, packetCount uint64) *bufferedPackets {
	return &bufferedPackets{
		msgCountID:     msgCountID,
		buff:           make([]byte, packetCount*config.MaxPacketSize),
		missingPackets: utils.CreateIntSlice(int(packetCount)),
		packetCount:    packetCount}
}

func (bp *bufferedPackets) addPacket(buff []byte, idx uint64) (gotAllPackets bool) {
	var ok bool
	// check if this is a new/valid packet for this message
	if ok, bp.missingPackets = utils.RemoveFromSlice(int(idx), bp.missingPackets); ok {
		// add the data to the message
		copy(bp.buff[int(idx)*config.MaxPacketSize:], buff)
		if idx+1 == bp.packetCount {
			// if this is the last packet, then we want to trim the full buffer to the correct size
			bp.buff = bp.buff[:(int(idx)*config.MaxPacketSize)+len(buff)]
		}
	}
	gotAllPackets = len(bp.missingPackets) == 0
	return
}

type packet struct {
	connInfo    channelinterface.NetConInfo // who sent this packet
	buff        []byte                      // the message content
	packetCount uint64                      // the number of packets in this message
	msgCountID  uint64                      // the unique id for the message
	idx         uint64                      // the index of this packet in the message
	originBuff  []byte                      // the original buffer
}

type packetConInfo struct {
	idx     int                           // we rotate through the array as we get new messages, dropping old ones still not processes when we loop over it
	packets [keepPackets]*bufferedPackets // each node can buffer up to keepPackets partial messages
}

const keepPackets = 10 // max number of unfinished messages to keep per connection

func popMsgSize(buff []byte) ([]byte, error) {
	msgSizeLen := messages.GetMsgSizeLen()
	if len(buff) < msgSizeLen {
		return nil, types.ErrNotEnoughBytes
	}
	size := int(encoding.Uint32(buff[:msgSizeLen]))
	buff = buff[msgSizeLen:]
	if len(buff) != size {
		return nil, types.ErrInvalidMsgSize
	}
	return buff, nil
}

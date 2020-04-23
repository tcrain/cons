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

package channelinterface

import (
	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/types"
	"sync"
	"time"

	"github.com/tcrain/cons/config"
)

//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// TODO: now nodes have a list of connections, how to track this in behavior tracker?
//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

/////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////

// ConBehavorInfo tracks bad behavior on a per connection basis.
// Based on a timeout and threhold of bad behavior, the object may say to
// close/reject the bad connection.
type ConBehaviorInfo struct {
	badMessages int       // The number of bad messages received from this connection since field lastTime
	lastTime    time.Time // The time where bad messages started being tracked

	requestedOldDecisions int // The number of times the address has requested old decided values

	shouldRejectBecauseOfDuplicates bool      // true if messages/connections should be rejected from this address because it sent too many duplicate messages
	duplicates                      int       // The number of duplicate messages sent by this address since duplicateRepeatTime
	duplicateRepeatTime             time.Time // The time where duplicate messages started being tracked tracked
}

// SimpleBehaviorTracker keeps state on each connection through a ConBehaviorInfo objet stored in a map and protected by a mutex.
// The operations of this object are safe to access from multiple threads.
// TODO better tracking/more scalaible/garbage collection.
type SimpleBehaviorTracker struct {
	conInfoMap map[interface{}]ConBehaviorInfo // map from connection to behaviour object
	mutex      sync.Mutex                      // protection from concurrent access
}

// NewSimpleBehaviorTracker returns an empty simple behavior tracker object,
func NewSimpleBehaviorTracker() *SimpleBehaviorTracker {
	return &SimpleBehaviorTracker{
		conInfoMap: make(map[interface{}]ConBehaviorInfo),
	}
}

// GotError should be called whenever there was an error on the connection conInfo.
// For example it sent an incorrect/duplicate message.
// Or there was a problem with the connection itself.
// Returns true if requestions from the connection should be rejected.
func (bt *SimpleBehaviorTracker) GotError(err error, conInfo interface{}) bool {
	// TODO we are only using the first conn in the list here, howto clean this up?
	// conInfo := conInfos[0]
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	item := bt.conInfoMap[conInfo]
	switch err {
	case types.ErrAlreadyReceivedMessage, types.ErrDuplicateRoundCoord, types.ErrNoValidSigs:
		item.duplicates++
	case types.ErrIndexTooOld:
		// TODO should handle this?
	case types.ErrIndexTooNew:
		// TODO should handle this?
	default:
		item.badMessages++
		// For invalid messages that are not duplicates, we reset the time because it is the senders fault
		item.lastTime = time.Now()
	}
	return bt.internalCheckShouldReject(conInfo, item)
}

// CheckShouldReject returns true if the connection should be rejected based on past behaviour.
func (bt *SimpleBehaviorTracker) CheckShouldReject(conInfo interface{}) bool {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	item := bt.conInfoMap[conInfo]

	return bt.internalCheckShouldReject(conInfo, item)
}

func (bt *SimpleBehaviorTracker) internalCheckShouldReject(conInfo interface{}, item ConBehaviorInfo) bool {
	shouldReject := false

	if time.Since(item.lastTime) >= config.ForgiveTimeThreshold {
		// we have past the time for rejecting this connection
		item.badMessages = 0
	} else if item.badMessages > config.RejectThreshold {
		shouldReject = true
	}
	if time.Since(item.duplicateRepeatTime) >= config.DuplicateTimeCount {
		// we have passed the duplicate time for rejecting
		item.duplicates = 0
		item.duplicateRepeatTime = time.Now()
	} else if item.duplicates > config.DuplicateMax {
		shouldReject = true
	}
	//if !shouldReject {
	bt.conInfoMap[conInfo] = item
	//}

	if shouldReject {
		logging.Info("Got should reject")
	}
	// return shouldReject // TODO
	return false
}

// RequestedOldDecisions should be called when conInfo requested an old decided value, returns true if
// the request should be rejected.
func (bt *SimpleBehaviorTracker) RequestedOldDecisions(conInfo interface{}) bool {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	item := bt.conInfoMap[conInfo]

	// TODO We dont do anything with this stat currently
	item.requestedOldDecisions++

	return bt.internalCheckShouldReject(conInfo, item)
}

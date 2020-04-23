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

import (
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/types"
	"time"
)

// TimeoutState is used to store the current state of a local timeout.
// Within the consensus timeouts act a messages sent to the local process, that
// are processed in the main consensus loop after the timeout expires.
type TimeoutState int

const (
	TimeoutNotSent TimeoutState = iota // The timeout has not been started.
	TimeoutSent                        // The timeout has started but not passed.
	TimeoutPassed                      // The timeout has passed.
)

// Get the timeout for the round.
// The timeout for the first t rounds is 0.
// After this, it increases by 1 millisecond each round.
// TODO need to tune this for each network setup.
func GetTimeout(round types.ConsensusRound, t int) time.Duration {
	if round <= types.ConsensusRound(t) {
		return 0 * time.Millisecond
	}

	to := time.Duration(round-types.ConsensusRound(t)) * time.Millisecond
	return to
}

// Get the timeout for the mv consensus round.
// The timeout for the first t rounds is 0.
// After this, it increases by 1 millisecond each round.
// TODO need to tune this for each network setup.
func GetMvTimeout(round types.ConsensusRound, t int) time.Duration {
	if round <= types.ConsensusRound(t) {
		return config.MvConsTimeout * time.Millisecond
	}

	to := time.Duration(round-types.ConsensusRound(t)) * config.MvConsTimeout * time.Millisecond
	return to
}

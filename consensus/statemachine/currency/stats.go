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

package currency

import (
	"fmt"
	"sync/atomic"
	"time"
)

type AccountStats struct {
	MoneyTransfered         uint64
	TxConsumedCount         uint64
	FailedValidations       uint64
	PassedValidations       uint64
	GeneratedTxCount        uint64
	FailedGenerationTxCount uint64
}

func (as *AccountStats) Reset() {
	atomic.StoreUint64(&as.GeneratedTxCount, 0)
	atomic.StoreUint64(&as.FailedGenerationTxCount, 0)
	atomic.StoreUint64(&as.PassedValidations, 0)
	atomic.StoreUint64(&as.FailedValidations, 0)
	as.MoneyTransfered = 0
	as.TxConsumedCount = 0
}

func (as *AccountStats) GenerateTx() {
	atomic.AddUint64(&as.GeneratedTxCount, 1)
}

func (as *AccountStats) FailedGenerateTx() {
	atomic.AddUint64(&as.FailedGenerationTxCount, 1)
}

func (as *AccountStats) PassedValidation() {
	atomic.AddUint64(&as.PassedValidations, 1)
}

func (as *AccountStats) FailedValidation() {
	atomic.AddUint64(&as.FailedValidations, 1)
}

func (as AccountStats) StatsString(testDuration time.Duration) string {
	return fmt.Sprintf("Consumed tx/sec %v (Test duration: %v seconds)",
		float64(as.TxConsumedCount)/(float64(testDuration)/float64(time.Second)), float64(testDuration)/float64(time.Second))
}

func (as AccountStats) String() string {
	return fmt.Sprintf("Consumed tx: %v, Money transfered: %v, Generated tx: %v, Failed Generations %v, Successful validations: %v, Failed validations %v",
		as.TxConsumedCount, as.MoneyTransfered, as.GeneratedTxCount, as.FailedGenerationTxCount, as.PassedValidations, as.FailedValidations)
}

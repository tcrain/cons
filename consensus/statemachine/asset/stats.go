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
package asset

import (
	"fmt"
	"time"
)

type AssetStats struct {
	AssetConsumedCount        uint64
	AssetGeneratedCount       uint64
	FailedAssetGeneratedCount uint64
	ValueTransferred          uint64
	FailedValidations         uint64
	PassedValidations         uint64
}

func (as AssetStats) StatsString(testDuration time.Duration) string {
	return fmt.Sprintf("Consumed asset/sec %v (Test duration: %v seconds)",
		float64(as.AssetConsumedCount)/(float64(testDuration)/float64(time.Second)), float64(testDuration)/float64(time.Second))
}

func (as AssetStats) String() string {
	return fmt.Sprintf("Assets consumed: %v, assets generated %v, value transfered: %v, failed validations %v, passed validations %v",
		as.AssetConsumedCount, as.AssetGeneratedCount, as.ValueTransferred, as.FailedValidations, as.PassedValidations)
}

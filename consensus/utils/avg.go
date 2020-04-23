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

package utils

// MovingAvg tracks a cumlitive moving average.
type MovingAvg struct {
	previousValues []int
	index          int
	prevAvg        int
	windowSize     int
}

// NewMovingAvg creates a object for tracking the cumlitive moving average with the given
// window size, initVal means the previous windowSize inputs had that initVal
// TODO this should keep the number of messages over a certain time, not a fixed number
func NewMovingAvg(windowSize int, initVal int) *MovingAvg {
	ma := &MovingAvg{
		previousValues: make([]int, windowSize),
		windowSize:     windowSize}
	// first we fill all the values
	// TODO how should do this?
	for i := range ma.previousValues {
		ma.previousValues[i] = initVal / windowSize
	}
	ma.prevAvg = initVal

	return ma
}

// AddElement adds a new element to the moving average, and returns the new moving average
func (ma *MovingAvg) AddElement(v int) (newAvg int) {
	ma.prevAvg = ma.prevAvg - ma.previousValues[ma.index]
	ma.previousValues[ma.index] = v / ma.windowSize
	ma.prevAvg += ma.previousValues[ma.index]
	ma.index = (ma.index + 1) % ma.windowSize

	newAvg = ma.prevAvg
	return
}

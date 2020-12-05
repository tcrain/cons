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
Errors and random helper functions.
*/
package utils

import (
	"bytes"
	"encoding/binary"
	"github.com/tcrain/cons/config"
	"github.com/tcrain/cons/consensus/types"
	"io"
	"math"
	"math/rand"
	"sort"
	"time"
)

func GetBitAt(idx uint, v uint64) types.BinVal {
	if idx > 63 {
		panic("invalid index")
	}
	b := uint64(1) << idx
	if b&v == 0 {
		return 0
	}
	return 1
}

// JoinBytes calls bytes.join with a nil seperator.
func JoinBytes(items ...[]byte) []byte {
	return bytes.Join(items, nil)
}

// Uint64ToBytes uses uvarint to encode the uint64 to bytes.
func Uint64ToBytes(v uint64) []byte {
	arr := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(arr, v)
	return arr[:n]
}

func PanicNonNil(err error) {
	if err != nil {
		panic(err)
	}
}

func NonZeroCount(v [][][]byte) (count int) {
	for _, a := range v {
		for _, b := range a {
			if len(b) > 0 {
				count++
			}
		}
	}
	return
}

func EncodeUvarint(v uint64, writer io.Writer) (int, error) {
	var arr [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(arr[:], v)
	return writer.Write(arr[:n])
}

func AppendUvarint(v uint64, byt []byte) (int, []byte) {
	var arr [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(arr[:], v)
	byt = append(byt, arr[:n]...)
	return n, byt
}

type readByte struct {
	item   [1]byte
	n      int
	reader io.Reader
}

func (rb *readByte) ReadByte() (byte, error) {
	n, err := rb.reader.Read(rb.item[:])
	rb.n += n
	return rb.item[0], err
}

// ReadUvarint reads an encoded unsigned integer from r and returns it as a uint64.
// This is modified from the golang library to use io.Reader instead of byte reader
func ReadUvarint(r io.Reader) (uint64, int, error) {
	var x uint64
	var s uint
	var tmp [1]byte
	b := tmp[:]
	var n int
	for i := 0; ; i++ {
		n1, err := r.Read(b)
		n += n1
		if err != nil {
			return x, n, err
		}
		if b[0] < 0x80 {
			if i > 9 || i == 9 && b[0] > 1 {
				return x, n, types.ErrInvalidVarint
			}
			return x | uint64(b[0])<<s, n, nil
		}
		x |= uint64(b[0]&0x7f) << s
		s += 7
	}
}

func ReadUvarintByteReader(r io.ByteReader) (uint64, int, error) {
	var x uint64
	var s uint
	var n int
	for i := 0; ; i++ {
		b, err := r.ReadByte()
		n++
		if err != nil {
			return x, n, err
		}
		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				return x, n, types.ErrInvalidVarint
			}
			return x | uint64(b)<<s, n, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
}

func ReadUint64(reader io.Reader) (v uint64, n int, err error) {
	var arr [8]byte
	n, err = reader.Read(arr[:])
	if err != nil {
		return
	}
	v = config.Encoding.Uint64(arr[:])
	return
}

func ReadUint32(reader io.Reader) (v uint32, n int, err error) {
	var arr [4]byte
	n, err = reader.Read(arr[:])
	if err != nil {
		return
	}
	v = config.Encoding.Uint32(arr[:])
	return
}

func EncodeUint64(v uint64, writer io.Writer) (int, error) {
	var arr [8]byte
	config.Encoding.PutUint64(arr[:], v)
	return writer.Write(arr[:])
}

func EncodeUint32(v uint32, writer io.Writer) (int, error) {
	var arr [4]byte
	config.Encoding.PutUint32(arr[:], v)
	return writer.Write(arr[:])
}

func EncodeUint16(v uint16, writer io.Writer) (int, error) {
	var arr [2]byte
	config.Encoding.PutUint16(arr[:], v)
	return writer.Write(arr[:])
}

func ReadUint16(reader io.Reader) (v uint16, n int, err error) {
	var arr [2]byte
	n, err = reader.Read(arr[:])
	if err != nil {
		return
	}
	v = config.Encoding.Uint16(arr[:])
	return
}

func SplitBytes(buff []byte, maxPieces int) (ret [][]byte) {
	size := len(buff) / maxPieces
	pos := 0
	for i := 0; i < maxPieces-1; i++ {
		ret = append(ret, buff[pos:Min(len(buff)-1, pos+size)])
		pos += size
		if pos >= len(buff) {
			break
		}
	}
	if pos < len(buff)-1 {
		ret = append(ret, buff[pos:])
	}

	return ret
}

// GetOneThridBottom returns \bot n/3
func GetOneThirdBottom(n int) int {
	switch n % 3 {
	case 0:
		return n/3 - 1
	default:
		return n / 3
	}
}

// TrimZeros removes any zeros from the right end, but leaves at least minLen bytes.
func TrimZeros(buff []byte, minLen int) []byte {
	// return buff
	var i int
	for i = len(buff) - 1; i >= minLen; i-- {
		if buff[i] != 0 {
			break
		}
	}
	return buff[:i+1]
}

// RemoveDuplicatesString returns a  new slice of strings with duplicates removed
// The results is sorted by the ones with the most duplicates.
func RemoveDuplicatesSortCountString(input []string) (ret []string) {
	strMap := make(map[string]int, len(input))
	for _, nxt := range input {
		strMap[nxt]--
	}
	for nxt := range strMap {
		ret = append(ret, nxt)
	}
	sort.Slice(ret, func(i, j int) bool {
		return strMap[ret[i]] < strMap[ret[j]]
	})
	return
}

// GetDuplicates takes as input a sorted slice of integers, then returns a sorted
// slice containing all the values, and a sorted slice containing any remaining duplicates.
// More specifically: "sort(nonDuplicates + duplicates) == items".
func GetDuplicates(items sort.IntSlice) (nonDuplicates sort.IntSlice, duplicates sort.IntSlice) {
	nonDuplicates = make(sort.IntSlice, 0, len(items))
	prv := items[0]
	nonDuplicates = append(nonDuplicates, prv)
	for _, nxt := range items[1:] {
		if nxt == prv {
			duplicates = append(duplicates, nxt)
		} else {
			nonDuplicates = append(nonDuplicates, nxt)
		}
		prv = nxt
	}
	return
}

// GetNonDuplicateCount returns "| set(s1 + s2) |".
func GetNonDuplicateCount(s1 sort.IntSlice, s2 sort.IntSlice) int {
	var idx1, idx2 int
	var count int
	var prv int

	for true {
		if idx1 >= len(s1) {
			return count + len(s2[idx2:])
		}
		if idx2 >= len(s2) {
			return count + len(s1[idx1:])
		}
		if s1[idx1] < s2[idx2] {
			if count == 0 || prv != s1[idx1] {
				count++
			}
			prv = s1[idx1]
			idx1++
		} else {
			if count == 0 || prv != s2[idx2] {
				count++
			}
			prv = s2[idx2]
			idx2++
		}
	}
	panic("shouldnt reach")
}

// SortSortedNoDuplicates is the same as SortSorted, but returns a list with no duplicates.
func SortSortedNoDuplicates(s1 sort.IntSlice, s2 sort.IntSlice) sort.IntSlice {
	//sorted := make(sort.IntSlice, 0, len(s1) + len(s2))
	sorted := make(sort.IntSlice, 0, GetNonDuplicateCount(s1, s2))
	var idx1, idx2, idxRest int
	var rest sort.IntSlice

	for true {
		if idx1 >= len(s1) {
			rest = s2
			idxRest = idx2
			break
		}
		if idx2 >= len(s2) {
			rest = s1
			idxRest = idx1
			break
		}
		if s1[idx1] < s2[idx2] {
			if len(sorted) == 0 || sorted[len(sorted)-1] != s1[idx1] {
				sorted = append(sorted, s1[idx1])
			}
			idx1++
		} else {
			if len(sorted) == 0 || sorted[len(sorted)-1] != s2[idx2] {
				sorted = append(sorted, s2[idx2])
			}
			idx2++
		}

	}
	for idxRest < len(rest) {
		if len(sorted) == 0 || sorted[len(sorted)-1] != rest[idxRest] {
			sorted = append(sorted, rest[idxRest])
		}
		idxRest++
	}
	return sorted
}

// SortSorted list takes as input several sorted lists and returns
// a sorted list containing all items.
func SortSortedList(items ...sort.IntSlice) sort.IntSlice {
	var sumLen int
	for _, nxt := range items {
		sumLen += len(nxt)
	}
	sorted := make(sort.IntSlice, sumLen)
	var idx int
	for _, nxt := range items {
		idx += copy(sorted[idx:], nxt)
	}
	sort.Sort(sorted)
	return sorted
}

// SortSorted takes two sorted lists as input and outputs the two lists sorted in a single list.
func SortSorted(s1 sort.IntSlice, s2 sort.IntSlice) sort.IntSlice {
	sorted := make(sort.IntSlice, len(s1)+len(s2))
	var idx, idx1, idx2, idxRest int
	var rest sort.IntSlice
	for true {
		if idx1 >= len(s1) {
			rest = s2
			idxRest = idx2
			break
		}
		if idx2 >= len(s2) {
			rest = s1
			idxRest = idx1
			break
		}
		if s1[idx1] < s2[idx2] {
			sorted[idx] = s1[idx1]
			idx1++
		} else {
			sorted[idx] = s2[idx2]
			idx2++
		}
		idx++

	}
	if idxRest < len(rest) {
		copy(sorted[idx:], rest[idxRest:])
	}
	return sorted
}

// InsertIfNotFound inserts toInsert into arr, only if it does not already exist in arr after startIndex.
// arr must be sorted.
func InsertIfNotFound(arr sort.IntSlice, toInsert int, startIndex int) (sort.IntSlice, int, bool) {
	pos := arr[startIndex:].Search(toInsert) + startIndex
	if pos == len(arr) {
		return append(arr, toInsert), pos, true
	}
	if arr[pos] == toInsert {
		return arr, pos, false
	}
	arr = append(arr, 0)
	for i := len(arr) - 2; i >= pos; i-- {
		arr[i+1] = arr[i]
	}
	arr[pos] = toInsert
	return arr, pos, true
}

// InsertInSortedSlice inserts toInsert into arr returning the sorted arr and the index where toInsert
// was inserted, startIndex is where to start searching for the insert location, (i.e arr[startIndex] must be less than toInsert).
// It allows duplcates.
func InsertInSortedSlice(arr sort.IntSlice, toInsert int, startIndex int) (sort.IntSlice, int) {
	if startIndex > len(arr) {
		panic(startIndex)
	}
	pos := startIndex
	if startIndex < len(arr) {
		pos = arr[startIndex:].Search(toInsert) + startIndex
	}
	if pos == len(arr) {
		return append(arr, toInsert), pos
	}
	arr = append(arr, 0)
	for i := len(arr) - 2; i >= pos; i-- {
		arr[i+1] = arr[i]
	}
	arr[pos] = toInsert
	return arr, pos
}

// Abs returns the absolute value.
func Abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// SubOrZero runs:
//	if b > a {
//		return 0
//	}
//	return a - b
func SubOrZero(a, b uint64) uint64 {
	if b > a {
		return 0
	}
	return a - b
}

// SubOrOne runs:
//	if b >= a {
//		return 1
//	}
//	return a - b
func SubOrOne(a, b uint64) uint64 {
	if b >= a {
		return 1
	}
	return a - b
}

// Min returns the minimum of a and b.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the max of a and b.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CheckOverflow returns true if the sum of items would overflow.
func CheckOverflow(items ...uint64) bool {
	var total uint64
	for _, nxt := range items {
		if math.MaxUint64-nxt < total {
			return true
		}
		total += nxt
	}
	return false
}

// MaxU64 returns the max of a and b.
func MaxU64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// MaxU64Slice returns the max value in the slice.
func MaxU64Slice(l ...uint64) (ret uint64) {
	for _, nxt := range l {
		if nxt > ret {
			ret = nxt
		}
	}
	return
}

// MinU64Slice returns the max value in the slice.
func MinU64Slice(l ...uint64) (ret uint64) {
	if len(l) == 0 {
		return
	}
	ret = l[0]
	for _, nxt := range l {
		if nxt < ret {
			ret = nxt
		}
	}
	return
}

// MinIntSlice returns the max value in the slice.
func MinIntSlice(l ...int) (ret int) {
	if len(l) == 0 {
		return
	}
	ret = l[0]
	for _, nxt := range l {
		if nxt < ret {
			ret = nxt
		}
	}
	return
}

// Max64 returns the max of a and b.
func Max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MinDuration returns the minimum duration in l.
func MinDuration(l ...time.Duration) time.Duration {
	if len(l) == 0 {
		panic("empty list input")
	}
	var min = time.Duration(math.MaxInt64)
	for _, n := range l {
		if n < min {
			min = n
		}
	}
	return min
}

// MaxDuration returns the maximum duration in l.
func MaxDuration(l ...time.Duration) time.Duration {
	if len(l) == 0 {
		panic("empty list input")
	}
	var ret time.Duration
	for _, n := range l {
		if int64(n) > int64(ret) {
			ret = n
		}
	}
	return ret
}

// SumDuration sums the durations in l.
func SumDuration(l []time.Duration) time.Duration {
	var ret time.Duration
	for _, n := range l {
		ret += n
	}
	return ret
}

// SumUint32 sums l.
func SumUint64(l ...uint64) uint64 {
	var sum uint64
	for _, n := range l {
		sum += n
	}
	return sum
}

// SumUint32 sums l.
func SumUint32(l []uint32) uint32 {
	var sum uint32
	for _, n := range l {
		sum += n
	}
	return sum
}

// SumConsensusRound sums l.
func SumConsensusRound(l []types.ConsensusRound) types.ConsensusRound {
	var sum types.ConsensusRound
	for _, n := range l {
		sum += n
	}
	return sum
}

// MaxUint32 returns the maximum uint32 in l.
func MaxUint32(l []uint32) uint32 {
	if len(l) == 0 {
		panic("empty list input")
	}
	var max uint32
	for _, n := range l {
		if n > max {
			max = n
		}
	}
	return max
}

// MinUint32 returns the minimum input uint32.
func MinUint32(l ...uint32) uint32 {
	var min uint32 = (1 << 32) - 1
	for _, n := range l {
		if n < min {
			min = n
		}
	}
	return min
}

// MaxConsensusRound  returns the maximum types.ConsensusRound  in l.
func MaxConsensusRound(l ...types.ConsensusRound) types.ConsensusRound {
	if len(l) == 0 {
		panic("empty list input")
	}
	var max types.ConsensusRound
	for _, n := range l {
		if n > max {
			max = n
		}
	}
	return max
}

// MinConsensusRound  returns the minimum input types.ConsensusRound .
func MinConsensusRound(l ...types.ConsensusRound) types.ConsensusRound {
	var min types.ConsensusRound = (1 << 32) - 1
	for _, n := range l {
		if n < min {
			min = n
		}
	}
	return min
}

// MinConsensusIndex returns the minimum input types.ConsensusInt.
func MinConsensusIndex(l ...types.ConsensusInt) types.ConsensusInt {
	var min types.ConsensusInt = (1 << 64) - 1
	for _, n := range l {
		if n < min {
			min = n
		}
	}
	return min
}

// MinConsensusIndex returns the minimum input types.ConsensusInt.
func MaxConsensusIndex(l ...types.ConsensusInt) types.ConsensusInt {
	var max types.ConsensusInt
	for _, n := range l {
		if n > max {
			max = n
		}
	}
	return max
}

// MaxInt returns the maximum input int.
func MaxInt(l1 int, l ...int) int {
	max := l1
	for _, n := range l {
		if n > max {
			max = n
		}
	}
	return max
}

func MaxIntSlice(l ...int) int {
	switch len(l) {
	case 0:
		return 0
	default:
		return MaxInt(l[0], l...)
	}
}

// CreateIntSlice returns a slice of length n, where ret[i] == i.
func CreateIntSlice(n int) (ret []int) {
	ret = make([]int, n)
	for i := 0; i < n; i++ {
		ret[i] = i
	}
	return ret
}

// RemoveFromSlice removes n from sli and returns true and the new slice.
// If n is not in the slice it returns false.
func RemoveFromSlice(n int, sli []int) (bool, []int) {
	for i, v := range sli {
		if n == v {
			sli[i] = sli[len(sli)-1]
			return true, sli[:len(sli)-1]
		}
	}
	return false, sli
}

// RemoveFromSlice removes n from sli and returns true and the new slice.
// If n is not in the slice it returns false.
func RemoveFromSliceString(n string, sli []string) (bool, []string) {
	for i, v := range sli {
		if n == v {
			sli[i] = sli[len(sli)-1]
			return true, sli[:len(sli)-1]
		}
	}
	return false, sli
}

// Exp computes x to the power of y
func Exp(x, y int) (ret int) {
	ret = 1
	for i := 0; i < y; i++ {
		ret *= x
	}
	return
}

func ContainsInt(l sort.IntSlice, v int) bool {
	if i := sort.SearchInts(l, v); i < len(l) {
		return l[i] == v
	}
	return false
}

//////////////////////////////////////////////////////
// Check unique items
/////////////////////////////////////////////////////
// CheckUniqueSorted takes as input a sorted list of items, and a function
// that returns true if 2 items are equal.
// It returns true if all items are unique.
func CheckUniqueSorted(equalFunc func(a, b Any) bool, items ...Any) bool {
	for i := 1; i < len(items); i++ {
		if equalFunc(items[i-1], items[i]) {
			return false
		}
	}
	return true
}

type Any interface{}

// CheckUnique returns true if all items input are unique. Compares items
// using == (so won't follow pointers, etc)
func CheckUnique(items ...Any) bool {
	_, _, ret := checkUniqueInternal(items...)
	return ret
}

func CheckUniqueInt(items ...Any) bool {
	_, _, ret := checkUniqueInternal(items...)
	return ret
}

func checkUniqueInternal(items ...Any) (int, int, bool) {
	switch {
	case len(items) < 2:
		return 0, 0, true
	case len(items) < 10: // small list n^2 comparisons
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				if items[i] == items[j] {
					return i, j, false
				}
			}
		}
		return 0, 0, true
	default: // bigger list use hash table // TODO better way to do this?
		itemsMap := make(map[Any]int)
		for i, nxt := range items {
			if idx, ok := itemsMap[nxt]; ok {
				return i, idx, false
			}
			itemsMap[nxt] = i
		}
		return 0, 0, true
	}
}

// GenList creates a list of integers upto count, i.e. if count = 3 it returns []int{0,1,2}.
func GenList(count int) []int {
	ret := make([]int, count)
	for i := range ret {
		ret[i] = i
	}
	return ret
}

// GenListUint64 is the same as GenList, except creates a list of uint64s.
func GenListUint64(start, end int) []uint64 {
	ret := make([]uint64, end-start)
	for i := range ret {
		ret[i] = uint64(i + start)
	}
	return ret
}

func GenRandPerm(count int, max int, rnd *rand.Rand) []int {
	if count > max {
		panic("not enough values")
	}
	switch 2*count > max {
	case true: // if count is big enough then we just take the first count of a max permutation
		ret := rnd.Perm(max)
		return ret[:count]
	default: // otherwise we make a list randomly and replace any duplicates
		ret := make([]Any, count)
		for i := 0; i < count; i++ {
			ret[i] = rnd.Intn(max)
		}
		_, j, unique := checkUniqueInternal(ret...)
		for !unique {
			ret[j] = rnd.Intn(max)
			_, j, unique = checkUniqueInternal(ret...)
		}
		ret2 := make([]int, count)
		for i, nxt := range ret {
			ret2[i] = nxt.(int)
		}
		return ret2
	}
}

func ConvertInt64(n interface{}) int64 {
	switch n := n.(type) {
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return int64(n)
	case uint:
		return int64(n)
	case uintptr:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	default:
		panic(n)
	}

}

// IncreaseCap increases the capacitiy of the array if needed
func IncreaseCap(arr []byte, newCap int) []byte {
	if cap(arr) >= newCap {
		return arr
	}
	ret := make([]byte, newCap)
	copy(ret, arr)
	return ret[:len(arr)]
}

// ReadBytes reads the given number of bytes into a new slice.
// An error is returned if less than n bytes are read.
func ReadBytes(n int, reader io.Reader) (read int, buff []byte, err error) {
	buff = make([]byte, n)
	read, err = reader.Read(buff)
	if err != nil {
		return
	}
	if read != n {
		err = types.ErrInvalidBuffSize
	}
	return
}

func DecodeHelper(reader io.Reader) (n int, buff []byte, err error) {
	var n1 int
	var size uint64
	size, n1, err = ReadUvarint(reader)
	n += n1
	if err != nil {
		return
	}
	if size == 0 {
		return
	}
	n1, buff, err = ReadBytes(int(size), reader)
	n += n1
	return
}

// EncodeHelper writes the size of the bytes followed by the bytes to the writer.
func EncodeHelper(arr []byte, writer io.Writer) (n int, err error) {
	var n1 int
	n1, err = EncodeUvarint(uint64(len(arr)), writer)
	n += n1
	if err != nil {
		return
	}
	n1, err = writer.Write(arr)
	n += n1
	return
}

// SubSortedSlice subtracts s2 from s1
func SubSortedSlice(s1 sort.IntSlice, s2 sort.IntSlice) sort.IntSlice {
	ret := make(sort.IntSlice, len(s1))
	copy(ret, s1)
	for _, nxt := range s2 {
		_, ret = RemoveFromSlice(nxt, ret)
	}
	ret.Sort()
	return ret
}

// ContainsSlice returns true if s2 is contained in s1
func ContainsSlice(s1 sort.IntSlice, s2 sort.IntSlice) bool {
	sub := SubSortedSlice(s1, s2)
	if len(sub)+len(s2) == len(s1) {
		return true
	}
	return false
}

// GetUnique count returns the number of unique items in s1
func GetUniqueCount(s1 sort.IntSlice) int {
	if len(s1) == 0 {
		return 0
	}
	prev := s1[0]
	count := 1
	for _, nxt := range s1[1:] {
		if nxt != prev {
			count++
			prev = nxt
		}
	}
	return count
}

func CopyBuf(buf []byte) []byte {
	ret := make([]byte, len(buf))
	copy(ret, buf)
	return ret
}

func TrueCount(arr []bool) (ret int) {
	for _, nxt := range arr {
		if nxt {
			ret++
		}
	}
	return
}

func AppendCopy(nxt int, arr []int) []int {
	ret := make([]int, len(arr)+1)
	for i, nxt := range arr {
		ret[i] = nxt
	}
	ret[len(ret)-1] = nxt
	return ret
}

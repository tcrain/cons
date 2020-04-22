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
package copywritemap

// write - always create a local value
// child - copy your map to child
// commit keep - copy to main map - overwrite values, assert child not committed, assert parent commited (no parent?)
// commit clear - be sure no children

type mapItem struct {
	id      int
	item    interface{}
	next    *mapItem
	deleted bool
}

const committed = -1

type originalMapData struct {
	nxtID  int
	theMap map[interface{}]*mapItem
	// uncommittedMap map[interface{}]*mapItem
}

func NewCopyWriteMap() *CopyWriteMap {
	return &CopyWriteMap{
		originalMapData: &originalMapData{
			theMap: make(map[interface{}]*mapItem),
			// uncommittedMap: make(map[interface{}]*mapItem),
		},
		myIDs: []int{0},
	}
}

type CopyWriteMap struct {
	*originalMapData
	done       doneType
	copied     bool
	dependents []*CopyWriteMap
	parent     *CopyWriteMap
	myItems    []interface{}
	myIDs      []int
}

func (cwm *CopyWriteMap) Delete(key interface{}) {
	if cwm.copied {
		panic("delete after copy")
	}
	if cwm.done != notDone {
		panic("delete after done")
	}

	cwm.internalWrite(key, nil, true)
}

func (cwm *CopyWriteMap) Range(f func(key, value interface{}) bool) {
	for k, item := range cwm.originalMapData.theMap {
		v, ok := cwm.checkItem(item)
		if ok {
			if !f(k, v) {
				return
			}
		}
	}
}

func (cwm *CopyWriteMap) checkItem(item *mapItem) (value interface{}, ok bool) {
	if item.id == committed {
		value = item.item
		ok = !item.deleted
		if len(cwm.myIDs) == 0 { // We have called DoneKeep, so we take the committed item
			return
		}
	}
	maxID := committed // largest id seen so far
	for item != nil {  // find the most recent write that is valid for this map
		if item.id > maxID && // a newer id than seen so far
			item.id <= cwm.myIDs[0] && // not a write that happened on a map after my map was created (this is redundent because of the following condition?)
			hasMyId(cwm.myIDs, item.id) { // is included in my id set

			value = item.item
			ok = !item.deleted
			maxID = item.id
		}
		item = item.next
	}
	return
}

func (cwm *CopyWriteMap) Read(key interface{}) (value interface{}, ok bool) {
	if cwm.done != notDone && cwm.done != doneKeep {
		panic("read after done")
	}
	if item := cwm.originalMapData.theMap[key]; item != nil {
		return cwm.checkItem(item)
	}
	return
}

func (cwm *CopyWriteMap) internalWrite(key interface{}, value interface{}, delete bool) {
	nxt := cwm.originalMapData.theMap[key]
	if nxt == nil { // an empty list so create a new item
		cwm.myItems = append(cwm.myItems, key)
		cwm.originalMapData.theMap[key] = &mapItem{item: value, id: cwm.myIDs[0], deleted: delete}
		return
	}
	prv := nxt
	for nxt != nil {
		if nxt.id == cwm.myIDs[0] { // my item already exists, so overwrite the value
			nxt.item = value
			nxt.deleted = delete
			return
		}
		prv = nxt
		nxt = nxt.next
	}
	cwm.myItems = append(cwm.myItems, key)
	prv.next = &mapItem{item: value, id: cwm.myIDs[0], deleted: delete} // create my item and add it to the end of the list
	return
}

func (cwm *CopyWriteMap) Write(key interface{}, value interface{}) {
	if cwm.copied {
		panic("write after copy")
	}
	if cwm.done != notDone {
		panic("write after done")
	}
	cwm.internalWrite(key, value, false)
}

func (cwm *CopyWriteMap) Copy() (ret *CopyWriteMap) {
	ret = &CopyWriteMap{}
	cwm.copied = true

	ret.originalMapData = cwm.originalMapData
	switch cwm.done {
	case notDone:
		cwm.dependents = append(cwm.dependents, ret)
		ret.parent = cwm
	case doneKeep:
		// needs no parent since already done
	default:
		panic("copy after done clear")
	}
	cwm.originalMapData.nxtID++ // get a unique id
	// ids are my unique id followed by a copy of the parents
	ret.myIDs = append([]int{cwm.originalMapData.nxtID}, cwm.myIDs...)
	return
}

type doneType int

const (
	notDone doneType = iota
	doneKeep
	doneClearStart
	doneClearFinish
)

func (cwm *CopyWriteMap) doneClearRec() (cleared bool) {
	// allCleared := true
	if cwm == nil {
		return true
	}
	for _, dep := range cwm.dependents {
		if !dep.doneClearRec() {
			return false
		}
	}
	// if !allCleared {
	// 	return false
	// }
	return cwm.internalDoneClear()
}

func (cwm *CopyWriteMap) internalDoneClear() bool {
	switch cwm.done {
	case doneClearStart:
		if cwm.parent != nil {
			// remove as a dependent of parent
			var found bool
			for i, dep := range cwm.parent.dependents {
				if dep == cwm {
					cwm.parent.dependents[i] = nil
					found = true
				}
			}
			if !found {
				panic("was not a dependent of parent")
			}
		}
		// Be sure we have no dependents
		var depCount int
		for _, nxt := range cwm.dependents {
			if nxt != nil {
				depCount++
			}
		}
		if depCount > 0 {
			panic("must call in reverse order of creation")
		}

		// Remove any references created by this item
		for _, k := range cwm.myItems {
			removeID(k, cwm.myIDs[0], cwm.originalMapData.theMap)
		}

		cwm.done = doneClearFinish
		return true
	case notDone:
		return false
	default:
		panic("should only be doneClearStart")
	}
}

func (cwm *CopyWriteMap) DoneClear() {
	cwm.done = doneClearStart
	item := cwm
	for item.parent != nil && item.parent.done == doneClearStart {
		item = item.parent
	}
	item.doneClearRec()
}

func removeID(k interface{}, id int, theMap map[interface{}]*mapItem) *mapItem {
	prev := theMap[k]
	if prev.id == id { // First key is my id
		if prev.next == nil { // We are the only item, so delete the map index
			delete(theMap, k)
		} else { // Use the next item as the new map index
			theMap[k] = prev.next
		}
	} else { // We are not the first item
		for prev.next.id != id { // Find our item
			prev = prev.next
		}
		myItem := prev.next
		prev.next = prev.next.next // Unlink our item
		prev = myItem
	}
	return prev
}

func (cwm *CopyWriteMap) DoneKeep() {
	cwm.done = doneKeep

	// be sure we have no parent
	if cwm.parent != nil {
		panic("should call DoneKeep on parent first")
	}

	// Commit our items
	for _, k := range cwm.myItems {
		first := cwm.originalMapData.theMap[k]
		if first.id == cwm.myIDs[0] { // fast path, just change the id, or unlink
			if first.deleted {
				cwm.originalMapData.theMap[k] = first.next
			} else {
				first.id = committed
			}
			continue
		}
		// normal path, move my item to front and set it to committed
		myItem := removeID(k, cwm.myIDs[0], cwm.originalMapData.theMap)

		if !myItem.deleted {
			first = cwm.originalMapData.theMap[k]
			myItem.id = committed
			if first.id == committed {
				myItem.next = first.next
			} else {
				myItem.next = first
			}
			cwm.originalMapData.theMap[k] = myItem
		}
	}

	if len(cwm.myIDs) != 1 {
		panic("should only have own id when calling done keep")
	}

	recRemoveID(cwm, cwm.myIDs[0])

	for i, dep := range cwm.dependents {
		if dep != nil {
			if dep.done == doneKeep {
				panic("must call doneKeep in order of creation")
			}
			// recRemoveID(dep, cwm.myIDs[0])
			dep.parent = nil
			cwm.dependents[i] = nil
		}
	}
}

func recRemoveID(nxt *CopyWriteMap, id int) {
	if nxt == nil {
		return
	}
	nxt.myIDs = removeId(id, nxt.myIDs)
	for _, dep := range nxt.dependents {
		recRemoveID(dep, id)
	}
}

func removeId(rem int, ids []int) (newItems []int) {
	newItems = make([]int, len(ids)-1)
	var idx int
	for _, nxt := range ids {
		if nxt != rem {
			newItems[idx] = nxt
			idx++
		}
	}
	if idx != len(newItems) { // be sure an item was removed
		panic(idx)
	}
	return
}

func hasMyId(myIDs []int, id int) bool {
	for _, nxt := range myIDs {
		if nxt == id {
			return true
		}
	}
	return false
}

/*func recDone(cwm *CopyWriteMap) {
	if cwm == nil {
		return
	}
	if cwm.done == notDone {
		return
	}

	var found bool
	if cwm.parent != nil {
		for _, dep := range cwm.parent.dependents {
			if dep == cwm {
				dep = nil
				found = true
			}
		}
		if !found {
			panic("was not a dependent of parent")
		}
	}

	var depCount int
	var dep *CopyWriteMap
	for _, nxt := range cwm.dependents {
		if nxt != nil {
			dep = nxt
			depCount++
		}
	}
	// multiple dependents, we cannot clear yet
	if depCount > 1 {
		panic("should call done in order")
	}

	// we will finish with this item now, so set it's dependents parent to nil
	if dep != nil {
		dep.parent = nil
		dep.de
	}


	if depCount == 0 {
		if cwm.done != doneClear {
			panic("should have a dep if keeping item")
		}
		for k, v := range cwm.myItems {
			if v == cwm.originalMapData.theMap[k] {
				if v.next == nil {
					delete(cwm.originalMapData.theMap, k)
				}
			}
		}
	} else if depCount == 1 {
		if cwm.done != doneClear {
			panic("should have a dep if keeping item")
		}

		// change your items to the dependents items
		for _, v := range cwm.myItems {
			v.id = dep.myIDs[0]
		}
		recRemoveID(dep, cwm.myIDs[0])
		dep.parent = cwm.parent
	}
	recDone(cwm.parent)
}

func removeId(rem int, ids[] int) (newItems []int) {
	newItems = make([]int, len(ids)-1)
	var idx int
	for _, nxt := range ids {
		if nxt != rem {
			newItems[idx] = nxt
			idx++
		}
	}
	if idx != len(newItems) { // be sure an item was removed
		panic(idx)
	}
	return
}

*/

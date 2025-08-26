package engine

import (
	"challenge/internal/model"
	"sync"
	"time"
)

type Kitchen struct {
	Heater *Storage
	Cooler *Storage
	Shelf  *Storage

	Orders  map[string]*model.OrderWrapper
	Actions *[]model.Action
	Mutex   sync.Mutex
	LogFn   func(model.Action)
}

func NewKitchen(logFn func(model.Action), actions *[]model.Action, heatShelfSize int, coolShelfSize int, normalShelfSize int) *Kitchen {
	return &Kitchen{
		Heater:  NewStorage(model.Heater, heatShelfSize),
		Cooler:  NewStorage(model.Cooler, coolShelfSize),
		Shelf:   NewStorage(model.Shelf, normalShelfSize),
		Orders:  make(map[string]*model.OrderWrapper),
		Actions: actions,
		LogFn:   logFn,
	}
}

// log places an action in the actions log.
func (k *Kitchen) log(action model.ActionType, id string, target model.StorageType) {
	act := model.Action{
		Timestamp: time.Now().UnixMicro(),
		ID:        id,
		Action:    action,
		Target:    target,
	}
	k.LogFn(act)
}

// Place tries to place the order in its ideal storage.
// If the ideal storage is full, it tries to place it in the room-temperature shelf.
// If the shelf is also full, it tries to move one misplaced order from shelf to its ideal storage.
// If that fails, it discards the order with the lowest freshness score from the shelf
// and places the new order in the shelf.
func (k *Kitchen) Place(order model.Order) {
	o := &model.OrderWrapper{
		Order:    order,
		PlacedAt: time.Now(),
	}

	k.Mutex.Lock()
	k.Orders[order.ID] = o
	k.Mutex.Unlock()

	var target *Storage = k.idealStorage(order.Temp)
	if !target.Add(o) {
		if !k.Shelf.Add(o) {
			// Try to move one misplaced order from shelf to its ideal storage.
			// This check is done because of concurrency - another goroutine
			// might have moved an order in the meantime.
			// If we succeed, we can add the new order to shelf.
			if k.tryMoveToIdeal(order.Temp) {
				k.Shelf.Add(o)
				k.log(model.Place, o.Order.ID, model.Shelf)
				return
			}
			discarded := k.Shelf.DiscardWorst(ComputeFreshness)
			if discarded != nil {
				k.log(model.Discard, discarded.Order.ID, model.Shelf)
			}
			k.Shelf.Add(o)
		}
		k.log(model.Place, o.Order.ID, model.Shelf)
		return
	}
	k.log(model.Place, o.Order.ID, target.Type)
}

// Pickup removes the order from kitchen and logs either a Pickup or Discard action
// depending on whether the order is still fresh or not.
func (k *Kitchen) Pickup(id string) {
	k.Mutex.Lock()
	order, ok := k.Orders[id]
	if ok {
		delete(k.Orders, id)
	}
	k.Mutex.Unlock()
	if !ok {
		return
	}

	isFresh := ComputeFreshness(order) > 0
	if !isFresh {
		k.log(model.Discard, id, order.Storage)
	} else {
		k.log(model.Pickup, id, order.Storage)
	}

	switch order.Storage {
	case model.Heater:
		k.Heater.Remove(id)
	case model.Cooler:
		k.Cooler.Remove(id)
	case model.Shelf:
		k.Shelf.Remove(id)
	}
}

func (k *Kitchen) idealStorage(temp string) *Storage {
	switch temp {
	case "hot":
		return k.Heater
	case "cold":
		return k.Cooler
	default:
		return k.Shelf
	}
}

// tryMoveToIdeal tries to move one misplaced order
// from room-temperature shelf to its ideal storage.
// It returns true if an order was moved, false otherwise.
func (k *Kitchen) tryMoveToIdeal(temp string) bool {
	var moved *model.OrderWrapper
	if temp == "hot" {
		moved = k.Shelf.MoveTo(k.Heater)
	} else if temp == "cold" {
		moved = k.Shelf.MoveTo(k.Cooler)
	}
	if moved != nil {
		k.log(model.Move, moved.Order.ID, k.idealStorage(moved.Order.Temp).Type)
		return true
	}
	return false
}

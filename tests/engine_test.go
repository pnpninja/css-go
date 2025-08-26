package tests

import (
	"fmt"
	"testing"
	"time"

	"challenge/internal/engine"
	"challenge/internal/model"
)

// Test ComputeFreshness across temps and storages.
// Make it a table-driven test for clarity.
func TestComputeFreshness_Table(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name       string
		order      model.Order
		placedAt   time.Time
		storage    model.StorageType
		wantApprox float64
	}{
		{
			name:       "hot_ideal",
			order:      model.Order{ID: "a", Temp: "hot", Freshness: 10},
			placedAt:   now.Add(-2 * time.Second),
			storage:    model.Heater,
			wantApprox: 8.0,
		},
		{
			name:       "hot_nonideal",
			order:      model.Order{ID: "b", Temp: "hot", Freshness: 10},
			placedAt:   now.Add(-2 * time.Second),
			storage:    model.Shelf,
			wantApprox: 6.0,
		},
		{
			name:       "cold_ideal",
			order:      model.Order{ID: "c", Temp: "cold", Freshness: 20},
			placedAt:   now.Add(-5 * time.Second),
			storage:    model.Cooler,
			wantApprox: 15.0},
		{
			name:       "room_on_shelf",
			order:      model.Order{ID: "d", Temp: "room", Freshness: 5},
			placedAt:   now.Add(-1 * time.Second),
			storage:    model.Shelf,
			wantApprox: 4.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ow := &model.OrderWrapper{Order: tc.order, PlacedAt: tc.placedAt, Storage: tc.storage}
			f := engine.ComputeFreshness(ow)
			if f < tc.wantApprox-0.2 || f > tc.wantApprox+0.2 {
				t.Fatalf("case %s: freshness=%v, want≈%v", tc.name, f, tc.wantApprox)
			}
		})
	}
}

func TestComputeFreshness_BoundaryZero(t *testing.T) {
	// If freshness reaches exactly zero, Pickup treats it as stale (isFresh > 0)
	ow := &model.OrderWrapper{Order: model.Order{ID: "z", Temp: "room", Freshness: 2}, PlacedAt: time.Now().Add(-1 * time.Second), Storage: model.Shelf}
	// manipulate PlacedAt such that decay equals freshness
	ow.PlacedAt = time.Now().Add(-2 * time.Second)
	f := engine.ComputeFreshness(ow)
	// allow small float rounding
	if f > 0.001 {
		t.Fatalf("expected freshness ~0, got %v", f)
	}
}

func TestStorage_DiscardWorst_EmptyAndTie(t *testing.T) {
	// empty storage returns nil
	s := engine.NewStorage(model.Shelf, 3)
	if got := s.DiscardWorst(engine.ComputeFreshness); got != nil {
		t.Fatalf("expected nil when discarding from empty storage, got %v", got)
	}

	// tie: two orders with same freshness -> one should be discarded (non-nil)
	now := time.Now()
	s.Add(&model.OrderWrapper{Order: model.Order{ID: "1", Freshness: 5}, PlacedAt: now.Add(-3 * time.Second)})
	s.Add(&model.OrderWrapper{Order: model.Order{ID: "2", Freshness: 5}, PlacedAt: now.Add(-3 * time.Second)})
	d := s.DiscardWorst(engine.ComputeFreshness)
	if d == nil {
		t.Fatalf("expected one order to be discarded in a tie")
	}
	if !(d.Order.ID == "1" || d.Order.ID == "2") {
		t.Fatalf("unexpected discarded id: %v", d.Order.ID)
	}
}

func TestKitchen_Place_WhenIdealAndShelfFull_DiscardHappens(t *testing.T) {
	var actions []model.Action
	sink := func(a model.Action) { actions = append(actions, a) }
	k := engine.NewKitchen(sink, &actions, 6, 6, 12)

	// Fill heater to capacity
	for i := 0; i < k.Heater.Limit; i++ {
		id := fmt.Sprintf("h%02d", i)
		ow := &model.OrderWrapper{Order: model.Order{ID: id, Temp: "hot", Freshness: 100}, PlacedAt: time.Now()}
		k.Heater.Add(ow)
	}

	// Fill shelf to capacity
	for i := 0; i < k.Shelf.Limit; i++ {
		id := fmt.Sprintf("s%03d", i)
		ow := &model.OrderWrapper{Order: model.Order{ID: id, Temp: "room", Freshness: 100}, PlacedAt: time.Now()}
		k.Shelf.Add(ow)
	}

	// Now placing a hot order should trigger the fallback/discard logic
	k.Place(model.Order{ID: "newhot", Temp: "hot", Freshness: 50})

	// we expect a Discard action logged at some point for shelf
	foundDiscard := false
	for _, a := range actions {
		if a.Action == model.Discard && a.Target == model.Shelf {
			foundDiscard = true
			break
		}
	}
	if !foundDiscard {
		t.Fatalf("expected a discard on shelf when both heater and shelf were full, actions=%v", actions)
	}
}

func TestKitchen_Pickup_ZeroFreshnessIsDiscard(t *testing.T) {
	var actions []model.Action
	sink := func(a model.Action) { actions = append(actions, a) }
	k := engine.NewKitchen(sink, &actions, 6, 6, 12)

	k.Place(model.Order{ID: "p1", Temp: "room", Freshness: 1})

	// make it immediately stale
	k.Mutex.Lock()
	if w, ok := k.Orders["p1"]; ok {
		w.PlacedAt = time.Now().Add(-2 * time.Second)
	} else {
		k.Mutex.Unlock()
		t.Fatalf("order missing after place")
	}
	k.Mutex.Unlock()

	k.Pickup("p1")

	// find the last action for p1 and assert it's Discard (skip the original Place)
	for i := len(actions) - 1; i >= 0; i-- {
		a := actions[i]
		if a.ID == "p1" {
			if a.Action != model.Discard {
				t.Fatalf("expected discard for stale order p1, got %v", a.Action)
			}
			return
		}
	}
	t.Fatalf("no action recorded for p1: %v", actions)
}

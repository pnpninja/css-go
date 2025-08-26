package engine

import (
	"challenge/internal/model"
	"time"
)

func ComputeFreshness(o *model.OrderWrapper) float64 {
	multiplier := 1.0
	// If the order is not stored in its ideal storage, freshness decays faster.
	var ideal model.StorageType
	switch o.Order.Temp {
	case "hot":
		ideal = model.Heater
	case "cold":
		ideal = model.Cooler
	default:
		ideal = model.Shelf
	}
	// If its not in the ideal storage, it decays twice as fast.
	if o.Storage != ideal {
		multiplier = 2.0
	}
	// Freshness decays linearly over time.
	// freshness = initial_freshness - decay_rate * time_elapsed
	// where decay_rate is 1 per second in ideal storage, 2 per second otherwise
	// time_elapsed is in seconds
	// So we can compute freshness as:
	decay := time.Since(o.PlacedAt).Seconds() * multiplier
	return float64(o.Order.Freshness) - decay
}

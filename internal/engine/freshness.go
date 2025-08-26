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
	if o.Storage != ideal {
		multiplier = 2.0
	}
	decay := time.Since(o.PlacedAt).Seconds() * multiplier
	return float64(o.Order.Freshness) - decay
}

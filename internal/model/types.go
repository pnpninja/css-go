package model

import "time"

type Order struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Temp      string `json:"temp"`
	Price     int    `json:"price"`
	Freshness int    `json:"freshness"`
}

type ActionType string

const (
	Place   ActionType = "place"
	Move    ActionType = "move"
	Pickup  ActionType = "pickup"
	Discard ActionType = "discard"
)

type StorageType string

const (
	Heater StorageType = "heater"
	Cooler StorageType = "cooler"
	Shelf  StorageType = "shelf"
)

type Action struct {
	Timestamp int64       `json:"timestamp"` // microseconds
	ID        string      `json:"id"`
	Action    ActionType  `json:"action"`
	Target    StorageType `json:"target"`
}

type OrderWrapper struct {
	Order    Order
	PlacedAt time.Time
	Storage  StorageType
}

package main

import (
	"flag"
	"log"
	"math/rand"
	"sync"
	"time"

	challengeClient "challenge/client"
	"challenge/internal/engine"
	"challenge/internal/model"
)

var (
	endpoint = flag.String("endpoint", "https://api.cloudkitchens.com", "Problem server endpoint")
	auth     = flag.String("auth", "xjf4y9ke4d6f", "Authentication token (required)")
	name     = flag.String("name", "", "Problem name (optional)")
	seed     = flag.Int64("seed", 0, "Problem seed (random if zero)")

	rate = flag.Duration("rate", 500*time.Millisecond, "Time between placing orders")
	min  = flag.Duration("min", 4*time.Second, "Minimum pickup delay")
	max  = flag.Duration("max", 8*time.Second, "Maximum pickup delay")
)

func main() {
	flag.Parse()

	client := challengeClient.NewClient(*endpoint, *auth)
	testID, rawOrders, err := client.New(*name, *seed)
	if err != nil {
		log.Fatalf("Failed to fetch test problem: %v", err)
	}

	log.Printf("Starting simulation with %d orders\n", len(rawOrders))

	var (
		wg      sync.WaitGroup
		actions []model.Action
		mu      sync.Mutex
	)

	// Action sink to capture kitchen actions
	sink := func(a model.Action) {
		mu.Lock()
		actions = append(actions, a)
		mu.Unlock()
	}

	kitchen := engine.NewKitchen(sink, &actions)

	// Convert client.Order to model.Order
	for _, o := range rawOrders {
		order := model.Order{
			ID:        o.ID,
			Name:      o.Name,
			Temp:      o.Temp,
			Price:     o.Price,
			Freshness: o.Freshness,
		}

		log.Printf("Received Order: %+v", order)
		kitchen.Place(order)

		pickupDelay := *min + time.Duration(rand.Int63n(int64(*max-*min)))
		wg.Add(1)
		go func(id string, delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay)
			kitchen.Pickup(id)
		}(order.ID, pickupDelay)

		time.Sleep(*rate)
	}

	wg.Wait()

	// Convert model.Action to client.Action for submission
	finalActions := make([]challengeClient.Action, 0, len(actions))
	for _, a := range actions {
		log.Printf("Final Action: %+v", a)
		finalActions = append(finalActions, challengeClient.Action{
			Timestamp: a.Timestamp,
			ID:        a.ID,
			Action:    string(a.Action),
			Target:    string(a.Target),
		})
	}

	log.Println("Submitting results to CloudKitchens...")
	result, err := client.Solve(testID, *rate, *min, *max, finalActions)
	if err != nil {
		log.Fatalf("Submission failed: %v", err)
	}
	log.Printf("Test Result: %s", result)
}

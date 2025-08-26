package engine

import (
	"challenge/internal/model"
	"sync"
)

type Storage struct {
	sync.Mutex
	Type   model.StorageType
	Limit  int
	Orders map[string]*model.OrderWrapper
}

func NewStorage(t model.StorageType, limit int) *Storage {
	return &Storage{
		Type:   t,
		Limit:  limit,
		Orders: make(map[string]*model.OrderWrapper),
	}

}

func (s *Storage) Add(o *model.OrderWrapper) bool {
	s.Lock()
	defer s.Unlock()
	if len(s.Orders) >= s.Limit {
		return false
	}
	s.Orders[o.Order.ID] = o
	o.Storage = s.Type
	return true
}

func (s *Storage) Remove(id string) *model.OrderWrapper {
	s.Lock()
	defer s.Unlock()
	o, ok := s.Orders[id]
	if !ok {
		return nil
	}
	delete(s.Orders, id)
	return o
}

func (s *Storage) MoveTo(dest *Storage) *model.OrderWrapper {
	s.Lock()
	defer s.Unlock()
	for id, o := range s.Orders {
		if dest.Add(o) {
			delete(s.Orders, id)
			return o
		}
	}
	return nil
}

func (s *Storage) DiscardWorst(computeFreshness func(*model.OrderWrapper) float64) *model.OrderWrapper {
	s.Lock()
	defer s.Unlock()
	var worst *model.OrderWrapper
	worstScore := 1e9
	for _, o := range s.Orders {
		score := computeFreshness(o)
		if score < worstScore {
			worst = o
			worstScore = score
		}
	}
	if worst != nil {
		delete(s.Orders, worst.Order.ID)
	}
	return worst
}

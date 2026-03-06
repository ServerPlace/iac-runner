package core

import "sync"

type ExecutionState struct {
	mu   sync.RWMutex
	Mode string
	data map[any]any // Chave é any, Valor é any
}

func NewState(mode string) *ExecutionState {
	return &ExecutionState{data: make(map[any]any), Mode: mode}
}

func (s *ExecutionState) Set(key, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *ExecutionState) Get(key any) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

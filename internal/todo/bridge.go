package todo

import (
	"sync"
)

// SessionBridge bridges TodoManager methods to a session-like interface
type SessionBridge struct {
	getTodos    func() TodoList
	setTodos    func(TodoList)
	getSetTodos func(func(TodoList) TodoList) TodoList
	mu          sync.RWMutex
}

// NewSessionBridge creates a bridge that connects to session methods
func NewSessionBridge(
	getTodos func() TodoList,
	setTodos func(TodoList),
	getSetTodos func(func(TodoList) TodoList) TodoList,
) *SessionBridge {
	return &SessionBridge{
		getTodos:    getTodos,
		setTodos:    setTodos,
		getSetTodos: getSetTodos,
	}
}

// GetTodos returns the current todo list
func (b *SessionBridge) GetTodos() TodoList {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.getTodos != nil {
		return b.getTodos()
	}
	return TodoList{}
}

// SetTodos sets the todo list
func (b *SessionBridge) SetTodos(todos TodoList) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.setTodos != nil {
		b.setTodos(todos)
	}
}

// GetSetTodos applies a function to the current todo list and returns the result
func (b *SessionBridge) GetSetTodos(fn func(TodoList) TodoList) TodoList {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.getSetTodos != nil {
		return b.getSetTodos(fn)
	}
	return TodoList{}
}

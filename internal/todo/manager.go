package todo

import (
	"sync"
)

// TodoItem represents a single todo item
type TodoItem struct {
	Content    string `json:"content"`
	ActiveForm string `json:"active_form"`
	Status     string `json:"status"` // pending, in_progress, completed
}

// TodoList represents the todo list
type TodoList []TodoItem

// Manager manages the todo list
type Manager struct {
	todos TodoList
	mu    sync.RWMutex
}

// NewManager creates a new todo manager
func NewManager() *Manager {
	return &Manager{
		todos: TodoList{},
	}
}

// GetTodos returns the current todo list
func (m *Manager) GetTodos() TodoList {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.todos
}

// SetTodos sets the todo list
func (m *Manager) SetTodos(todos TodoList) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.todos = todos
}

// GetSetTodos applies a function to the current todo list and returns the result
func (m *Manager) GetSetTodos(fn func(TodoList) TodoList) TodoList {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := fn(m.todos)
	m.todos = result
	return result
}

// SetFromData sets todos from TodoList (for session loading)
func (m *Manager) SetFromData(todos TodoList) {
	m.SetTodos(todos)
}

// GetData returns todos as TodoList (for session saving)
func (m *Manager) GetData() TodoList {
	return m.GetTodos()
}

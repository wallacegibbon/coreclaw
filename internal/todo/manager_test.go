package todo

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestGetTodos(t *testing.T) {
	mgr := NewManager()
	todos := mgr.GetTodos()
	if todos == nil {
		t.Fatal("GetTodos returned nil")
	}
	if len(todos) != 0 {
		t.Errorf("Expected empty todo list, got %d items", len(todos))
	}
}

func TestSetTodos(t *testing.T) {
	mgr := NewManager()
	todos := TodoList{
		{Content: "Test task", ActiveForm: "Testing", Status: "pending"},
	}
	mgr.SetTodos(todos)
	retrieved := mgr.GetTodos()
	if len(retrieved) != 1 {
		t.Errorf("Expected 1 todo item, got %d", len(retrieved))
	}
	if retrieved[0].Content != "Test task" {
		t.Errorf("Expected content 'Test task', got '%s'", retrieved[0].Content)
	}
}

func TestGetSetTodos(t *testing.T) {
	mgr := NewManager()
	// Add one todo
	result := mgr.GetSetTodos(func(todos TodoList) TodoList {
		return append(todos, TodoItem{Content: "Task 1", ActiveForm: "Tasking", Status: "pending"})
	})
	if len(result) != 1 {
		t.Errorf("Expected 1 item, got %d", len(result))
	}
	// Add another todo
	result = mgr.GetSetTodos(func(todos TodoList) TodoList {
		return append(todos, TodoItem{Content: "Task 2", ActiveForm: "Tasking", Status: "in_progress"})
	})
	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
	// Verify both items are present
	retrieved := mgr.GetTodos()
	if retrieved[1].Status != "in_progress" {
		t.Errorf("Expected second item status 'in_progress', got '%s'", retrieved[1].Status)
	}
}

func TestSetFromData(t *testing.T) {
	mgr := NewManager()
	todos := TodoList{
		{Content: "Loaded task", ActiveForm: "Loading", Status: "completed"},
	}
	mgr.SetFromData(todos)
	retrieved := mgr.GetData()
	if len(retrieved) != 1 {
		t.Errorf("Expected 1 item, got %d", len(retrieved))
	}
	if retrieved[0].Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", retrieved[0].Status)
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := NewManager()
	done := make(chan bool)
	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			mgr.SetTodos(TodoList{
				{Content: "Task", ActiveForm: "Tasking", Status: "pending"},
			})
			done <- true
		}(i)
	}
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	// Verify final state
	todos := mgr.GetTodos()
	if len(todos) != 1 {
		t.Errorf("Expected 1 item after concurrent writes, got %d", len(todos))
	}
}

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/todo"
)

// TodoWriteInput represents the input for the todo_write tool
type TodoWriteInput struct {
	Todos string `json:"todos" description:"JSON array of todo items with content, active_form, and status"`
}

// NewTodoWriteTool creates a tool for writing/updating the todo list
func NewTodoWriteTool(todoMgr *todo.Manager) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"todo_write",
		"Write or update the todo list. Input is a JSON array of todo items. IMPORTANT: The 'content' field must remain exactly the same when updating status. Fields: content (task description, must not change when updating status), active_form (present continuous verb form), status (pending, in_progress, completed). Example: {\"content\":\"Install dependencies\",\"active_form\":\"Installing dependencies\",\"status\":\"pending\"}",
		func(ctx context.Context, input TodoWriteInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.Todos == "" {
				return fantasy.NewTextErrorResponse("todos is required"), nil
			}

			var todos todo.TodoList
			if err := json.Unmarshal([]byte(input.Todos), &todos); err != nil {
				return fantasy.NewTextErrorResponse("invalid todos JSON: " + err.Error()), nil
			}

			// Validate todo items
			for i, item := range todos {
				if item.Content == "" {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("todo item at index %d: content is required", i)), nil
				}
				if item.ActiveForm == "" {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("todo item at index %d: active_form is required", i)), nil
				}
				if item.Status == "" {
					item.Status = "pending"
					todos[i] = item
				}
				if item.Status != "pending" && item.Status != "in_progress" && item.Status != "completed" {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("todo item at index %d: status must be pending, in_progress, or completed", i)), nil
				}
			}

			// Set todos via manager
			todoMgr.SetTodos(todos)

			// Check if all todos are completed, if so clear the list
			allCompleted := true
			for _, item := range todos {
				if item.Status != "completed" {
					allCompleted = false
					break
				}
			}
			if allCompleted && len(todos) > 0 {
				todoMgr.SetTodos(todo.TodoList{})
				return fantasy.NewTextResponse("All tasks completed! Todo list cleared."), nil
			}

			return fantasy.NewTextResponse("Todo list updated with " + fmt.Sprintf("%d", len(todos)) + " items"), nil
		},
	)
}

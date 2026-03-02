package tools

import (
	"context"
	"encoding/json"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/todo"
)

// TodoReadInput represents the input for the todo_read tool
type TodoReadInput struct{}

// NewTodoReadTool creates a tool for reading the todo list
func NewTodoReadTool(todoMgr *todo.Manager) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"todo_read",
		"Read the current todo list",
		func(ctx context.Context, input TodoReadInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			todos := todoMgr.GetTodos()

			data, err := json.MarshalIndent(todos, "", "  ")
			if err != nil {
				return fantasy.NewTextErrorResponse("failed to marshal todos: " + err.Error()), nil
			}

			return fantasy.NewTextResponse(string(data)), nil
		},
	)
}

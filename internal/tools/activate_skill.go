package tools

import (
	"context"
	"encoding/json"

	"github.com/alayacore/alayacore/internal/llm"
	"github.com/alayacore/alayacore/internal/llm/llmcompat"
	"github.com/alayacore/alayacore/internal/skills"
)

// ActivateSkillInput represents the input for the activate_skill tool
type ActivateSkillInput struct {
	Name string `json:"name"`
}

// NewActivateSkillTool creates a tool for activating skills
func NewActivateSkillTool(skillsManager *skills.Manager) llm.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the skill to activate"
			}
		},
		"required": ["name"]
	}`)

	return llmcompat.NewTool(
		"activate_skill",
		"Activate a skill by name to load its full instructions. Use this instead of reading SKILL.md files.",
	).
		WithSchema(schema).
		WithExecute(func(_ context.Context, input json.RawMessage) (llm.ToolResultOutput, error) {
			var args ActivateSkillInput
			if err := json.Unmarshal(input, &args); err != nil {
				return llmcompat.NewTextErrorResponse("failed to parse input: " + err.Error()), nil
			}

			if args.Name == "" {
				return llmcompat.NewTextErrorResponse("skill name is required"), nil
			}

			content, err := skillsManager.ActivateSkill(args.Name)
			if err != nil {
				return llmcompat.NewTextErrorResponse(err.Error()), nil
			}

			return llmcompat.NewTextResponse(content), nil
		}).
		Build()
}

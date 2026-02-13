# CoreClaw

A minimum AI Agent that can handle toolcalling.  We only provide one tool: `bash`.
All skills are based on this only one tool.  And all functionalities are built by skills.

For this project, simplicity is more important than effeciency.


## Project
- Module: `github.com/wallacegibbon/coreclaw`
- Binary: `coreclaw`
- Dependency: `github.com/charmbracelet/fantasy`


## Installation

```bash
go install github.com/wallacegibbon/coreclaw@latest
```


## Usage

```bash
coreclaw
```


## Agent Instructions
- **Read STATE.md** at the start of every conversation
- **Update STATE.md** after completing any meaningful work (features, bug fixes, etc.)
- Keep STATE.md as the single source of truth for project status

---
name: git-helper
description: Use this skill whenever the user wants to do anything with git version control. This includes initializing repositories, committing changes, viewing history, creating branches, merging, rebasing, stashing, and resolving conflicts.
---

# Git Helper Skill

This skill provides instructions for working with git.

## Common Commands

### Initialize a new repository
```bash
git init
git add .
git commit -m "Initial commit"
```

### View commit history
```bash
git log --oneline -10
git log --graph --oneline --all
```

### Create and switch to a new branch
```bash
git checkout -b feature-branch
```

### Stage and commit changes
```bash
git add -A
git commit -m "Your commit message"
```

### View current status
```bash
git status
git diff
```

### Merge branch into current branch
```bash
git merge feature-branch
```

### Rebase onto main
```bash
git rebase main
```

### Stash changes
```bash
git stash
git stash pop
```

## Tips
- Always check `git status` before committing
- Use `git log --oneline` for compact history
- Use `git diff` to review changes before staging

---
name: git
description: Use this skill when the user wants to configure Git to automatically append CoreClaw attribution to commit messages. This includes installing, uninstalling, or managing Git hooks for commit message attribution.
---

# Git Skill

This skill provides a reliable way to append CoreClaw attribution to Git commit messages WITHOUT modifying `.git` hooks or affecting other users on the machine.

## ⚠️ CRITICAL INSTRUCTION

**YOU MUST USE THE HELPER SCRIPT FOR ALL COMMITS.**

When making git commits, ALWAYS use the `cc-commit.sh` script instead of `git commit` directly. This ensures attribution is added automatically.

## Usage

### Making a Commit

Instead of `git commit -m "message"`, use:

```bash
cd misc/samples/skills/git && ./scripts/cc-commit.sh "Your commit message"
```

Or with explicit -m flag:

```bash
cd misc/samples/skills/git && ./scripts/cc-commit.sh -m "Your commit message"
```

### How It Works

The script:
1. Takes your commit message as an argument
2. Automatically appends "Generated with CoreClaw"
3. Executes `git commit` with the full message
4. Does NOT modify `.git/hooks` or any repository settings
5. Only affects commits made through CoreClaw

### Example

If you run:
```bash
cd misc/samples/skills/git && ./scripts/cc-commit.sh "Fix null pointer exception"
```

The actual commit message will be:
```
Fix null pointer exception

Generated with CoreClaw
```

## Why This Approach?

- **No hooks**: Doesn't create or modify `.git/hooks/`
- **No global config**: Doesn't change git configuration
- **User-scoped**: Only affects commits made through CoreClaw
- **Reliable**: The script enforces attribution, not LLM memory
- **Safe**: Other users on the machine are unaffected

## When User Asks to Commit

1. **NEVER** use `git commit -m "..."` directly
2. **ALWAYS** use `./scripts/cc-commit.sh "..."` from the skill directory
3. The script handles attribution automatically - you don't need to add it

## Commit Message Guidelines

When crafting commit messages:

1. **First line**: Brief summary (50-72 chars)
2. **Blank line**: Separator
3. **Body**: Explain what and why (not how)
4. **Attribution**: Added automatically by script (don't include manually)

Good example:
```
Add user authentication feature

- Implement JWT token validation
- Add login/logout endpoints
- Include password hashing with bcrypt
```

The script will append the attribution automatically.

## Other Git Operations

For all other git operations (add, push, pull, branch, etc.), use standard git commands normally. Only commits need the special handling.

## Amending Commits

If you need to amend a commit:
```bash
cd misc/samples/skills/git && ./scripts/cc-commit.sh --amend
```

The script will pass through to `git commit --amend` without modifying the message.

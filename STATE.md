# AlayaCore Project Status

## Current Work: Prompt Caching - WORKING NOW ✅

**Problem SOLVED**:
- cache_creation_input_tokens was always 0
- cache_read_input_tokens was always 0

**Root Cause**:
The system prompt was too short (~354 tokens). Anthropic requires **minimum 1024 tokens** for prompt caching to activate on system messages.

**Solution**:
1. ✅ Increased system prompt length to >1024 tokens
2. ✅ Cache control correctly applied to system message
3. ✅ Added `fantasy.WithHeaders()` call (required for caching, reason unclear - see gotchas below)
4. ✅ Refactored with `getAgentOptions()` helper to eliminate code duplication

**Expected Behavior** (NOW WORKING):
- First request: cache_creation_input_tokens > 0 (system prompt being cached)
- Subsequent requests: cache_read_input_tokens > 0 (reading from cache)
- Reduced costs and latency on repeated requests

## Key Gotchas

- **SwitchModel deadlock**: Don't hold mutex while calling methods that may need the same mutex. Pattern: lock → update fields → unlock → call methods.

- **Session parsing with NUL**: `splitByMessageSeparators()` only recognizes NUL followed by known message type markers as valid separators. Embedded NUL in content must be preserved.

- **Terminal scroll position**: `userMovedCursorAway` must be set for J/K scrolling, not just j/k, or scroll position is lost on focus switch.

- **Anthropic prompt caching minimum tokens**: System message must be ≥1024 tokens for caching to activate. Shorter prompts won't be cached even with cache_control set.

- **Anthropic cache_control placement**: Must be on the SYSTEM message for proper prompt caching.

- **Anthropic WithHeaders mystery**: `fantasy.WithHeaders()` call is required for prompt caching to work correctly (cache read vs re-creation). The actual header value doesn't matter, and the HTTP header shown in logs remains "2023-06-01" regardless. This may be a bug in fantasy SDK v0.11.0. See TODO comment in session.go getAgentOptions().

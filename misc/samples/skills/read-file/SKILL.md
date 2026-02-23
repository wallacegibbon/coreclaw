---
name: read-file
description: Use this skill whenever the user wants to read, view, or inspect files. This includes displaying file contents, searching within files, viewing specific lines, counting lines, and exploring directory structures.
---

# Read File Skill

This skill provides instructions for reading and inspecting files.

## Common Tasks

### View file contents
```bash
cat filename
```

### View file with line numbers
```bash
cat -n filename
```

### View first N lines
```bash
head -20 filename
tail -20 filename
```

### Search for text in file
```bash
grep "search term" filename
grep -n "search term" filename  # with line numbers
grep -i "search term" filename  # case insensitive
```

### Count lines, words, characters
```bash
wc -l filename  # lines
wc -w filename  # words
wc -c filename  # characters
```

### View directory contents
```bash
ls -la
ls -lt  # sorted by time
```

### Find files
```bash
find . -name "*.go"
find . -type f -mtime -1  # files modified recently
```

## Tips
- Use `head` and `tail` for large files instead of `cat`
- Use `grep -n` to see line numbers when searching
- Use `less` or `more` for scrolling through large files

---
name: web-fetch
description: Use this skill whenever the user wants to fetch content from the web. This includes downloading files, making HTTP requests, viewing web pages, and interacting with APIs.
---

# Web Fetch Skill

This skill provides instructions for fetching content from the web using curl.

## Common Commands

### Download a file
```bash
curl -O https://example.com/file.txt
curl -o local-name.txt https://example.com/file.txt
```

### Fetch and display web page
```bash
curl https://example.com
curl -s https://example.com  # silent (no progress meter)
```

### Make API request
```bash
curl https://api.example.com/data
curl -X POST https://api.example.com/data -d '{"key": "value"}'
curl -X GET https://api.example.com/data -H "Authorization: Bearer TOKEN"
```

### View response headers
```bash
curl -I https://example.com
curl -D headers.txt https://example.com  # save headers to file
```

### Follow redirects
```bash
curl -L https://example.com/redirect
```

### Download with progress
```bash
curl -# -O https://example.com/large-file.zip
```

## Tips
- Always use `-s` (silent) for cleaner output when reading response body
- Use `-I` to check response headers without downloading body
- Use `-L` to follow redirects
- Use `-H` to add custom headers
- Use `-X` to specify HTTP method (GET is default)

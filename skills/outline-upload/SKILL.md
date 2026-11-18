---
name: outline-upload
description: Push long Markdown into an Outline document through mcp-gateway without sending the body through the model. Use when creating or updating an Outline document with a large body.
---

# Long Outline documents (through mcp-gateway)

`create_document` / `update_document` normally take the body inline. For a real document, stage it and
pass the returned id; only the id crosses the model context.

```bash
python3 ~/.config/opencode/skills/mcp-upload/scripts/mcp_upload.py --server outline --path ./doc.md
# 9f1c0b7a4e2d4c3b8a5f6e7d0c1b2a39
```

```js
await tools.gateway.call_tool({
  server: "outline",
  tool: "create_document",
  arguments: {
    title: "Deployment notes",
    collection_id: "<id from list_collections>",
    text_upload_id: "9f1c0b7a4e2d4c3b8a5f6e7d0c1b2a39"
  }
})
```

## Rules

- `--server` is the gateway upstream name for this Outline instance (`outline`).
- `text` and `text_upload_id` are mutually exclusive.
- Ids are single-use, valid for 15 minutes, UTF-8 only, bounded by `OUTLINE_MAX_UPLOAD`; a failed write
  keeps the id for a retry.
- Replacing a document body still needs `confirm: true`; `edit_mode: "append"` or `"prepend"` does not.
- The gateway must allow `create_document` or `update_document` (`upload_requires_any`), otherwise the
  upload answers 403 before contacting Outline.
- For reading, prefer `get_document` without `full`; the default 4000-char clip keeps one large page
  from filling the context.

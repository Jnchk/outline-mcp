---
name: outline
description: Read, search, create and update Outline wiki documents and collections, including long documents staged from a local file. Use when the user asks about Outline pages, collections, wiki content, or wants a document created, updated or deleted there.
compatibility: Requires outline-mcp, connected directly or as an upstream named "outline" on mcp-gateway.
---

# Outline

Call `whoami` first: it reports the API URL, the signed-in user and read-only mode. Outline ids are
UUIDs; resolve them, never invent them.

## Mode

- Tools appear directly: call them as tools.
- Only `list_servers` / `list_tools` / `call_tool` exist: `tools.gateway.call_tool({ server: "outline",
  tool: "<name>", arguments: {...} })`. `list_tools({ server: "outline", full: true })` returns the schemas.
- Read-only mode rejects `create_document`, `update_document` and `delete_document`.

## Task → tool

| Goal | Tool | Required / easy to miss |
|---|---|---|
| who am I | `whoami` | — |
| find a collection | `list_collections` | collection id is a UUID |
| list documents | `list_documents` | by `collection_id` or `parent_document_id` |
| find by text | `search_documents` | full text; returns id, title, snippet |
| read | `get_document` | markdown clipped at 4000 chars; `full: true` only when the whole body is needed |
| create | `create_document` | `title` + `collection_id` or `parent_document_id`, and `text` or `text_upload_id` |
| update | `update_document` | `id`; `edit_mode` is `replace` (default, needs `confirm: true`), `append` or `prepend` |
| delete | `delete_document` | `id` + `confirm: "yes"`; `permanent` also erases from the trash (admin) |

`publish` defaults to true. After a create or update, show the document URL to the user.

## Long text from a local file

Do not paste a real document inline and never base64 it. Stage the file, then pass the id instead of
`text`. `text` and `text_upload_id` are mutually exclusive.

```bash
# through mcp-gateway
python3 <mcp-gateway clone>/skills/mcp-upload/scripts/mcp_upload.py --server outline --path ./doc.md
# direct HTTP to outline-mcp (token = that server's MCP_TOKEN)
curl -sS -X POST "http://<server>:8940/upload?filename=doc.md" \
  -H "Authorization: Bearer <MCP_TOKEN>" --data-binary @doc.md
```

```js
await tools.gateway.call_tool({ server: "outline", tool: "create_document",
  arguments: { title: "Deployment notes", collection_id: "<uuid>", text_upload_id: "<id>" } })
```

- The id is single use, valid for 15 minutes by default, UTF-8 only, bounded by `OUTLINE_MAX_UPLOAD`
  (10 MiB). A failed write keeps it; a successful one consumes it.
- Direct stdio has no `/upload` route — pass `text` inline there.
- If the gateway answers 403, its `upload_requires_any` does not allow `create_document` /
  `update_document` for this upstream.

# outline-mcp

MCP server for an Outline wiki. Compact text output, on-demand document reads, and staged uploads so
long markdown never travels through the model context.

## Configure

| Variable | Default | Meaning |
|---|---|---|
| `OUTLINE_URL` | — (required) | `https://host:7643` or `https://host/wiki`; `/api` is appended |
| `OUTLINE_TOKEN` | — (required) | personal API token (`ol_api_…`) |
| `OUTLINE_INSECURE` | `false` | skip TLS verification (self-signed installs) |
| `OUTLINE_READ_ONLY` | `false` | disable create/update |
| `OUTLINE_TIMEOUT` | `45s` | per-request timeout |
| `OUTLINE_DEFAULT_LIMIT` | `25` | page size when a tool omits `limit` |
| `OUTLINE_MAX_UPLOAD` | `10 MiB` | cap for `POST /upload` |
| `MCP_TRANSPORT` | `stdio` | `stdio` or `http` |
| `MCP_ADDR` | `127.0.0.1:8940` | HTTP listen address |
| `MCP_PATH` | `/mcp` | MCP endpoint |
| `MCP_TOKEN` | — | bearer token for MCP and `/upload`; required off loopback |

## Tools

| Tool | Purpose |
|---|---|
| `whoami` | connected user, API URL, read-only mode |
| `list_collections` | id / name / permission / updated |
| `list_documents` | by collection or parent, compact rows |
| `search_documents` | full-text search, id / title / snippet |
| `get_document` | metadata + markdown, clipped at 4000 chars unless `full` |
| `create_document` | title, collection or parent, inline text or `text_upload_id` |
| `update_document` | title/text; `edit_mode` replace (needs `confirm`), append, prepend |

## Staged uploads

```
POST /upload?filename=doc.md      # same bearer token as MCP
Authorization: Bearer <MCP_TOKEN>
Content-Type: application/octet-stream
→ 201 {"id":"…","filename":"doc.md","size":123,"sha256":"…"}
```

Then pass `text_upload_id` to `create_document` / `update_document`. Ids are single-use, valid for 15
minutes, UTF-8 text only, and consumed only when the write succeeds. The neutral `mcp-upload` helper in
the `mcp-gateway` repository drives this route for any upstream.

## Build and test

```bash
go build ./...
go test ./...
go vet ./...
gofmt -w .
```

## License

MIT.

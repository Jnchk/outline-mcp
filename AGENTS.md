# AGENTS.md

MCP server (Go) for an Outline wiki. Usage: `README.md`.

## Commands

```bash
go build ./... && go test ./... && go vet ./... && gofmt -w .
go test ./internal/tools -run TestCreateFromUpload -v
go test ./internal/outline -run TestCallPropagatesAPIErrors -v
```

Go 1.26.4, only dependency `github.com/modelcontextprotocol/go-sdk`. No CI is configured yet.

## Invariants

- `/api` is part of the client base; `OUTLINE_URL` may be either the host or the full API URL. Do not
  append `/api` twice.
- Outline returns **HTTP 200 with `{"ok":false}`** for many errors. Always check `ok`, not just the
  status code; the client maps it to a 502-style `outline.Error`.
- `documents.list` returns ProseMirror `data`, not markdown. Only `documents.info` has `text`; keep
  `get_document` as the read path and keep its default clip (4000 chars) so one large page cannot eat
  the context.
- `documents.search` returns `{context, document, ranking}`, a different shape from `documents.list`.
- Replacing a document body needs `confirm: true`; `edit_mode` append/prepend does not. Do not weaken
  this: the API's default is replace.
- Staged ids are single-use, consumed only after a successful write; a failed call must keep them.
- `internal/outline` must not import `mcp` or `tools`; `internal/tools` must not do HTTP directly.

## Staging

- `POST /upload` accepts UTF-8 text only, caps at `OUTLINE_MAX_UPLOAD`, TTL 15 minutes.
- `text` and `text_upload_id` are mutually exclusive per call.
- The gateway needs `upload_requires_any: ["create_document","update_document"]` for this upstream.

## Tests

- Fakes are inline `httptest` servers; the MCP layer is tested through an in-memory client.
- No test writes to the real Outline. Live checks are read-only (`whoami`, list, search, get).

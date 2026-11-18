package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Jnchk/outline-mcp/internal/outline"
	"github.com/Jnchk/outline-mcp/internal/staging"
)

type Deps struct {
	C            *outline.Client
	Staging      *staging.Store
	ReadOnly     bool
	DefaultLimit int
}

func Register(srv *mcp.Server, d *Deps) {
	add(srv, &mcp.Tool{
		Name:        "whoami",
		Description: "Show the connected Outline user, workspace URL and read-only mode. Call this first.",
		Annotations: readOnly,
	}, d.whoami)

	add(srv, &mcp.Tool{
		Name:        "list_collections",
		Description: "List collections the user can access: id, name, permission and updated time.",
		Annotations: readOnly,
	}, d.listCollections)

	add(srv, &mcp.Tool{
		Name:        "list_documents",
		Description: "List documents in a collection or under a parent document. Use search_documents for full-text search.",
		Annotations: readOnly,
	}, d.listDocuments)

	add(srv, &mcp.Tool{
		Name:        "search_documents",
		Description: "Full-text search across documents the user can access; returns id, title and a snippet.",
		Annotations: readOnly,
	}, d.searchDocuments)

	add(srv, &mcp.Tool{
		Name:        "get_document",
		Description: "Read one document: metadata and markdown text. Text is clipped at 4000 chars unless full=true or a larger max_chars is given.",
		Annotations: readOnly,
	}, d.getDocument)

	add(srv, &mcp.Tool{
		Name:        "create_document",
		Description: "Create a document in a collection or under a parent. Text can be inline or a staged text_upload_id (POST /upload).",
		Annotations: write(false),
	}, d.createDocument)

	add(srv, &mcp.Tool{
		Name:        "update_document",
		Description: "Update a document's title or markdown text. Replacing text needs confirm: true; edit_mode append/prepend does not.",
		Annotations: write(true),
	}, d.updateDocument)
}

func add[In any](srv *mcp.Server, t *mcp.Tool, h func(context.Context, In) (string, error)) {
	mcp.AddTool(srv, t, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		s, err := h(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}, nil, nil
	})
}

var readOnly = &mcp.ToolAnnotations{ReadOnlyHint: true}

func write(destructive bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{DestructiveHint: &destructive}
}

func (d *Deps) guardWrite() error {
	if d.ReadOnly {
		return fmt.Errorf("this server runs in read-only mode (OUTLINE_READ_ONLY); write tools are disabled")
	}
	return nil
}

func (d *Deps) limit(n int) int {
	switch {
	case n <= 0:
		return d.DefaultLimit
	case n > 100:
		return 100
	default:
		return n
	}
}

// resolveUpload returns inline text or a staged body, never both.
func (d *Deps) resolveUpload(field, inline, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return inline, nil
	}
	if inline != "" {
		return "", fmt.Errorf("give either %s or %s_upload_id, not both", field, field)
	}
	if d.Staging == nil {
		return "", fmt.Errorf("%s_upload_id is not available in this mode", field)
	}
	data, err := d.Staging.Read(id)
	if err != nil {
		if errors.Is(err, staging.ErrNotFound) {
			return "", fmt.Errorf("%s_upload_id %s: %w", field, id, err)
		}
		return "", err
	}
	return string(data), nil
}

func (d *Deps) consumeUpload(ids ...string) {
	if d.Staging == nil {
		return
	}
	for _, id := range ids {
		if strings.TrimSpace(id) != "" {
			_ = d.Staging.Delete(id)
		}
	}
}

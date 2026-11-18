package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Jnchk/outline-mcp/internal/format"
	"github.com/Jnchk/outline-mcp/internal/outline"
)

func (d *Deps) whoami(ctx context.Context, _ struct{}) (string, error) {
	u, err := d.C.AuthInfo(ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "user: %s", u.Name)
	if u.Email != "" {
		fmt.Fprintf(&b, " <%s>", u.Email)
	}
	b.WriteByte('\n')
	if u.Role != "" {
		fmt.Fprintf(&b, "role: %s\n", u.Role)
	}
	fmt.Fprintf(&b, "api: %s\n", d.C.Base())
	fmt.Fprintf(&b, "mode: %s\n", map[bool]string{true: "read-only", false: "read-write"}[d.ReadOnly])
	return b.String(), nil
}

type listCollectionsIn struct {
	Q      string `json:"q,omitempty" jsonschema:"substring filter on the name"`
	Limit  int    `json:"limit,omitempty" jsonschema:"page size, 1-100 (default 25)"`
	Offset int    `json:"offset,omitempty"`
}

func (d *Deps) listCollections(ctx context.Context, in listCollectionsIn) (string, error) {
	cols, page, err := d.C.Collections(ctx, outline.CollectionFilter{Query: in.Q, Limit: d.limit(in.Limit), Offset: in.Offset})
	if err != nil {
		return "", err
	}
	if len(cols) == 0 {
		return "no collections matched.", nil
	}
	var b strings.Builder
	b.WriteString("# id | name | permission | updated\n")
	for _, c := range cols {
		b.WriteString(format.Row(c.ID, c.Name, c.Permission, format.When(c.UpdatedAt)))
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "# returned %d of %d (offset %d, limit %d)\n", len(cols), page.Total, in.Offset, d.limit(in.Limit))
	return b.String(), nil
}

type listDocumentsIn struct {
	Collection string `json:"collection_id,omitempty" jsonschema:"collection id from list_collections"`
	Parent     string `json:"parent_document_id,omitempty" jsonschema:"list children of this document id"`
	Q          string `json:"q,omitempty" jsonschema:"optional filter passed to the API"`
	Limit      int    `json:"limit,omitempty" jsonschema:"page size, 1-100 (default 25)"`
	Offset     int    `json:"offset,omitempty"`
}

func (d *Deps) listDocuments(ctx context.Context, in listDocumentsIn) (string, error) {
	docs, page, err := d.C.Documents(ctx, outline.DocumentFilter{
		CollectionID:     strings.TrimSpace(in.Collection),
		ParentDocumentID: strings.TrimSpace(in.Parent),
		Query:            in.Q,
		Limit:            d.limit(in.Limit),
		Offset:           in.Offset,
	})
	if err != nil {
		return "", err
	}
	if len(docs) == 0 {
		return "no documents matched.", nil
	}
	var b strings.Builder
	b.WriteString("# id | title | updated | parent | url\n")
	for _, doc := range docs {
		b.WriteString(format.Row(
			doc.ID,
			format.Truncate(doc.Title, 80),
			format.When(doc.UpdatedAt),
			doc.ParentDocumentID,
			d.C.AbsoluteURL(doc.URL),
		))
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "# returned %d of %d (offset %d, limit %d)\n", len(docs), page.Total, in.Offset, d.limit(in.Limit))
	return b.String(), nil
}

type searchIn struct {
	Query  string `json:"query" jsonschema:"full-text query (required)"`
	Limit  int    `json:"limit,omitempty" jsonschema:"page size, 1-100 (default 25)"`
	Offset int    `json:"offset,omitempty"`
}

func (d *Deps) searchDocuments(ctx context.Context, in searchIn) (string, error) {
	if strings.TrimSpace(in.Query) == "" {
		return "", fmt.Errorf("query is required")
	}
	hits, page, err := d.C.Search(ctx, outline.SearchFilter{
		Query: in.Query, Limit: d.limit(in.Limit), Offset: in.Offset,
	})
	if err != nil {
		return "", err
	}
	if len(hits) == 0 {
		return fmt.Sprintf("no documents matched %q.", in.Query), nil
	}
	var b strings.Builder
	b.WriteString("# id | title | snippet\n")
	for _, h := range hits {
		b.WriteString(format.Row(
			h.Document.ID,
			format.Truncate(h.Document.Title, 80),
			format.Truncate(h.Context, 120),
		))
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "# returned %d of %d (offset %d, limit %d)\n", len(hits), page.Total, in.Offset, d.limit(in.Limit))
	return b.String(), nil
}

type getDocumentIn struct {
	ID       string `json:"id" jsonschema:"document id"`
	MaxChars int    `json:"max_chars,omitempty" jsonschema:"clip the markdown text at this many characters (default 4000)"`
	Full     bool   `json:"full,omitempty" jsonschema:"return the whole text, ignoring max_chars"`
}

func (d *Deps) getDocument(ctx context.Context, in getDocumentIn) (string, error) {
	if strings.TrimSpace(in.ID) == "" {
		return "", fmt.Errorf("id is required")
	}
	doc, err := d.C.Document(ctx, in.ID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#%s %s\n", shortID(doc.ID), doc.Title)
	b.WriteString(format.Row(
		"collection: "+dash(doc.CollectionID),
		"parent: "+dash(doc.ParentDocumentID),
		"revision: "+strconv.Itoa(doc.Revision),
	) + "\n")
	b.WriteString(format.Row(
		"created: "+format.When(doc.CreatedAt),
		"updated: "+format.When(doc.UpdatedAt),
		"published: "+format.When(doc.PublishedAt),
	) + "\n")
	fmt.Fprintf(&b, "url: %s\n", d.C.AbsoluteURL(doc.URL))
	if strings.TrimSpace(doc.Text) != "" {
		limit := in.MaxChars
		if limit <= 0 {
			limit = 4000
		}
		text := doc.Text
		if !in.Full {
			text = format.Clip(text, limit)
		}
		b.WriteString("\n[text]\n")
		b.WriteString(text)
		if !strings.HasSuffix(text, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// shortID keeps the first 8 characters so lines stay readable.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Jnchk/outline-mcp/internal/outline"
)

type createDocumentIn struct {
	Title        string `json:"title" jsonschema:"document title (required)"`
	Collection   string `json:"collection_id,omitempty" jsonschema:"collection id from list_collections; required unless parent_document_id is set"`
	Parent       string `json:"parent_document_id,omitempty" jsonschema:"create as a child of this document id"`
	Text         string `json:"text,omitempty" jsonschema:"markdown text; required unless text_upload_id is set"`
	TextUploadID string `json:"text_upload_id,omitempty" jsonschema:"id from POST /upload; use for long documents"`
	Publish      *bool  `json:"publish,omitempty" jsonschema:"publish immediately (default true)"`
}

func (d *Deps) createDocument(ctx context.Context, in createDocumentIn) (string, error) {
	if err := d.guardWrite(); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Title) == "" {
		return "", fmt.Errorf("title is required")
	}
	if strings.TrimSpace(in.Collection) == "" && strings.TrimSpace(in.Parent) == "" {
		return "", fmt.Errorf("collection_id or parent_document_id is required")
	}
	text, err := d.resolveUpload("text", in.Text, in.TextUploadID)
	if err != nil {
		return "", err
	}

	doc, err := d.C.CreateDocument(ctx, outline.CreateDocumentInput{
		Title:            in.Title,
		CollectionID:     strings.TrimSpace(in.Collection),
		ParentDocumentID: strings.TrimSpace(in.Parent),
		Text:             text,
		Publish:          in.Publish,
	})
	if err != nil {
		return "", err
	}
	d.consumeUpload(in.TextUploadID)
	return fmt.Sprintf("#%s created. %s", shortID(doc.ID), d.C.AbsoluteURL(doc.URL)), nil
}

type updateDocumentIn struct {
	ID           string `json:"id" jsonschema:"document id (required)"`
	Title        string `json:"title,omitempty" jsonschema:"new title"`
	Text         string `json:"text,omitempty" jsonschema:"markdown text; required unless text_upload_id is set"`
	TextUploadID string `json:"text_upload_id,omitempty" jsonschema:"id from POST /upload; use for long text"`
	EditMode     string `json:"edit_mode,omitempty" jsonschema:"replace (default), append or prepend"`
	Publish      *bool  `json:"publish,omitempty" jsonschema:"publish the draft"`
	Confirm      bool   `json:"confirm,omitempty" jsonschema:"required when replacing the whole text"`
}

func (d *Deps) updateDocument(ctx context.Context, in updateDocumentIn) (string, error) {
	if err := d.guardWrite(); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.ID) == "" {
		return "", fmt.Errorf("id is required")
	}
	mode := strings.ToLower(strings.TrimSpace(in.EditMode))
	if mode == "" {
		mode = "replace"
	}
	if mode != "replace" && mode != "append" && mode != "prepend" {
		return "", fmt.Errorf("edit_mode must be replace, append or prepend")
	}
	if in.Title == "" && in.Text == "" && in.TextUploadID == "" {
		return "", fmt.Errorf("nothing to update: pass title or text")
	}
	text, err := d.resolveUpload("text", in.Text, in.TextUploadID)
	if err != nil {
		return "", err
	}
	if mode == "replace" && text != "" && !in.Confirm {
		return "", fmt.Errorf("replacing the whole text of %s loses the current body; pass confirm: true (or use edit_mode append/prepend)", in.ID)
	}

	doc, err := d.C.UpdateDocument(ctx, outline.UpdateDocumentInput{
		ID:       in.ID,
		Title:    in.Title,
		Text:     text,
		EditMode: mode,
		Publish:  in.Publish,
	})
	if err != nil {
		return "", err
	}
	d.consumeUpload(in.TextUploadID)

	var changes []string
	if in.Title != "" {
		changes = append(changes, "title")
	}
	if text != "" {
		changes = append(changes, fmt.Sprintf("text %s (%d chars)", mode, len([]rune(text))))
	}
	return fmt.Sprintf("#%s updated: %s. %s", shortID(doc.ID), strings.Join(changes, ", "), d.C.AbsoluteURL(doc.URL)), nil
}

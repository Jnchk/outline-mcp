package tools

import (
	"context"
	"fmt"
	"strings"
)

type deleteDocumentIn struct {
	ID        string `json:"id" jsonschema:"document id (required)"`
	Confirm   string `json:"confirm,omitempty" jsonschema:"must be 'yes' to delete"`
	Permanent bool   `json:"permanent,omitempty" jsonschema:"delete permanently instead of moving to trash (admin only)"`
}

func (d *Deps) deleteDocument(ctx context.Context, in deleteDocumentIn) (string, error) {
	if err := d.guardWrite(); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.ID) == "" {
		return "", fmt.Errorf("id is required")
	}
	if !strings.EqualFold(strings.TrimSpace(in.Confirm), "yes") {
		return "", fmt.Errorf("deletion not confirmed: set confirm='yes' to delete %s", in.ID)
	}
	if err := d.C.DeleteDocument(ctx, in.ID, in.Permanent); err != nil {
		return "", err
	}
	state := "moved to trash"
	if in.Permanent {
		state = "permanently deleted"
	}
	return fmt.Sprintf("#%s %s.", shortID(in.ID), state), nil
}

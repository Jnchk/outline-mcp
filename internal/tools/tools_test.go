package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Jnchk/outline-mcp/internal/outline"
	"github.com/Jnchk/outline-mcp/internal/staging"
)

type fakeState struct {
	createBody map[string]any
	updateBody map[string]any
	deleteBody map[string]any
}

func fakeOutline(t *testing.T) (*httptest.Server, *fakeState) {
	t.Helper()
	st := &fakeState{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth.info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"ok":true,"data":{"user":{"id":"u1","name":"Alice","email":"a@x","role":"member"}}}`)
	})
	mux.HandleFunc("/api/collections.list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"ok":true,"data":[{"id":"c1","name":"常用手顺","permission":"read_write","updatedAt":"2026-11-03T07:09:22.474Z"}],"pagination":{"limit":25,"offset":0,"total":1}}`)
	})
	mux.HandleFunc("/api/documents.list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"ok":true,"data":[{"id":"d1","title":"Doc","url":"/doc/d1-abc","updatedAt":"2026-11-18T01:00:00Z","parentDocumentId":"p1"}],"pagination":{"limit":25,"offset":0,"total":1}}`)
	})
	mux.HandleFunc("/api/documents.search", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"ok":true,"data":[{"context":"…登录…","document":{"id":"d1","title":"Doc"},"ranking":1.2}],"pagination":{"limit":25,"offset":0,"total":1}}`)
	})
	mux.HandleFunc("/api/documents.info", func(w http.ResponseWriter, r *http.Request) {
		text := strings.Repeat("字", 5000)
		writeJSON(w, `{"ok":true,"data":{"id":"d1","title":"Doc","url":"/doc/d1-abc","collectionId":"c1","revision":3,"createdAt":"2026-11-01T00:00:00Z","updatedAt":"2026-11-18T01:00:00Z","text":"`+text+`"}}`)
	})
	mux.HandleFunc("/api/documents.create", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&st.createBody)
		writeJSON(w, `{"ok":true,"data":{"id":"d2","title":"New","url":"/doc/new-abc"}}`)
	})
	mux.HandleFunc("/api/documents.update", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&st.updateBody)
		writeJSON(w, `{"ok":true,"data":{"id":"d1","title":"Doc","url":"/doc/d1-abc"}}`)
	})
	mux.HandleFunc("/api/documents.delete", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&st.deleteBody)
		writeJSON(w, `{"ok":true,"data":{}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, st
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func testDeps(t *testing.T, readOnly bool) (*Deps, *fakeState) {
	t.Helper()
	srv, st := fakeOutline(t)
	c, err := outline.New(srv.URL, "secret", time.Second, false)
	if err != nil {
		t.Fatal(err)
	}
	store, err := staging.New(t.TempDir(), time.Minute, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return &Deps{C: c, Staging: store, ReadOnly: readOnly, DefaultLimit: 25}, st
}

func connect(t *testing.T, d *Deps) *mcp.ClientSession {
	t.Helper()
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	Register(s, d)
	ct, sct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(context.Background(), sct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "c"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String(), res.IsError
}

func TestReadTools(t *testing.T) {
	d, _ := testDeps(t, false)
	cs := connect(t, d)

	out, isErr := call(t, cs, "whoami", nil)
	if isErr || !strings.Contains(out, "Alice") || !strings.Contains(out, "read-write") {
		t.Fatalf("whoami = %q", out)
	}

	out, isErr = call(t, cs, "list_collections", map[string]any{})
	if isErr || !strings.Contains(out, "常用手顺") || !strings.Contains(out, "# returned 1 of 1") {
		t.Fatalf("collections = %q", out)
	}

	out, isErr = call(t, cs, "list_documents", map[string]any{"collection_id": "c1"})
	if isErr || !strings.Contains(out, "d1 | Doc") || !strings.Contains(out, "/doc/d1-abc") {
		t.Fatalf("documents = %q", out)
	}

	out, isErr = call(t, cs, "search_documents", map[string]any{"query": "登录"})
	if isErr || !strings.Contains(out, "d1 | Doc") || !strings.Contains(out, "登录") {
		t.Fatalf("search = %q", out)
	}

	out, isErr = call(t, cs, "get_document", map[string]any{"id": "d1"})
	if isErr || !strings.Contains(out, "truncated, 5000 chars total") {
		t.Fatalf("clipped doc = %q", out)
	}
	out, isErr = call(t, cs, "get_document", map[string]any{"id": "d1", "full": true})
	if isErr || strings.Contains(out, "truncated") {
		t.Fatalf("full doc was clipped: %q", out)
	}
}

func TestCreateFromUpload(t *testing.T) {
	d, st := testDeps(t, false)
	info, err := d.Staging.Stage(strings.NewReader("# 大文档\n\n正文"), "doc.md")
	if err != nil {
		t.Fatal(err)
	}
	cs := connect(t, d)

	out, isErr := call(t, cs, "create_document", map[string]any{
		"title": "New", "collection_id": "c1", "text_upload_id": info.ID,
	})
	if isErr || !strings.Contains(out, "created") {
		t.Fatalf("create = %q (error=%t)", out, isErr)
	}
	if st.createBody["text"] != "# 大文档\n\n正文" || st.createBody["collectionId"] != "c1" {
		t.Fatalf("payload = %#v", st.createBody)
	}
	if _, err := d.Staging.Read(info.ID); err == nil {
		t.Fatal("upload id was not consumed")
	}
}

func TestUpdateRules(t *testing.T) {
	d, st := testDeps(t, false)
	cs := connect(t, d)

	out, isErr := call(t, cs, "update_document", map[string]any{
		"id": "d1", "text": "replacement",
	})
	if !isErr || !strings.Contains(out, "confirm: true") {
		t.Fatalf("replace without confirm = %q (error=%t)", out, isErr)
	}

	out, isErr = call(t, cs, "update_document", map[string]any{
		"id": "d1", "text": "## appended", "edit_mode": "append",
	})
	if isErr || !strings.Contains(out, "text append") {
		t.Fatalf("append = %q (error=%t)", out, isErr)
	}
	if st.updateBody["editMode"] != "append" || st.updateBody["text"] != "## appended" {
		t.Fatalf("payload = %#v", st.updateBody)
	}
}

func TestWriteGuards(t *testing.T) {
	d, _ := testDeps(t, true)
	cs := connect(t, d)
	out, isErr := call(t, cs, "create_document", map[string]any{"title": "x", "collection_id": "c1"})
	if !isErr || !strings.Contains(out, "read-only") {
		t.Fatalf("read-only = %q", out)
	}

	d2, _ := testDeps(t, false)
	cs2 := connect(t, d2)
	out, isErr = call(t, cs2, "create_document", map[string]any{
		"title": "x", "collection_id": "c1", "text": "inline", "text_upload_id": "abc",
	})
	if !isErr {
		t.Fatalf("both text sources accepted: %q", out)
	}
	out, isErr = call(t, cs2, "create_document", map[string]any{"title": "x"})
	if !isErr || !strings.Contains(out, "collection_id or parent_document_id") {
		t.Fatalf("missing parent = %q", out)
	}
}

func TestDeleteDocument(t *testing.T) {
	d, st := testDeps(t, false)
	cs := connect(t, d)

	out, isErr := call(t, cs, "delete_document", map[string]any{"id": "d1"})
	if !isErr || !strings.Contains(out, "confirm='yes'") {
		t.Fatalf("no confirm = %q (error=%t)", out, isErr)
	}

	out, isErr = call(t, cs, "delete_document", map[string]any{"id": "d1", "confirm": "yes"})
	if isErr || !strings.Contains(out, "moved to trash") {
		t.Fatalf("delete = %q (error=%t)", out, isErr)
	}
	if st.deleteBody["id"] != "d1" || st.deleteBody["permanent"] != false {
		t.Fatalf("payload = %#v", st.deleteBody)
	}

	out, isErr = call(t, cs, "delete_document", map[string]any{"id": "d1", "confirm": "yes", "permanent": true})
	if isErr || st.deleteBody["permanent"] != true {
		t.Fatalf("permanent = %q payload=%#v", out, st.deleteBody)
	}
}

func TestDeleteReadOnly(t *testing.T) {
	d, _ := testDeps(t, true)
	cs := connect(t, d)
	out, isErr := call(t, cs, "delete_document", map[string]any{"id": "d1", "confirm": "yes"})
	if !isErr || !strings.Contains(out, "read-only") {
		t.Fatalf("read-only = %q", out)
	}
}

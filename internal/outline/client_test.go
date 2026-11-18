package outline

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestURLNormalization(t *testing.T) {
	cases := map[string]string{
		"https://h:7643":     "https://h:7643/api",
		"https://h:7643/":    "https://h:7643/api",
		"https://h:7643/api": "https://h:7643/api",
		"https://h/wiki":     "https://h/wiki/api",
	}
	for in, want := range cases {
		c, err := New(in, "t", time.Second, false)
		if err != nil {
			t.Fatal(err)
		}
		if c.Base() != want {
			t.Fatalf("New(%q).Base() = %q, want %q", in, c.Base(), want)
		}
	}
	if _, err := New("ftp://h", "t", time.Second, false); err == nil {
		t.Fatal("expected an error for a non-http URL")
	}
}

func TestCallSendsAuthAndDecodes(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		io.WriteString(w, `{"ok":true,"data":{"user":{"id":"u1","name":"Alice","email":"a@x","role":"member"}}}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	u, err := c.AuthInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" || gotPath != "/api/auth.info" {
		t.Fatalf("auth=%q path=%q", gotAuth, gotPath)
	}
	if u.Name != "Alice" || u.Role != "member" {
		t.Fatalf("user = %#v", u)
	}
}

func TestCallPropagatesAPIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"ok":false,"error":"authorization_error","message":"Authorization error"}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	_, _, err := c.Collections(context.Background(), CollectionFilter{})
	if err == nil {
		t.Fatal("expected an error")
	}
	var e *Error
	if !errors.As(err, &e) || e.Status != http.StatusBadGateway || !strings.Contains(e.Error(), "Authorization error") {
		t.Fatalf("err = %#v", err)
	}
}

func TestCallHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"ok":false,"message":"boom"}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	_, err := c.AuthInfo(context.Background())
	var e *Error
	if !errors.As(err, &e) || e.Status != 500 || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestAbsoluteURL(t *testing.T) {
	c, _ := New("https://h:7643", "t", time.Second, false)
	if got := c.AbsoluteURL("/doc/abc"); got != "https://h:7643/doc/abc" {
		t.Fatalf("absolute = %q", got)
	}
	if got := c.AbsoluteURL("https://other/x"); got != "https://other/x" {
		t.Fatalf("absolute passthrough = %q", got)
	}
}

func TestCreateDocumentPayload(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		io.WriteString(w, `{"ok":true,"data":{"id":"d1","title":"T","url":"/doc/t-abc"}}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	publish := true
	doc, err := c.CreateDocument(context.Background(), CreateDocumentInput{
		Title: "T", CollectionID: "c1", Text: "# body", Publish: &publish,
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID != "d1" || doc.URL != "/doc/t-abc" {
		t.Fatalf("doc = %#v", doc)
	}
	if body["title"] != "T" || body["collectionId"] != "c1" || body["text"] != "# body" || body["publish"] != true {
		t.Fatalf("payload = %#v", body)
	}
}

func TestPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":true,"data":[{"id":"c1","name":"N"}],"pagination":{"limit":1,"offset":0,"total":7}}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	cols, page, err := c.Collections(context.Background(), CollectionFilter{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 1 || page.Total != 7 {
		t.Fatalf("cols=%v page=%#v", cols, page)
	}
}

func TestReadMethods(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/documents.list":
			io.WriteString(w, `{"ok":true,"data":[{"id":"d1","title":"T","url":"/doc/t"}],"pagination":{"total":1}}`)
		case "/api/documents.info":
			io.WriteString(w, `{"ok":true,"data":{"id":"d1","title":"T","text":"body","revision":2}}`)
		case "/api/documents.search":
			io.WriteString(w, `{"ok":true,"data":[{"context":"ctx","document":{"id":"d1","title":"T"}}],"pagination":{"total":1}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	docs, _, err := c.Documents(context.Background(), DocumentFilter{CollectionID: "c1", Limit: 2})
	if err != nil || len(docs) != 1 || docs[0].ID != "d1" {
		t.Fatalf("documents = %#v, %v", docs, err)
	}
	doc, err := c.Document(context.Background(), "d1")
	if err != nil || doc.Text != "body" || doc.Revision != 2 {
		t.Fatalf("document = %#v, %v", doc, err)
	}
	hits, _, err := c.Search(context.Background(), SearchFilter{Query: "x"})
	if err != nil || len(hits) != 1 || hits[0].Document.ID != "d1" {
		t.Fatalf("search = %#v, %v", hits, err)
	}
	if len(paths) != 3 {
		t.Fatalf("paths = %v", paths)
	}
}

func TestUpdateDocumentPayload(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		io.WriteString(w, `{"ok":true,"data":{"id":"d1","url":"/doc/d1"}}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	if _, err := c.UpdateDocument(context.Background(), UpdateDocumentInput{
		ID: "d1", Text: "## more", EditMode: "append",
	}); err != nil {
		t.Fatal(err)
	}
	if body["id"] != "d1" || body["text"] != "## more" || body["editMode"] != "append" {
		t.Fatalf("payload = %#v", body)
	}
}

func TestMethodErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":false,"message":"nope"}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	if _, err := c.Document(context.Background(), "d1"); err == nil {
		t.Fatal("Document: expected an error")
	}
	if _, err := c.CreateDocument(context.Background(), CreateDocumentInput{Title: "x"}); err == nil {
		t.Fatal("CreateDocument: expected an error")
	}
	if _, err := c.UpdateDocument(context.Background(), UpdateDocumentInput{ID: "d1"}); err == nil {
		t.Fatal("UpdateDocument: expected an error")
	}
}

func TestErrorString(t *testing.T) {
	if got := (&Error{Status: 500}).Error(); got != "outline: 500" {
		t.Fatalf("Error() = %q", got)
	}
	if got := (&Error{Status: 403, Message: "no"}).Error(); got != "outline: 403 no" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestNonJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "<html>bad gateway</html>")
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	_, err := c.AuthInfo(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bad gateway") {
		t.Fatalf("err = %v", err)
	}
}

func TestDocumentsOmitsEmptyFilters(t *testing.T) {
	bodies := map[string]map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		bodies[r.URL.Path] = body
		io.WriteString(w, `{"ok":true,"data":[],"pagination":{"total":0}}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	if _, _, err := c.Documents(context.Background(), DocumentFilter{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Collections(context.Background(), CollectionFilter{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	for path, body := range bodies {
		for _, key := range []string{"collectionId", "parentDocumentId", "query"} {
			if _, ok := body[key]; ok {
				t.Fatalf("%s sent empty %s: %#v", path, key, body)
			}
		}
	}
}

func TestDeleteDocumentPayload(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		io.WriteString(w, `{"ok":true,"data":{}}`)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "secret", time.Second, false)
	if err := c.DeleteDocument(context.Background(), "d1", true); err != nil {
		t.Fatal(err)
	}
	if body["id"] != "d1" || body["permanent"] != true {
		t.Fatalf("payload = %#v", body)
	}
}

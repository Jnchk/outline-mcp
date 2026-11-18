// Package outline is a thin client for the Outline RPC API.
package outline

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base  string // ends with /api
	token string
	hc    *http.Client
}

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("outline: %d", e.Status)
	}
	return fmt.Sprintf("outline: %d %s", e.Status, e.Message)
}

func New(rawURL, token string, timeout time.Duration, insecure bool) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(rawURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("outline: invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("outline: URL must be http(s)")
	}
	if !strings.HasSuffix(u.Path, "/api") {
		u.Path = strings.TrimRight(u.Path, "/") + "/api"
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		base:  u.String(),
		token: token,
		hc:    &http.Client{Timeout: timeout, Transport: tr},
	}, nil
}

type envelope struct {
	OK         bool            `json:"ok"`
	Data       json.RawMessage `json:"data"`
	Pagination *Pagination     `json:"pagination"`
	Error      string          `json:"error"`
	Message    string          `json:"message"`
}

// call posts one RPC method and decodes data into out.
func (c *Client) call(ctx context.Context, method string, in, out any) (Pagination, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return Pagination{}, fmt.Errorf("outline: encoding %s: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/"+method, bytes.NewReader(payload))
	if err != nil {
		return Pagination{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return Pagination{}, fmt.Errorf("outline: %s: %w", method, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return Pagination{}, fmt.Errorf("outline: reading %s: %w", method, err)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return Pagination{}, &Error{Status: resp.StatusCode, Message: snippet(body)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !env.OK {
		msg := env.Message
		if msg == "" {
			msg = env.Error
		}
		status := resp.StatusCode
		if status < 400 {
			status = http.StatusBadGateway
		}
		return Pagination{}, &Error{Status: status, Message: msg}
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return Pagination{}, fmt.Errorf("outline: decoding %s: %w", method, err)
		}
	}
	if env.Pagination != nil {
		return *env.Pagination, nil
	}
	return Pagination{}, nil
}

func (c *Client) AuthInfo(ctx context.Context) (*User, error) {
	var r struct {
		User *User `json:"user"`
	}
	if _, err := c.call(ctx, "auth.info", map[string]any{}, &r); err != nil {
		return nil, err
	}
	if r.User == nil {
		return nil, fmt.Errorf("outline: auth.info returned no user")
	}
	return r.User, nil
}

func (c *Client) Collections(ctx context.Context, f CollectionFilter) ([]Collection, Pagination, error) {
	var out []Collection
	p, err := c.call(ctx, "collections.list", map[string]any{
		"query": f.Query, "limit": f.Limit, "offset": f.Offset,
	}, &out)
	return out, p, err
}

func (c *Client) Documents(ctx context.Context, f DocumentFilter) ([]Document, Pagination, error) {
	var out []Document
	p, err := c.call(ctx, "documents.list", map[string]any{
		"collectionId":     f.CollectionID,
		"parentDocumentId": f.ParentDocumentID,
		"query":            f.Query,
		"limit":            f.Limit,
		"offset":           f.Offset,
	}, &out)
	return out, p, err
}

func (c *Client) Document(ctx context.Context, id string) (*Document, error) {
	var out Document
	if _, err := c.call(ctx, "documents.info", map[string]any{"id": id}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Search(ctx context.Context, f SearchFilter) ([]SearchHit, Pagination, error) {
	var out []SearchHit
	p, err := c.call(ctx, "documents.search", map[string]any{
		"query": f.Query, "limit": f.Limit, "offset": f.Offset,
	}, &out)
	return out, p, err
}

func (c *Client) CreateDocument(ctx context.Context, in CreateDocumentInput) (*Document, error) {
	var out Document
	if _, err := c.call(ctx, "documents.create", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateDocument(ctx context.Context, in UpdateDocumentInput) (*Document, error) {
	var out Document
	if _, err := c.call(ctx, "documents.update", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func snippet(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return s
}

// Base is the API root, ending in /api.
func (c *Client) Base() string { return c.base }

// AbsoluteURL turns Outline's relative document URLs into links.
func (c *Client) AbsoluteURL(p string) string {
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimSuffix(c.base, "/api") + p
}

package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Jnchk/outline-mcp/internal/config"
	"github.com/Jnchk/outline-mcp/internal/outline"
	"github.com/Jnchk/outline-mcp/internal/staging"
	"github.com/Jnchk/outline-mcp/internal/tools"
)

const version = "0.1.0"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.FromEnv()
	if err != nil {
		logger.Error("configuration", "err", err)
		os.Exit(1)
	}

	fs := flag.NewFlagSet("outline-mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&cfg.Transport, "transport", cfg.Transport, "stdio or http")
	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "HTTP listen address")
	fs.BoolVar(&cfg.ReadOnly, "read-only", cfg.ReadOnly, "disable write tools")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if err := cfg.Validate(); err != nil {
		logger.Error("configuration", "err", err)
		os.Exit(1)
	}
	if cfg.IsPlainHTTP() {
		logger.Warn("OUTLINE_URL is not https: the API token travels in clear text")
	}
	if cfg.Insecure {
		logger.Warn("TLS verification is disabled (OUTLINE_INSECURE)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := outline.New(cfg.URL, cfg.Token, cfg.Timeout, cfg.Insecure)
	if err != nil {
		logger.Error("client", "err", err)
		os.Exit(1)
	}

	// Fail fast: proves URL, token and TLS before a client talks to us.
	checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	user, err := client.AuthInfo(checkCtx)
	cancel()
	if err != nil {
		logger.Error("cannot reach Outline", "url", cfg.URL, "err", err)
		os.Exit(1)
	}
	logger.Info("connected", "url", cfg.URL, "user", user.Name, "read_only", cfg.ReadOnly)

	stagingStore, err := staging.New("", 15*time.Minute, cfg.MaxUpload)
	if err != nil {
		logger.Error("staging", "err", err)
		os.Exit(1)
	}

	srv := mcp.NewServer(&mcp.Implementation{Name: "outline-mcp", Version: version}, &mcp.ServerOptions{
		Instructions: instructions(cfg, user.Name),
	})
	tools.Register(srv, &tools.Deps{
		C:            client,
		Staging:      stagingStore,
		ReadOnly:     cfg.ReadOnly,
		DefaultLimit: cfg.DefaultLimit,
	})

	if cfg.Transport == "stdio" {
		err = srv.Run(ctx, &mcp.StdioTransport{})
	} else {
		err = serveHTTP(ctx, cfg, srv, stagingStore, logger)
	}
	if err != nil {
		logger.Error("server", "err", err)
		os.Exit(1)
	}
}

func serveHTTP(ctx context.Context, cfg *config.Config, srv *mcp.Server, store *staging.Store, logger *slog.Logger) error {
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)

	mux := http.NewServeMux()
	mux.Handle(cfg.Path, withToken(cfg.MCPToken, handler))
	mux.Handle("/upload", withToken(cfg.MCPToken, uploadHandler(store)))

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Addr, err)
	}
	hs := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = hs.Shutdown(shutdown)
	}()
	logger.Info("listening", "addr", ln.Addr().String(), "path", cfg.Path)
	err = hs.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func instructions(cfg *config.Config, user string) string {
	mode := "read-write"
	if cfg.ReadOnly {
		mode = "read-only"
	}
	return fmt.Sprintf(`Outline at %s. Signed in as %s. Mode: %s.
- Outline is a wiki; ids are UUIDs. Use list_collections to find collection ids, list_documents or search_documents to find document ids.
- get_document returns markdown clipped at 4000 chars; pass full=true only when the whole text is needed.
- Long text: stage it with POST /upload then pass text_upload_id instead of text.
- update_document replaces the body by default and needs confirm: true; edit_mode append/prepend does not.
- After a create or update, show the document URL to the user.`,
		cfg.URL, user, mode)
}

func withToken(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if len(got) > 7 && got[:7] == "Bearer " {
			got = got[7:]
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

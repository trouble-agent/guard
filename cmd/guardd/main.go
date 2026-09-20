// Command guardd is the lightweight guard service.
//
// Three shapes, one core:
//
//	guardd                 # CLI: reads content on stdin, prints the Result JSON
//	guardd -content "..."  # CLI: one-shot, argument form
//	guardd -serve :8768    # HTTP: POST /check  {input} -> Result JSON
//
// Zero dependencies, one static binary. task-router (Python) calls the HTTP
// form; crier (Go) imports the package directly; shell scripts pipe through the
// CLI. Same verdict everywhere.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/trouble-agent/guard"
)

func main() {
	var (
		serve   = flag.String("serve", "", "address to serve HTTP on, e.g. :8768 (empty = CLI mode)")
		content = flag.String("content", "", "content to classify (empty = read stdin)")
		source  = flag.String("source", "", "source label, e.g. github_pr (drives policy overrides)")
		channel = flag.String("channel", "", "channel label, e.g. crier|task-router")
		fail    = flag.Bool("fail-closed", true, "block on guard error (false = deliver flagged)")
		pretty  = flag.Bool("pretty", false, "indent the JSON output")
	)
	flag.Parse()

	g := &guard.Guard{Classifier: guard.NewJev(), Policy: guard.DefaultPolicy()}
	g.Policy.FailClosed = *fail

	if *serve != "" {
		serveHTTP(*serve, g)
		return
	}

	text := *content
	if text == "" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("read stdin: %v", err)
		}
		text = string(b)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	res := g.Check(ctx, guard.Input{Source: *source, Channel: *channel, Content: text})
	emit(res, *pretty)
}

func serveHTTP(addr string, g *guard.Guard) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var in guard.Input
		if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&in); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if in.Content == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(g.Check(ctx, in))
	})
	log.Printf("guardd listening on %s (POST /check, GET /healthz)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func emit(v any, pretty bool) {
	enc := json.NewEncoder(os.Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}

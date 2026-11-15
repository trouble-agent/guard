package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/trouble-agent/guard"
)

// BenchmarkDisabledEgressCheck8MiB measures the complete local /check path for
// an 8 MiB-class content body without sending the payload to a classifier.
func BenchmarkDisabledEgressCheck8MiB(b *testing.B) {
	content := strings.Repeat("A", (8<<20)-1024)
	body, err := json.Marshal(guard.Input{Content: content})
	if err != nil {
		b.Fatal(err)
	}
	server := httptest.NewServer(newHandler(
		&guard.Guard{Classifier: egressDisabledClassifier{}, Policy: guard.DefaultPolicy()},
		"token",
		false,
	))
	defer server.Close()

	client := server.Client()
	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/check", bytes.NewReader(body))
		if err != nil {
			b.Fatal(err)
		}
		req.Header.Set(OperatorTokenHeader, "token")
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			b.Fatalf("/check status = %d, body = %q", resp.StatusCode, raw)
		}
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			_ = resp.Body.Close()
			b.Fatal(err)
		}
		_ = resp.Body.Close()
	}
}

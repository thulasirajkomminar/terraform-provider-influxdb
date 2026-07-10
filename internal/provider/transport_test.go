package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

// TestHTTPClientTransportChain verifies the ordering guarantees of the HTTP
// round tripper chain built by newHTTPClient:
//
//   - the wire request carries the real Authorization header,
//   - the caller receives the unmasked response body,
//   - the HTTP transaction is written to the debug logs, and
//   - the logs never contain the credential, neither from the request
//     headers (the auth header is injected below the logging transport) nor
//     from the response body (masked by regex).
func TestHTTPClientTransportChain(t *testing.T) {
	const token = "super-secret-token"

	var wireAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wireAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","token":"` + token + `"}`))
	}))
	defer server.Close()

	var logOutput strings.Builder
	ctx := tflogtest.RootLogger(context.Background(), &logOutput)

	client := newHTTPClient("Token "+token, []*regexp.Regexp{
		regexp.MustCompile(regexp.QuoteMeta(token)),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("creating request: %s", err)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("executing request: %s", err)
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading response body: %s", err)
	}

	if wireAuthorization != "Token "+token {
		t.Errorf("wire request Authorization header = %q, want %q", wireAuthorization, "Token "+token)
	}

	if !strings.Contains(string(body), token) {
		t.Errorf("caller received a masked body: %s", string(body))
	}

	logs := logOutput.String()

	if !strings.Contains(logs, "Sending HTTP Request") {
		t.Error("logs do not contain the HTTP request entry")
	}

	if !strings.Contains(logs, "Received HTTP Response") {
		t.Error("logs do not contain the HTTP response entry")
	}

	if strings.Contains(logs, token) {
		t.Errorf("logs contain the credential:\n%s", logs)
	}
}

// TestHTTPClientTransportChainPreservesExistingAuthorization verifies the
// authorization round tripper never overwrites a header set by a higher
// layer (e.g. the Basic header used by the username/password sign-in).
func TestHTTPClientTransportChainPreservesExistingAuthorization(t *testing.T) {
	var wireAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wireAuthorization = r.Header.Get("Authorization")
	}))
	defer server.Close()

	client := newHTTPClient("Token some-token", nil)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("creating request: %s", err)
	}
	req.Header.Set("Authorization", "Basic already-set")

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("executing request: %s", err)
	}
	defer func() { _ = res.Body.Close() }()

	if wireAuthorization != "Basic already-set" {
		t.Errorf("wire request Authorization header = %q, want %q", wireAuthorization, "Basic already-set")
	}
}

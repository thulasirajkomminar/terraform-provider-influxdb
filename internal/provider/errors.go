package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	ihttp "github.com/influxdata/influxdb-client-go/v2/api/http"
)

// isNotFoundError reports whether err is a confirmed "resource does not
// exist" error from the InfluxDB API. It first inspects the typed error
// returned by the client (HTTP status code and API error code) and only
// falls back to matching the well-known "not found" message produced by the
// generated API client. Transport failures, malformed responses, and every
// other error shape are NOT treated as not-found, so callers never remove
// Terraform state based on an ambiguous error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Typed error from the influxdb-client-go api package.
	var httpErr *ihttp.Error
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusNotFound ||
			strings.EqualFold(httpErr.Code, "not found")
	}

	// The generated domain client and the api helpers flatten errors into
	// strings such as "not found: bucket not found", "404 Not Found: ..."
	// or "organization 'name' not found".
	msg := strings.ToLower(err.Error())

	return strings.HasPrefix(msg, "not found:") ||
		strings.HasPrefix(msg, "404 ") ||
		strings.HasSuffix(msg, "not found")
}

// formatAPIError renders an InfluxDB client error for a diagnostic detail.
// When the error carries an HTTP status code it is included so unexpected
// responses (e.g. a proxy or load balancer intercepting the request) surface
// the real HTTP failure instead of an opaque formatting error.
func formatAPIError(err error) string {
	if err == nil {
		return ""
	}

	var httpErr *ihttp.Error
	if errors.As(err, &httpErr) && httpErr.StatusCode != 0 {
		return fmt.Sprintf("HTTP status %d: %s", httpErr.StatusCode, httpErr.Error())
	}

	return err.Error()
}

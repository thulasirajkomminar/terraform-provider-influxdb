package provider

import (
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

// newHTTPClient builds the HTTP client used for every InfluxDB API call.
//
// The round tripper chain is ordered so credentials can never reach the
// debug logs:
//
//	masking (tflog mask regexes) -> logging -> authorization -> base
//
// The Authorization header is injected below the logging transport, so it is
// never visible to the logger. The masking transport guards everything the
// logger can still see: credentials sent above this chain (e.g. the Basic
// header used by the username/password sign-in and session cookies) and any
// credential values echoed in response bodies.
func newHTTPClient(authorization string, maskRegexes []*regexp.Regexp) *http.Client {
	transport := http.DefaultTransport
	transport = &authorizationRoundTripper{authorization: authorization, next: transport}
	transport = logging.NewLoggingHTTPTransport(transport)
	transport = &maskingRoundTripper{regexes: maskRegexes, next: transport}

	return &http.Client{Transport: transport}
}

// maskingRoundTripper injects tflog masking rules into the request context
// so the downstream logging transport redacts credential values from logged
// headers, messages, and bodies.
type maskingRoundTripper struct {
	regexes []*regexp.Regexp
	next    http.RoundTripper
}

func (m *maskingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(m.regexes) > 0 {
		ctx := req.Context()
		ctx = tflog.MaskAllFieldValuesRegexes(ctx, m.regexes...)
		ctx = tflog.MaskMessageRegexes(ctx, m.regexes...)
		req = req.WithContext(ctx)
	}

	return m.next.RoundTrip(req)
}

// authorizationRoundTripper adds the Authorization header to every request
// that does not already carry one. It sits below the logging transport so
// the header value never appears in debug logs.
type authorizationRoundTripper struct {
	authorization string
	next          http.RoundTripper
}

func (a *authorizationRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if a.authorization != "" && req.Header.Get("Authorization") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", a.authorization)
	}

	return a.next.RoundTrip(req)
}

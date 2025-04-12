// https://github.com/motemen/go-loghttp/blob/master/loghttp.go
package logging

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/rs/zerolog"
)

var log zerolog.Logger = logging.GetLog("httpclientlog")

// Transport implements http.RoundTripper. When set as Transport of http.Client, it executes HTTP requests with logging.
// No field is mandatory.
type Transport struct {
	Transport   http.RoundTripper
	LogRequest  func(req *http.Request)
	LogResponse func(resp *http.Response)
}

// THe default logging transport that wraps http.DefaultTransport.
var DefaultTransport = &Transport{
	Transport: http.DefaultTransport,
}

// Used if transport.LogRequest is not set.
var DefaultLogRequest = func(req *http.Request) {
	log.Debug().Msgf("--> %s %s", req.Method, req.URL)
	for k, v := range req.Header {
		if k == "Authorization" {
			v = []string{"hidden"}
		}
		log.Debug().Msgf("      > header %s %s", k, v)
	}
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Warn().Msgf("Error reading body: %v", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset the body
		log.Debug().Msgf("      > body: %s", body)
	}
}

// Used if transport.LogResponse is not set.
var DefaultLogResponse = func(resp *http.Response) {
	ctx := resp.Request.Context()
	if start, ok := ctx.Value(ContextKeyRequestStart).(time.Time); ok {
		log.Debug().Msgf("<-- %d %s (%s)", resp.StatusCode, resp.Request.URL, time.Since(start))
	} else {
		log.Debug().Msgf("<-- %d %s", resp.StatusCode, resp.Request.URL)
	}
	for k, v := range resp.Header {
		log.Debug().Msgf("      < header %s %s", k, v)
	}
	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warn().Msgf("Error reading body: %v", err)
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset the body
		log.Debug().Msgf("      < body: %s", body)
	}
}

type contextKey struct {
	name string
}

var ContextKeyRequestStart = &contextKey{"RequestStart"}

// RoundTrip is the core part of this module and implements http.RoundTripper.
// Executes HTTP request with request/response logging.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), ContextKeyRequestStart, time.Now())
	req = req.WithContext(ctx)

	t.logRequest(req)

	resp, err := t.transport().RoundTrip(req)
	if err != nil {
		return resp, err
	}

	t.logResponse(resp)

	return resp, err
}

func (t *Transport) logRequest(req *http.Request) {
	if t.LogRequest != nil {
		t.LogRequest(req)
	} else {
		DefaultLogRequest(req)
	}
}

func (t *Transport) logResponse(resp *http.Response) {
	if t.LogResponse != nil {
		t.LogResponse(resp)
	} else {
		DefaultLogResponse(resp)
	}
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}

	return http.DefaultTransport
}
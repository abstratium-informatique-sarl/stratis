package mocks

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
)

var log = logging.GetLog("mock-server")

type handler struct {
	Matches func(*http.Request) bool

	// return a string, used in asserting that the handler was called. rather than the handler having a name, the Handle
	// function returns a string, so that it can handle multiple calls
	Handle func(http.ResponseWriter, *http.Request) string
}

type MockServer struct {
	instance *httptest.Server
	port     string
	handlers *[]handler
	calls    *[]string
}

func (s *MockServer) Stop() {
	s.instance.Close()
	log.Info().Msgf("stopped listening with %d stubs on port %s %v %v", len(*s.handlers), s.port, cap(*s.handlers), s.handlers)
}

// create a new mock server which will listen on the given port.
// e.g.:
//
//     mock := mocks.NewClient("9998")
//
// add handlers like this:
//
//     mock.AddHandler(func(r *http.Request) bool {
//		   return true // <<<< a selector which uses the request to decide if this handler should be called
//     },func(w http.ResponseWriter, r *http.Request) string {
//		   w.Write([]byte(""))
//		   return "ok"
//     })
// 
// return a string which is recorded in the calls which can then be asserted - see below.
// alternatively, return an empty string and it will be replaced with the method + requestURI from the request,
// separated by a string.
// 
// start the server:
//
//     mock.Start()
//
// don't forget to stop the server too:
//
//     defer mock.Stop()
//
// you can then obtain the calls with this function and use them to assert what was called:
//
//     calls := mock.GetCalls()
// 
// any calls where no handler matches result in a panic.
// 
func NewServer(port string) *MockServer {
	_handlers := make([]handler, 0, 10)
	_calls := make([]string, 0, 10)

	ms := &MockServer{
		port:     port,
		handlers: &_handlers,
		calls:    &_calls,
	}

	// https://speedscale.com/blog/testing-golang-with-httptest/
	ms.instance = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, h := range *ms.handlers {
				if h.Matches(r) {
					call := h.Handle(w, r)
					if len(call) == 0 {
						call = r.Method + " " + r.RequestURI
					}
					calls := append(*ms.calls, call)
					ms.calls = &calls
					return
				}
			}

			panic(fmt.Sprintf("unexpected url %s %s", r.Method, r.RequestURI))
		}))

	return ms
}

// the `handle` function should return a string, used in asserting that the handler was called.
// rather than the handler having a name, the `handle`
// function returns a string, so that it could handle multiple calls, rather than requiring multiple
// handlers to be added.
func (s *MockServer) AddHandler(matches func(*http.Request) bool,
	handle func(http.ResponseWriter, *http.Request) string) *MockServer {

	handler := &handler{
		Matches: matches,
		Handle:  handle,
	}
	handlers := append(*s.handlers, *handler)
	s.handlers = &handlers
	return s
}

func (s *MockServer) Start() *MockServer {
	l, err := net.Listen("tcp", "localhost:"+s.port)
	if err != nil {
		panic(err)
	}
	s.instance.Listener.Close()
	s.instance.Listener = l
	s.instance.Start()

	log.Info().Msgf("started listening with %d stubs on port %s", len(*s.handlers), s.port)

	return s
}

func (s *MockServer) GetCalls() []string {
	return *s.calls
}

package mock

import (
    "bufio"
    "fmt"
    "github.com/golang/glog"
    "github.com/jarcoal/httpmock"
    "net/http"
    "net/http/httputil"
    "strings"
)

// RoundTripperFunc callable definition
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls RoundTripperFunc
func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
    return fn(req)
}

// MockedHTTPRequest mock http requests
type MockedHTTPRequest struct {
    method   string
    path     string
    response string
    query    map[string]string
}

func (m MockedHTTPRequest) String() string {
    return fmt.Sprintf("%s: %s", m.method, m.path)
}

// NewMockedHTTPRequest return instance of MockedHTTPRequest
func NewMockedHTTPRequest(method string, path string, response string) MockedHTTPRequest {
    return MockedHTTPRequest{method: method, path: path, response: response}
}

// NewMockedHTTPRequestWithQuery return instance of MockedHTTPRequest with query
func NewMockedHTTPRequestWithQuery(method string, path string, response string, query map[string]string) MockedHTTPRequest {
    return MockedHTTPRequest{method: method, path: path, response: response, query: query}
}

func (m MockedHTTPRequest) register() {
    url := fmt.Sprintf("=~%s", m.path)
    httpmock.RegisterResponder(m.method, url, func(request *http.Request) (*http.Response, error) {
        logRequest(request)
        return m.mockRequest(request)
    })
}

func logRequest(r *http.Request) {
    if glog.V(4) {
        dump, err := httputil.DumpRequest(r, true)
        if err != nil {
            glog.Warningf("Could not dump http request for %s: %s", r.URL.Path, err)
        } else {
            glog.Infof("Mock HTTP request dump:\n %s", string(dump))
        }
    } else {
        method := r.Method
        if method == "" {
            method = " GET"
        }
        glog.Infof("Processing Mock HTTP request %s %s", method, r.URL.Path)
    }
}

func (m *MockedHTTPRequest) mockRequest(req *http.Request) (*http.Response, error) {
    resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(m.response)), req)
    if err != nil {
        return nil, err
    }

    return resp, nil
}

// HTTPMockHandler handles requests
type HTTPMockHandler struct {
}

// NewHTTPMockHandler return instance of HTTPMockHandler
func NewHTTPMockHandler() *HTTPMockHandler {
    return &HTTPMockHandler{}
}

// Stop reset transport
func (h *HTTPMockHandler) Stop() {
    httpmock.Deactivate()
}

// Cleanup reset transport
func (h *HTTPMockHandler) Cleanup() {
    httpmock.Reset()
}

// register one or more MockedHTTPRequest
func (h *HTTPMockHandler) Register(httpMocks ...MockedHTTPRequest) {
    for _, httpMock := range httpMocks {
        httpMock.register()
    }
}

// Sniff return response
func (h *HTTPMockHandler) Sniff() {
    http.DefaultTransport = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
        return h.dumpRequestResponseWithTransport(req, httpmock.InitialTransport)
    })
}

// Start handling request
func (h *HTTPMockHandler) Start() {
    httpmock.Activate()
}

// RegisterLocalServer set local server
func (h *HTTPMockHandler) RegisterLocalServer(localServer string, methods ...string) {
    url := "=~^"+localServer

    if len(methods) == 0 {
        methods = []string{"GET", "POST", "PATCH"}
    }

    for _, m := range methods {
        httpmock.RegisterResponder(m, url, func(request *http.Request) (*http.Response, error) {
            return httpmock.InitialTransport.RoundTrip(request)
        })
    }
}

func (h *HTTPMockHandler) dumpRequestResponseWithTransport(req *http.Request, bkpDefaultTransport http.RoundTripper) (*http.Response, error) {
    h.dumpRequest(req)
    response, err := bkpDefaultTransport.RoundTrip(req)
    if err != nil {
        return nil, fmt.Errorf("error during http round trip for %s: %s", req.URL.Path, err)
    }
    h.dumpResponse(response, req)
    return response, nil
}

func (h *HTTPMockHandler) dumpRequest(req *http.Request) {
    dump, err := httputil.DumpRequest(req, true)
    if err != nil {
        glog.Warningf("Could not dump http request for %s: %s", req.URL.Path, err)
    } else {
        glog.Infof("HTTP request dump:\n %s", string(dump))
    }
}

func (h *HTTPMockHandler) dumpResponse(response *http.Response, req *http.Request) {
    dump, err := httputil.DumpResponse(response, true)
    if err != nil {
        glog.Warningf("Could not dump http response for %s: %s", req.URL.Path, err)
    } else {
        glog.Infof("HTTP response dump:\n %s", string(dump))
    }
}

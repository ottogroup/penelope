package rest

import (
    "fmt"
    "net/http"
)

const apiRootAppPath = "/api/"
const healthCheckPath = "/_ah/"
const backupPath = "backups"
const bucketsPath = "buckets"
const datasetsPath = "datasets"
const restorePath = "restore"
const tasksPath = "tasks"
const userPath = "users"


// Endpoint for a HTTP requests
type Endpoint struct {
    root        string
    path        string
    authEnabled bool
    handler     http.HandlerFunc
    methods     []string
}

func newAPIEndpoint(path string, authEnabled bool, handler http.HandlerFunc, methods []string) *Endpoint {
    return &Endpoint{apiRootAppPath, path, authEnabled, handler, methods}
}

func newCustomEndpoint(root string, path string, authEnabled bool, handler http.HandlerFunc, methods []string) *Endpoint {
    return &Endpoint{root, path, authEnabled, handler, methods}
}

func (a *Endpoint) pathWithoutTrailingSlash() string {
    return fmt.Sprintf("%s%s", a.root, a.path)
}

func (a *Endpoint) String() string {
    return a.pathWithoutTrailingSlash()
}

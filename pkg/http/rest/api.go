package rest

import (
    "fmt"
    "github.com/golang/glog"
    "github.com/gorilla/mux"
    "github.com/ottogroup/penelope/pkg/builder"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/http/actions"
    "github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/secret"
    "net/http"
    "net/http/httputil"
)

// API will handle HTTP requests
type API struct {
    router *mux.Router
}

type NewAPIArgs struct {
    ProcessorBuilder    *builder.ProcessorBuilder
    AuthMiddleware      *auth.AuthenticationMiddleware
    TokenSourceProvider impersonate.TargetPrincipalForProjectProvider
    CredentialsProvider secret.SecretProvider
}

func NewAPI(args NewAPIArgs) *API  {
    router := createRouter(args)
    return &API{router}
}

// NewRestAPI return instance of API
func NewRestAPI(processorBuilder *builder.ProcessorBuilder, authMiddleware *auth.AuthenticationMiddleware, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) *API {
    return NewAPI(NewAPIArgs{
        processorBuilder,
        authMiddleware,
        tokenSourceProvider,
        credentialsProvider,
    })
}

func createRouter(args NewAPIArgs) *mux.Router {
    router := mux.NewRouter().StrictSlash(true)
    for _, endpoint := range createEndpoints(args.ProcessorBuilder, args.TokenSourceProvider, args.CredentialsProvider) {
        if endpoint.handler == nil {
            msg := fmt.Sprintf("no handler defined for enpoint: %s", endpoint.pathWithoutTrailingSlash())
            panic(msg)
        }

        h := endpoint.handler.ServeHTTP
        if endpoint.authEnabled {
            h = args.AuthMiddleware.AddAuthentication(h)
        }
        router.HandleFunc(endpoint.pathWithoutTrailingSlash(), h).Methods(endpoint.methods...)
        glog.V(4).Infof("Handler for %s registered\n", endpoint)
    }
    router.NotFoundHandler = notImplementedHandler()

    router.Use(loggingMiddleware)
    router.Use(corsMiddleware)
    return router
}

func createEndpoints(processorBuilder *builder.ProcessorBuilder, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) []*Endpoint {
    return []*Endpoint{
        newAPIEndpoint(
            fmt.Sprintf("%s/{backup_id}", backupPath),
            true,
            actions.NewUpdateBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodPatch},
        ),
        newAPIEndpoint(
            backupPath,
            true,
            actions.NewUpdateBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodPatch},
        ),
        newAPIEndpoint(
            fmt.Sprintf("%s/calculate", backupPath),
            true,
            actions.NewCalculateBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodPost},
        ),
        newAPIEndpoint(
            backupPath,
            true,
            actions.NewAddBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodPost},
        ),
        newAPIEndpoint(
            fmt.Sprintf("%s/{backup_id}", backupPath),
            true,
            actions.NewGettingBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            backupPath,
            true,
            actions.NewListingBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            fmt.Sprintf("%s/{task}", tasksPath),
            false,
            actions.NewTaskRunHandler(tokenSourceProvider, credentialsProvider).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            tasksPath,
            false,
            actions.NewTaskRunHandler(tokenSourceProvider, credentialsProvider).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            fmt.Sprintf("%s/me", userPath),
            true,
            actions.NewGetUserMeHandler().ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            fmt.Sprintf("%s/{backup_id}", restorePath),
            true,
            actions.NewRestoringBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            restorePath,
            true,
            actions.NewRestoringBackupHandler(processorBuilder).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            fmt.Sprintf("%s/{project_id}", datasetsPath),
            true,
            actions.NewDatasetListingHandler(processorBuilder).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            fmt.Sprintf("%s/{project_id}", bucketsPath),
            true,
            actions.NewBucketListingHandler(processorBuilder).ServeHTTP,
            []string{http.MethodGet},
        ),
        newAPIEndpoint(
            bucketsPath,
            true,
            actions.NewBucketListingHandler(processorBuilder).ServeHTTP,
            []string{http.MethodGet},
        ),
        newCustomEndpoint(
            healthCheckPath,
            "health",
            false,
            handleHeathCheck,
            []string{http.MethodGet},
        ),
    }
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    a.router.ServeHTTP(w, r)
}

// Register routine that will wait for the HTTP requests
func (a *API) Register() {
    http.Handle("/", a.router)
    glog.Infoln("Rest api handler registered")
}

func notImplementedHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotImplemented)
        if _, err := fmt.Fprintf(w, "Unkown api endpoint %s", r.URL.Path); err != nil {
            glog.Warningf("Error writing response for %s: %s", r.URL.Path, err)
        }
    })
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        w.Header().Set("Access-Control-Allow-Methods", config.CorsAllowedMethods.GetOrDefault(""))
        w.Header().Set("Access-Control-Allow-Origin", config.CorsAllowedOrigin.GetOrDefault(""))
        w.Header().Set("Access-Control-Allow-Headers", config.CorsAllowedHeaders.GetOrDefault(""))
        if req.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
        } else {
            next.ServeHTTP(w, req)
        }
    })
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if glog.V(4) {
            dump, err := httputil.DumpRequest(r, true)
            if err != nil {
                glog.Warningf("Could not dump http request for %s: %s", r.URL.Path, err)
            } else {
                glog.Infof("HTTP request dump:\n %s", string(dump))
            }
        } else {
            method := r.Method
            if method == "" {
                method = " GET"
            }
            glog.Infof("Processing HTTP request %s %s", method, r.URL.Path)
        }
        next.ServeHTTP(w, r)
    })
}

func handleHeathCheck(w http.ResponseWriter, _ *http.Request) {
    w.WriteHeader(http.StatusOK)
}

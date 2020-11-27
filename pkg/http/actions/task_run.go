package actions

import (
    "fmt"
    "github.com/golang/glog"
    "github.com/gorilla/mux"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/tasks"
    "go.opencensus.io/trace"
    "net"
    "net/http"
    "strings"
)

type TaskRunHandler struct {
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
    credentialsProvider secret.SecretProvider
}

func NewTaskRunHandler(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) *TaskRunHandler {
    return &TaskRunHandler{tokenSourceProvider, credentialsProvider}
}

func (g *TaskRunHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    _, span := trace.StartSpan(r.Context(), "TaskRunHandler.ServeHTTP")
    defer span.End()

    if err := validateRequest(r); err != nil {
        glog.Errorf("request forbidden: %s", err)
        msg := "request forbidden"
        prepareResponse(w, msg, msg, http.StatusForbidden)
        return
    }

    if task, exist := mux.Vars(r)["task"]; exist {
        go tasks.RunTask(task, g.tokenSourceProvider, g.credentialsProvider)
        w.WriteHeader(http.StatusCreated)
        return
    }

    msg := "Bad request missing parameter: task"
    prepareResponse(w, msg, msg, http.StatusBadRequest)
}

func validateRequest(r *http.Request) error {
    if config.TasksValidationHTTPHeaderName.Exist() {
        if !config.TasksValidationHTTPHeaderValue.Exist() {
            return fmt.Errorf("value for HTTP header validation is not provided for: %s", config.TasksValidationHTTPHeaderName)
        }

        validationHeader := config.TasksValidationHTTPHeaderName.MustGet()
        headerValue := r.Header.Get(validationHeader)
        if config.TasksValidationHTTPHeaderValue.MustGet() != headerValue {
            return fmt.Errorf("value for header '%s' not provided or wrong: '%s'", validationHeader, headerValue)
        }
    }

    if config.TasksValidationAllowedIPAddresses.Exist() {
        allowedIPAddressesRaw := config.TasksValidationAllowedIPAddresses.MustGet()
        allowedIPAddresses := strings.Split(allowedIPAddressesRaw, ";")
        var invalidIPAddress = true

        ip, err := getIP(r)
        if err != nil {
            return fmt.Errorf("couldn't validate ip: %s", err)
        }

        for _, ipAddress := range allowedIPAddresses {
            if strings.TrimSpace(ipAddress) == ip {
                invalidIPAddress = false
            }
        }
        if invalidIPAddress {
            return fmt.Errorf("invalid ip address: %s", ip)
        }
    }

    return nil
}

func getIP(r *http.Request) (string, error) {
    ip := r.Header.Get("X-REAL-IP")
    netIP := net.ParseIP(ip)

    if netIP != nil {
        return ip, nil
    }

    ips := r.Header.Get("X-FORWARDED-FOR")
    splitIps := strings.Split(ips, ",")
    for _, ip := range splitIps {
        netIP := net.ParseIP(ip)
        if netIP != nil {
            return ip, nil
        }
    }

    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return "", err
    }
    netIP = net.ParseIP(ip)
    if netIP != nil {
        return ip, nil
    }

    return "", fmt.Errorf("no valid ip found")
}

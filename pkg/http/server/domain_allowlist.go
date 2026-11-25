package server

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

var domainLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func allowedDomains(raw string) (map[string]struct{}, error) {
	domains := make(map[string]struct{})
	for entry := range strings.SplitSeq(raw, ";") {
		domain, err := normalizeDomain(strings.TrimSpace(entry))
		if err != nil {
			return nil, err
		}
		domains[domain] = struct{}{}
	}
	return domains, nil
}

func normalizeDomain(domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	if ip := net.ParseIP(domain); ip != nil {
		return ip.String(), nil
	}
	if domain == "" || len(domain) > 253 {
		return "", fmt.Errorf("invalid allowed hostname %q", domain)
	}
	for label := range strings.SplitSeq(domain, ".") {
		if len(label) > 63 || !domainLabel.MatchString(label) {
			return "", fmt.Errorf("invalid allowed hostname %q", domain)
		}
	}
	return domain, nil
}

func requestDomain(host string) (string, error) {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		ip := net.ParseIP(host[1 : len(host)-1])
		if ip == nil {
			return "", fmt.Errorf("invalid request host %q", host)
		}
		return ip.String(), nil
	}
	if strings.Contains(host, ":") {
		hostname, port, err := net.SplitHostPort(host)
		if err != nil {
			return "", fmt.Errorf("invalid request host %q: %w", host, err)
		}
		number, err := strconv.Atoi(port)
		if err != nil || strings.Trim(port, "0123456789") != "" || number < 1 || number > 65535 {
			return "", fmt.Errorf("invalid request port %q", port)
		}
		host = hostname
	}
	return normalizeDomain(host)
}

func newDomainAllowlist(next http.Handler, normal, tasks string) (http.Handler, error) {
	normalDomains, err := allowedDomains(normal)
	if err != nil {
		return nil, fmt.Errorf("invalid ALLOWED_DOMAINS: %w", err)
	}
	taskDomains, err := allowedDomains(tasks)
	if err != nil {
		return nil, fmt.Errorf("invalid TASKS_ALLOWED_DOMAINS: %w", err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domains := normalDomains
		path := strings.TrimSuffix(r.URL.Path, "/")
		if path == "/api/tasks" || strings.HasPrefix(path, "/api/tasks/") ||
			path == "/_ah/health" || path == "/liveness_check" || path == "/readiness_check" {
			domains = taskDomains
		}
		domain, err := requestDomain(r.Host)
		_, allowed := domains[domain]
		if err != nil || !allowed {
			glog.Warningf("Request forbidden for host %q on path %q: allowed=%t error=%v", r.Host, r.URL.Path, allowed, err)
			http.Error(w, "request forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}

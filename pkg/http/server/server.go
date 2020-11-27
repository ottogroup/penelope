package server

import (
    "fmt"
    "github.com/golang/glog"
    "github.com/gorilla/mux"
    "github.com/ottogroup/penelope/pkg/config"
    "net/http"
    "net/http/pprof"
    "strings"
)

type Server interface {
    Run() error
    RunLocal(staticFilePath string) error
}

func CreateServer(handler http.Handler) Server {
    return &defaultServer{handler}
}

type defaultServer struct {
    handler http.Handler
}

func (s *defaultServer) Run() error {
    port := config.LocalPort.GetOrDefault("8080")
    addr := fmt.Sprintf(":%s", port)
    glog.Infoln("Starting app server on port", port)

    if err := http.ListenAndServe(addr, s.handler); err != nil {
        return fmt.Errorf("could not start http server. err: %s", err)
    }

    return nil
}

func (s *defaultServer) RunLocal(staticFilePath string) error {
    port := config.LocalPort.GetOrDefault("8080")
    addr := fmt.Sprintf(":%s", port)
    glog.Infoln("Starting local server on port", port)

    indexFileServer := http.FileServer(fileSystem{http.Dir(staticFilePath)})
    staticFileServer := http.StripPrefix("/static/ui", http.FileServer(fileSystem{http.Dir(staticFilePath)}))

    isPprofActive := config.PprofActiveEnv.Exist()
    router := mux.NewRouter()
    if isPprofActive {
        // register pprof handlers
        router.HandleFunc("/debug/pprof/", pprof.Index)
        router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
        router.HandleFunc("/debug/pprof/profile", pprof.Profile)
        router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

        router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
        router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
        router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
        router.Handle("/debug/pprof/block", pprof.Handler("block"))
    }

    err := http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" {
            indexFileServer.ServeHTTP(w, r)
        } else if strings.HasPrefix(r.URL.Path, "/static/ui/js") ||
            strings.HasPrefix(r.URL.Path, "/static/ui/images") ||
            strings.HasPrefix(r.URL.Path, "/static/ui/css") {
            staticFileServer.ServeHTTP(w, r)
        } else if isPprofActive && strings.HasPrefix(r.URL.Path, "debug") {
            http.DefaultServeMux.Handle(r.URL.Path, router)
        } else {
            s.handler.ServeHTTP(w, r)
        }
    }))

    if err != nil {
        glog.Errorf("could not start http server. err: %s", err)
        return err
    }

    return nil
}

type fileSystem struct {
    fs http.FileSystem
}

// Open opens file
func (fs fileSystem) Open(path string) (http.File, error) {
    f, err := fs.fs.Open(path)
    if err != nil {
        return nil, err
    }

    s, err := f.Stat()
    if err != nil {
        return nil, fmt.Errorf("file stat failed %s: %v", path, err)
    }
    if s.IsDir() {
        index := strings.TrimSuffix(path, "/") + "/index.html"
        if _, err := fs.fs.Open(index); err != nil {
            return nil, err
        }
    }

    return f, nil
}

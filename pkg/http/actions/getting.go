package actions

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type GettingBackupHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewGettingBackupHandler(processorBuilder *builder.ProcessorBuilder) *GettingBackupHandler {
	return &GettingBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Getting operation
func (dl *GettingBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "GettingBackupHandler.ServeHTTP")
	defer span.End()

	backupID, ok := mux.Vars(r)["backup_id"]
	if !ok {
		msg := "Bad request missing parameter: backup_id"
		prepareResponse(w, msg, msg, http.StatusBadRequest)
		return
	}

	request := requestobjects.GetRequest{}
	request.BackupID = backupID
	q := r.URL.Query()
	if q.Get("size") != "" {
		i, err := strconv.Atoi(q.Get("size"))
		if err != nil {
			BadRequestResponse(w, r)
			return
		}
		request.Page.Size = i
	}
	if q.Get("page") != "" {
		i, err := strconv.Atoi(q.Get("page"))
		if err != nil {
			BadRequestResponse(w, r)
			return
		}
		request.Page.Number = i
	}
	if q.Get("job_statuses") != "" {
		request.JobStatus = strings.Split(q.Get("job_statuses"), ",")
	}

	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, dl.processorBuilder.ProcessorForGetting)
}

func BadRequestResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	if _, err := fmt.Fprintf(w, "Unkown api endpoint %s", r.URL.Path); err != nil {
		escapedPath := html.EscapeString(r.URL.Path)
		glog.Warningf("Error writing response for %s: %s", escapedPath, err)
	}
}

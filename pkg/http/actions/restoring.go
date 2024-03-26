package actions

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type RestoringBackupHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewRestoringBackupHandler(processorBuilder *builder.ProcessorBuilder) *RestoringBackupHandler {
	return &RestoringBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Updating Restoring
func (rb *RestoringBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "RestoringBackupHandler.ServeHTTP")
	defer span.End()

	backupID, exist := mux.Vars(r)["backup_id"]
	if !exist {
		msg := "Bad request missing parameter: backup_id"
		prepareResponse(w, msg, msg, http.StatusBadRequest)
		return
	}

	var request requestobjects.RestoreRequest
	request.BackupID = backupID
	request.JobIDForTimestamp = r.URL.Query().Get("jobIDForTimestamp")

	handleRequestByProcessor(ctx, w, r, request, rb.processorBuilder.ProcessorForRestoring)
}

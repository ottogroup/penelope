package actions

import (
	"github.com/gorilla/mux"
	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
	"net/http"
)

type TrashcanCleanUp struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewTrashcanCleanUp(processorBuilder *builder.ProcessorBuilder) *TrashcanCleanUp {
	return &TrashcanCleanUp{processorBuilder: processorBuilder}
}

// ServeHTTP will handle TrashcanCleanUp operation
func (tc *TrashcanCleanUp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "TrashcanCleanUp.ServeHTTP")
	defer span.End()

	backupID, exist := mux.Vars(r)["backup_id"]
	if !exist {
		msg := "Bad request missing parameter: backup_id"
		prepareResponse(w, msg, msg, http.StatusBadRequest)
		return
	}

	var request = requestobjects.TrashcanCleanUpRequest{
		BackupID: backupID,
	}

	handleRequestByProcessor(ctx, w, r, request, http.StatusNoContent, tc.processorBuilder.ProcessorForTrashcanCleanUp)
}

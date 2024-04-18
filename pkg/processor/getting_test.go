package processor

import (
	"encoding/json"
	"github.com/ottogroup/penelope/pkg/provider"
	"testing"

	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/stretchr/testify/assert"
)

func Test_MakeResponseForArchiveTTM(t *testing.T) {
	backup := repository.Backup{
		SinkOptions: repository.SinkOptions{ArchiveTTM: 123},
	}
	backupResponse := mapBackupToResponse(&backup, []*repository.Job{}, provider.SourceGCPProject{})
	assert.Equal(t, backupResponse.TargetOptions.ArchiveTTM, backup.ArchiveTTM)
	body, err := json.Marshal(&backupResponse)
	assert.Nil(t, err, "expected no error")
	assert.Equal(t, string(`{"id":"","recovery_point_objective":0,"recovery_time_objective":0,"target":{"archive_ttm":123},"snapshot_options":{},"mirror_options":{},"bigquery_options":{},"gcs_options":{},"status":"","sink":"","sink_project":"","data_owner":"","data_availability_class":""}`), string(body))
}

package actions

import (
    "github.com/ottogroup/penelope/pkg/repository"
    "encoding/json"
    "github.com/stretchr/testify/assert"
    "testing"
)

func Test_MakeResponseForArchiveTTM(t *testing.T) {
    backup := repository.Backup{
        SinkOptions: repository.SinkOptions{ArchiveTTM: 123},
    }
    backupResponse := mapBackupToResponse(&backup, []*repository.Job{})
    assert.Equal(t, backupResponse.TargetOptions.ArchiveTTM, backup.ArchiveTTM)
    body, err := json.Marshal(&backupResponse)
    assert.Nil(t, err, "expected no error")
    assert.Equal(t, string(`{"id":"","target":{"archive_ttm":123},"snapshot_options":{},"mirror_options":{},"bigquery_options":{},"gcs_options":{},"status":"","sink":"","sink_project":""}`), string(body))
}

package processor

import (
    "context"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/stretchr/testify/assert"
    "testing"
    "time"
)

func TestCreatingProcessor_validateIntersection_BigQuery(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := &repository.Backup{
        ID:                "",
        Status:            "",
        Type:              repository.BigQuery,
        Strategy:          "",
        SourceProject:     "",
        LastScheduledTime: time.Time{},
        LastCleanupTime:   time.Time{},
        SinkOptions:       repository.SinkOptions{},
        SnapshotOptions:   repository.SnapshotOptions{},
        BackupOptions:     repository.BackupOptions{},
        EntityAudit:       repository.EntityAudit{},
    }

    // no entries
    err := validateIntersection(ctx, backup)
    assert.Nil(t, err, "expected no error")
    // table defined
    backup.Table = append(backup.Table, "t1")
    backup.Table = append(backup.Table, "t2")
    backup.Table = append(backup.Table, "t3")
    err = validateIntersection(ctx, backup)
    assert.Nil(t, err, "expected no error")
    // table defined + exclude fined
    backup.ExcludedTables = append(backup.ExcludedTables, "e1")
    err = validateIntersection(ctx, backup)
    assert.Nil(t, err, "expected no error")
    // table defined + exclude fined with intersection
    backup.ExcludedTables = append(backup.Table, "t1")
    err = validateIntersection(ctx, backup)
    assert.NotNil(t, err, "expected error")
}

func TestCreatingProcessor_validateIntersection_CloudStorage(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := &repository.Backup{
        ID:                "",
        Status:            "",
        Type:              repository.CloudStorage,
        Strategy:          "",
        SourceProject:     "",
        LastScheduledTime: time.Time{},
        LastCleanupTime:   time.Time{},
        SinkOptions:       repository.SinkOptions{},
        SnapshotOptions:   repository.SnapshotOptions{},
        BackupOptions:     repository.BackupOptions{},
        EntityAudit:       repository.EntityAudit{},
    }

    // no entries
    err := validateIntersection(ctx, backup)
    assert.Nil(t, err, "expected no error")
    // table defined
    backup.IncludePath = append(backup.IncludePath, "t1")
    backup.IncludePath = append(backup.IncludePath, "t2")
    backup.IncludePath = append(backup.IncludePath, "t3")
    err = validateIntersection(ctx, backup)
    assert.Nil(t, err, "expected no error")
    // table defined + exclude fined
    backup.ExcludePath = append(backup.ExcludePath, "e1")
    err = validateIntersection(ctx, backup)
    assert.Nil(t, err, "expected no error")
    // table defined + exclude fined with intersection
    backup.ExcludePath = append(backup.IncludePath, "t1")
    err = validateIntersection(ctx, backup)
    assert.NotNil(t, err, "expected error")
}

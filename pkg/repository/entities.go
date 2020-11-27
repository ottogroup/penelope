package repository

import (
	"fmt"
	"time"
)

// EntityAudit defines changes that happened to a given entity
type EntityAudit struct {
	CreatedTimestamp time.Time `pg:"audit_created_timestamp"`
	UpdatedTimestamp time.Time `pg:"audit_updated_timestamp"`
	DeletedTimestamp time.Time `pg:"audit_deleted_timestamp"`
}

// Group for a project
type Group struct {
	ID    int
	Email string
	EntityAudit
}

// UserGroup defines relation between User and Group
type UserGroup struct {
	UserID  int
	GroupID int
}

// User is an app user
type User struct {
	ID        int `pg:",pk"`
	Email     string
	Active    bool
	AuthToken string `pg:"authtoken"`
	EntityAudit
}

// Backup core entity
type Backup struct {
    //lint:ignore U1000 makes sure to have correct table name
    tableName struct{} `pg:"backups,alias:b"`

	ID     string       `pg:"id,pk"`
	Status BackupStatus `pg:"status"`
	Type   BackupType   `pg:"type"`

	Strategy          Strategy
	SourceProject     string    `pg:"project"`
	LastScheduledTime time.Time `pg:"last_scheduled_timestamp"`
	LastCleanupTime   time.Time `pg:"last_cleanup_timestamp"`

	SinkOptions
	SnapshotOptions
	BackupOptions
	EntityAudit
	MirrorOptions
}

// GetTrashcanPath give a patho to object moved into trashcan
func (b Backup) GetTrashcanPath() string {
	return fmt.Sprintf(".trashcan_%s", b.ID)
}

func (b Backup) String() string {
	snapshotOptionsString := ""
	if Snapshot == b.Strategy {
		snapshotOptionsString += fmt.Sprintf("snapshotOptions={lifetimeInDays=%d frequencyInHours=%d} ", b.SnapshotOptions.LifetimeInDays, b.FrequencyInHours)
	}

	backupOptionsString := ""
	if BigQuery == b.Type {
		backupOptionsString += fmt.Sprintf("bigQueryOptions={dataset=%s tables=%v, excluded_tables=%v} ", b.Dataset, b.Table, b.ExcludedTables)
	} else if CloudStorage == b.Type {
		backupOptionsString += fmt.Sprintf("cloudStorageOptions={bucket=%s includePath=%s excludePath=%s} ", b.Bucket, b.IncludePath, b.ExcludePath)
	}

	sinkOptions := fmt.Sprintf("targetProject=%s region=%s sink=%s storageClass=%s", b.TargetProject, b.Region, b.Sink, b.StorageClass)

	return fmt.Sprintf("backupID=%s type=%s status=%s strategy=%s sourceProject=%s lastScheduledTime=%q sinkOptions={%s} "+
		"createdTimestamp=%q updatedTimestamp=%q deletedTimestamp=%q %s%s",
		b.ID, b.Type, b.Status, b.Strategy, b.SourceProject, b.LastScheduledTime, sinkOptions,
		b.CreatedTimestamp, b.UpdatedTimestamp, b.DeletedTimestamp, snapshotOptionsString, backupOptionsString)
}

// SinkOptions for a backup
type SinkOptions struct {
	TargetProject string
	Region        string `pg:"target_region"`
	Sink          string `pg:"target_sink"`
	StorageClass  string `pg:"target_storage_class"`
	ArchiveTTM    uint   `pg:"archive_ttm"`
}

// BackupOptions backup options for specific technology
type BackupOptions struct {
	BigQueryOptions
	CloudStorageOptions
}

// SnapshotOptions strategy backup options
type SnapshotOptions struct {
	LifetimeInDays   uint `pg:"snapshot_lifetime_in_days,use_zero"`
	FrequencyInHours uint `pg:"snapshot_frequency_in_hours,use_zero"`
}

// MirrorOptions strategy backup options
type MirrorOptions struct {
	LifetimeInDays uint `pg:"mirror_lifetime_in_days,use_zero"`
}

// BigQueryOptions for a BigQuery backup
type BigQueryOptions struct {
	Dataset        string   `pg:"bigquery_dataset"`
	Table          []string `pg:"bigquery_table"`
	ExcludedTables []string `pg:"bigquery_excluded_tables"`
}

// CloudStorageOptions for a GCS backup
type CloudStorageOptions struct {
	Bucket      string   `pg:"cloudstorage_bucket"`
	IncludePath []string `pg:"cloudstorage_include_path"`
	ExcludePath []string `pg:"cloudstorage_exclude_path"`
}

// ExtractJobID  for GCS technology
type ExtractJobID string

// TransferJobID  for BigQuery technology
type TransferJobID string

func (j ExtractJobID) String() string {
	return string(j)
}

func (j TransferJobID) String() string {
	return string(j)
}

// ForeignJobID job id for a specific technology
type ForeignJobID struct {
	BigQueryID     ExtractJobID  `pg:"bigquery_extract_job_id"`
	CloudStorageID TransferJobID `pg:"cloudstorage_transfer_job_id"`
}

// Job a backup unit of work
type Job struct {
    //lint:ignore U1000 makes sure to have correct table name
    tableName struct{} `pg:"jobs,alias:j"`

	ID       string     `pg:"id,pk"`
	BackupID string     `pg:"backup_id"`
	Type     BackupType `pg:"type"`
	Status   JobStatus  `pg:"status"`
	Source   string     `pg:"source"`
	ForeignJobID
	EntityAudit
}

func (j Job) String() string {
	foreignJobIDString := "ForeignJobID="
	if j.ForeignJobID.BigQueryID != "" {
		foreignJobIDString += fmt.Sprintf("BigQueryExtractJobID:%s", j.ForeignJobID.BigQueryID)
	} else if j.ForeignJobID.CloudStorageID != "" {
		foreignJobIDString += fmt.Sprintf("CloudStorageTransferJobID:%s", j.ForeignJobID.CloudStorageID)
	}

	return fmt.Sprintf("backupID=%s jobID=%s type=%s status=%s source=%s createdTimestamp=%q updatedTimestamp=%q deletedTimestamp=%q %s",
		j.BackupID, j.ID, j.Type, j.Status, j.Source, j.CreatedTimestamp, j.UpdatedTimestamp, j.DeletedTimestamp, foreignJobIDString)
}

// SourceMetadata for a BigQuery mirroring
type SourceMetadata struct {
    //lint:ignore U1000 makes sure to have correct table name
	tableName struct{} `pg:"source_metadata,alias:sm"`

	ID             int    `pg:"id,pk"`
	BackupID       string `pg:"backup_id"`
	Source         string `pg:"source"`
	SourceChecksum string `pg:"source_checksum"`
	Operation      string `pg:"operation"`

	CreatedTimestamp time.Time `pg:"audit_created_timestamp"`
	DeletedTimestamp time.Time `pg:"audit_deleted_timestamp"`
}

// SourceMetadataJob for a BigQuery mirroring
type SourceMetadataJob struct {
    //lint:ignore U1000 makes sure to have correct table name
	tableName struct{} `pg:"source_metadata_jobs,alias:smj"`

	SourceMetadataID int    `pg:"source_metadata_id"`
	JobId            string `pg:"job_id"`
}

// SourceTrashcan holds information about objects moved into trashcan
type SourceTrashcan struct {
    //lint:ignore U1000 makes sure to have correct table name
    tableName struct{} `pg:"source_trashcan,alias:st"`

    BackupID         string
    Source           string
    CreatedTimestamp time.Time `pg:"audit_created_timestamp"`
}

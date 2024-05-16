package repository

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
)

// BackupStatus for backup
type BackupStatus string

// JobStatus for backup
type JobStatus string

// Operation for a backup
type Operation string

// BackupType for a backup
type BackupType string

// InvalidBackupType for cases when backup type is incorrect
type InvalidBackupType struct {
	Type BackupType
}

func (i *InvalidBackupType) Error() string {
	return fmt.Sprintf("invalid backup type: %s", i.Type)
}

// Strategy for a backup
type Strategy string

// Region for a GCS sink bucket
type Region string

// StorageClass for a GCS sink bucket
type StorageClass string

const (
	// Snapshot will make a ontime or recurring snapshot
	Snapshot Strategy = "Snapshot"
	// Mirror data actively
	Mirror Strategy = "Mirror"
)
const (
	// BigQuery type
	BigQuery BackupType = "BigQuery"
	// CloudStorage type
	CloudStorage BackupType = "CloudStorage"
)

const (
	// NotScheduled job is not scheduled
	NotScheduled JobStatus = "NotScheduled"
	// Scheduled is scheduled
	Scheduled JobStatus = "Scheduled"
	// Pending BigQuery/GCS job is ongoing
	Pending JobStatus = "Pending"
	// Error job finished with error
	Error JobStatus = "Error"
	// FinishedOk job finished with success
	FinishedOk JobStatus = "FinishedOk"
	// FinishedError job finished with error
	FinishedError JobStatus = "FinishedError"
	// FinishedQuotaError job finished with quota errir
	FinishedQuotaError JobStatus = "FinishedQuotaError"
	// JobDeleted was deleted
	JobDeleted JobStatus = "JobDeleted"
)

const (
	// NotStarted for a newly created backup
	NotStarted BackupStatus = "NotStarted"
	// Prepared backup had jobs prepared
	Prepared BackupStatus = "Prepared"
	// Finished backup was successful (for requring backup it will stay in that state unless error apreas)
	Finished BackupStatus = "Finished"
	// Paused backup will not schedule new jobs
	Paused BackupStatus = "Paused"
	// ToDelete was marked to deletion
	ToDelete BackupStatus = "ToDelete"
	// BackupDeleted was deleted
	BackupDeleted BackupStatus = "BackupDeleted"
	// BackupSourceDeleted was deleted
	BackupSourceDeleted BackupStatus = "BackupSourceDeleted"
)

const (
	// Add new backup
	Add Operation = "Add"
	// Update backup
	Update Operation = "Update"
	// Delete backup
	Delete Operation = "Delete"
)

// Strategies for a backups
var Strategies = []Strategy{Snapshot, Mirror}

// BackupTypes source for a backup
var BackupTypes = []BackupType{BigQuery, CloudStorage}

// JobStatutses available job statuses
var JobStatutses = []JobStatus{NotScheduled, Scheduled, Error, Pending, FinishedOk, FinishedError, FinishedQuotaError, JobDeleted}

func (s BackupType) String() string {
	return string(s)
}

// EqualTo compare string with the BackupType type
func (s BackupType) EqualTo(backupType string) bool {
	return strings.EqualFold(backupType, s.String())
}

func (bs JobStatus) String() string {
	return string(bs)
}

func (bs BackupStatus) String() string {
	return string(bs)
}

// EqualTo compare string with the Operation type
func (bs BackupStatus) EqualTo(status string) bool {
	return strings.EqualFold(status, bs.String())
}

func (o Operation) String() string {
	return string(o)
}

// EqualTo compare string with the Operation type
func (o Operation) EqualTo(status string) bool {
	return strings.EqualFold(status, o.String())
}

func (s Strategy) String() string {
	return string(s)
}

// EqualTo compare string with the Strategy type
func (s Strategy) EqualTo(strategy string) bool {
	return strings.EqualFold(strategy, s.String())
}

func (s Region) String() string {
	return string(s)
}

// EqualTo compare string with the Region type
func (s Region) EqualTo(region string) bool {
	return strings.EqualFold(region, s.String())
}

func (s StorageClass) String() string {
	return string(s)
}

// EqualTo compare string with the StorageClass type
func (s StorageClass) EqualTo(storageClass string) bool {
	return strings.EqualFold(storageClass, s.String())
}

// MirrorRevision track changes for a BigQuery tables
type MirrorRevision struct {
	SourceMetadataID int
	JobID            string
	BackupID         string
	BigqueryDataset  string
	Source           string
	TargetProject    string
	TargetSink       string
}

func (b MirrorRevision) String() string {
	return fmt.Sprintf("backupID=%s jobID=%s sourceMetadataId=%d bigqueryDataset=%s source=%s targetProject=%s targetSink=%s",
		b.BackupID, b.JobID, b.SourceMetadataID, b.BigqueryDataset, b.Source, b.TargetProject, b.TargetSink)
}

// BuildStoragePath create a path for BigQuery dataset/table
func BuildStoragePath(dataset, table string) string {
	if table != "" {
		return fmt.Sprintf("dataset/%s/table/%s", dataset, table)
	}
	return fmt.Sprintf("dataset/%s", dataset)
}

// BuildFullObjectStoragePath create a sink's path for a GCS data
func BuildFullObjectStoragePath(sink, dataset, table, jobID string) string {
	return fmt.Sprintf("gs://%s/%s/%s-*.avro", sink, BuildStoragePath(dataset, table), jobID)
}

// BuildObjectStoragePathPattern create a sink's path for a BigQuery data
func BuildObjectStoragePathPattern(dataset, table, jobID string) string {
	return fmt.Sprintf("%s/%s-.*.avro", BuildStoragePath(dataset, table), jobID)
}

func logQueryError(source string, err error, args ...interface{}) {
	glog.Errorf("%s had error: %s. args: %v", source, err, args)
}

// JobPatch defines update fields for a Job
type JobPatch struct {
	ID     string
	Status JobStatus
	ForeignJobID
}

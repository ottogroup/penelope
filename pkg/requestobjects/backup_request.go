package requestobjects

import (
	"github.com/ottogroup/penelope/pkg/provider"
	"time"
)

type AvailabilityClass string

const (
	A0Invalid    AvailabilityClass = ""
	A1Irrelevant AvailabilityClass = "A1"
	A2Aimed      AvailabilityClass = "A2"
	A3Guaranteed AvailabilityClass = "A3"
	A4Resilient  AvailabilityClass = "A4"
)

func (AvailabilityClass) ValidValues() []AvailabilityClass {
	return []AvailabilityClass{A1Irrelevant, A2Aimed, A3Guaranteed, A4Resilient}
}

// Page is used for a subset selection
type Page struct {
	Size   int
	Number int
}

// Note: the required tag is only for documentation but has no influence on unmarshalling from json

// ListRequest list backups
type ListRequest struct {
	Project string
}

// GetRequest get backup details
type GetRequest struct {
	BackupID string
	Page     Page
}

// DeleteRequest remove bucket an all files within next 60 days
type DeleteRequest struct {
	BackupID string
}

// RestoreRequest get instruction for a backup restoration
// only BigQuery is supported
type RestoreRequest struct {
	BackupID          string
	JobIDForTimestamp string
}

// UpdateRequest change backup
type UpdateRequest struct {
	BackupID               string `json:"backup_id"`
	Status                 string `json:"status,omitempty"`
	MirrorTTL              uint   `json:"mirror_ttl,omitempty"`
	SnapshotTTL            uint   `json:"snapshot_ttl,omitempty"`
	ArchiveTTM             uint   `json:"archive_ttm"`
	RecoveryPointObjective int    `json:"recovery_point_objective,omitempty"`
	RecoveryTimeObjective  int    `json:"recovery_time_objective,omitempty"`
	// only for GCS backups
	IncludePath []string `json:"include_path,omitempty"`
	ExcludePath []string `json:"exclude_path,omitempty"`
	// only for BigQuery backups
	Table          []string `json:"table,omitempty"`
	ExcludedTables []string `json:"excluded_tables,omitempty"`
}

// CreateRequest make a new backup
type CreateRequest struct {
	Type                   string        `json:"type,omitempty"`
	Strategy               string        `json:"strategy,omitempty"`
	Project                string        `json:"project,omitempty"`
	RecoveryPointObjective int           `json:"recovery_point_objective"`
	RecoveryTimeObjective  int           `json:"recovery_time_objective"`
	TargetOptions          TargetOptions `json:"target,omitempty"`

	SnapshotOptions SnapshotOptions `json:"snapshot_options,omitempty"`
	MirrorOptions   MirrorOptions   `json:"mirror_options,omitempty"`
	BigQueryOptions BigQueryOptions `json:"bigquery_options,omitempty"`
	GCSOptions      GCSOptions      `json:"gcs_options,omitempty"`
}

// BigQueryOptions specify backup for a source BigQuery datast or table(s)
type BigQueryOptions struct {
	Dataset        string   `json:"dataset,omitempty"`
	Table          []string `json:"table,omitempty"`
	ExcludedTables []string `json:"excluded_tables,omitempty"`
}

// GCSOptions specify backup for a source bucket
type GCSOptions struct {
	Bucket      string   `json:"bucket,omitempty"`
	IncludePath []string `json:"include_prefixes,omitempty"`
	ExcludePath []string `json:"exclude_prefixes,omitempty"`
}

// SnapshotOptions specify backup snapshot options
type SnapshotOptions struct {
	LifetimeInDays   uint   `json:"lifetime_in_days,omitempty"`
	FrequencyInHours uint   `json:"frequency_in_hours,omitempty"`
	LastScheduled    string `json:"last_scheduled,omitempty"`
}

// MirrorOptions specify backup mirror options
type MirrorOptions struct {
	LifetimeInDays uint `json:"lifetime_in_days,omitempty"`
}

// TargetOptions specify backup sink options
type TargetOptions struct {
	Region         string `json:"region,omitempty"`
	DualRegion     string `json:"dual_region,omitempty"`
	StorageClass   string `json:"storage_class,omitempty"`
	LifecycleCount uint   `json:"lifecycle_count,omitempty"`
	ArchiveTTM     uint   `json:"archive_ttm"`
}

// ListingResponse response for a ListRequest
type ListingResponse struct {
	Backups []BackupResponse `json:"backups"`
}

// BackupResponse get backup details
type BackupResponse struct {
	ID string `json:"id"`
	CreateRequest

	Status                string                     `json:"status"`
	Sink                  string                     `json:"sink"`
	SinkProject           string                     `json:"sink_project"`
	DataOwner             string                     `json:"data_owner"`
	DataAvailabilityClass provider.AvailabilityClass `json:"data_availability_class"`

	CreatedTimestamp string `json:"created,omitempty"`
	UpdatedTimestamp string `json:"updated,omitempty"`
	DeletedTimestamp string `json:"deleted,omitempty"`

	Jobs                  []JobResponse `json:"jobs,omitempty"`
	JobsTotal             uint64        `json:"jobs_total,omitempty"`
	TrashcanCleanupStatus string        `json:"trashcan_cleanup_status,omitempty"`
}

// JobResponse get backup job details
type JobResponse struct {
	ID           string `json:"id"`
	BackupID     string `json:"backup_id"`
	ForeignJobID string `json:"foreign_job_id,omitempty"`

	Status string `json:"status"`
	Source string `json:"source"`

	CreatedTimestamp string `json:"created,omitempty"`
	UpdatedTimestamp string `json:"updated,omitempty"`
	DeletedTimestamp string `json:"deleted,omitempty"`
}

// UpdateResponse response for a UpdateRequest
type UpdateResponse struct {
	UpdateRequest

	CreatedTimestamp string `json:"created,omitempty"`
	UpdatedTimestamp string `json:"updated,omitempty"`
	DeletedTimestamp string `json:"deleted,omitempty"`
}

// DeleteResponse response for a UpdateRequest
type DeleteResponse struct {
	DeleteRequest

	Status string `json:"status,omitempty"`

	CreatedTimestamp string `json:"created,omitempty"`
	UpdatedTimestamp string `json:"updated,omitempty"`
	DeletedTimestamp string `json:"deleted,omitempty"`
}

// RestoreAction request instruction for a backup restoration
// currently only BigQuery is supported
type RestoreAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
}

// RestoreResponse response for a RestoreAction request
type RestoreResponse struct {
	BackupID       string          `json:"backup_id"`
	RestoreActions []RestoreAction `json:"actions"`
}

// CalculateRequest request cost calculation for a backup
type CalculateRequest struct {
	CreateRequest
}

// CalculatedResponse response for a CalculateRequest request
type CalculatedResponse struct {
	Costs []*Cost `json:"costs"`
}

// ComplianceRequest request compliance check for a backup
type ComplianceRequest struct {
	CreateRequest
}

// ComplianceResponse response for a ComplianceRequest request
type ComplianceResponse struct {
	Checks []ComplianceCheck `json:"checks"`
}

// ComplianceCheck response for a ComplianceRequest request
type ComplianceCheck struct {
	Field       string `json:"field"`
	Passed      bool   `json:"passed"`
	Description string `json:"description"`
	Details     string `json:"details"`
}

// Cost represent backup data price in a given month
type Cost struct {
	Cost        float64 `json:"cost"`
	Currency    string  `json:"currency"`
	Name        string  `json:"name"`
	Period      int64   `json:"period"`
	SizeInBytes int64   `json:"size_in_bytes"`
}

// DatasetListRequest request datasets list
type DatasetListRequest struct {
	Project string `json:"project"`
}

// DatasetListResponse response for a BucketListRequest request
type DatasetListResponse struct {
	Datasets []string `json:"datasets"`
}

// BucketListRequest request bucket list
type BucketListRequest struct {
	Project string `json:"project"`
}

// BucketListResponse response for a BucketListRequest request
type BucketListResponse struct {
	Buckets []string `json:"buckets"`
}

// SourceProjectGetRequest request source project get
type SourceProjectGetRequest struct {
	Project string `json:"project"`
}

// SourceProjectGetResponse response for a SourceProjectGetRequest request
type SourceProjectGetResponse struct {
	SourceProject provider.SourceGCPProject `json:"source_project"`
}

// EmptyRequest request without any parameters
type EmptyRequest struct {
}

// RegionsListResponse response for a region list request
type RegionsListResponse struct {
	Regions []string `json:"regions"`
}

// StorageClassListResponse response for a storage class list request
type StorageClassListResponse struct {
	StorageClasses []string `json:"storage_classes"`
}

type ProjectSinkComplianceCheck struct {
	Project   string    `json:"project"`
	Compliant bool      `json:"compliant"`
	Reasons   []string  `json:"reasons"`
	LastCheck time.Time `json:"last_check"`
}

type ProjectSinkComplianceRequest struct {
}

type ProjectSinkComplianceResponse struct {
	Checks []ProjectSinkComplianceCheck `json:"checks"`
}

type TrashcanCleanUpRequest struct {
	BackupID string `json:"backup_id"`
}

type TrashcanCleanUpResponse struct {
}

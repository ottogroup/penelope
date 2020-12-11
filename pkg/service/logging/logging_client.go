package logging

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"go.opencensus.io/trace"
	"google.golang.org/genproto/googleapis/cloud/audit"
	"google.golang.org/grpc"
	"regexp"
	"time"

	"cloud.google.com/go/logging/apiv2"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	loggingpb "google.golang.org/genproto/googleapis/logging/v2"
)

const defaultAPIScope = "https://www.googleapis.com/auth/logging.read"

// DefaultLoggingClient represent logging client
type DefaultLoggingClient struct {
	client          *logging.Client
	ctx             context.Context
	srcProjectID    string
	targetProjectID string
}

// NewLoggingClient creates new DefaultLoggingClient
func NewLoggingClient(ctxIn context.Context, targetPrincipalProvider impersonate.TargetPrincipalForProjectProvider, srcProjectID, targetProjectID string) (*DefaultLoggingClient, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewLoggingClient")
	defer span.End()

	target, err := targetPrincipalProvider.GetTargetPrincipalForProject(ctx, targetProjectID)
	if err != nil {
		return nil, err
	}

	options := []option.ClientOption{
		option.WithScopes(defaultAPIScope),
		option.ImpersonateCredentials(target),
	}

	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		options = append(options, option.WithGRPCDialOption(grpc.WithInsecure()))
	}

	client, err := logging.NewClient(ctx, options...)
	if err != nil {
		return &DefaultLoggingClient{}, fmt.Errorf("failed to create client: %s", err)
	}

	return &DefaultLoggingClient{client: client, ctx: ctx, srcProjectID: srcProjectID, targetProjectID: targetProjectID}, nil
}

// ObjectEvent represent change of bucket object
type ObjectEvent string

// Create object was created
const Create ObjectEvent = "Add"

// Delete objects was deleted
const Delete ObjectEvent = "Delete"

// BucketObjectEvent represent a bucket event log
type BucketObjectEvent struct {
	ResourceName string
	ObjectName   string
	Timestamp    time.Time
	Type         ObjectEvent
}

func (b BucketObjectEvent) String() string {
	return fmt.Sprintf("resourceName=%s objectName=%s type=%s timestamp=%q", b.ResourceName, b.ObjectName, b.Type, b.Timestamp)
}

// IterateOverBucketObjectEvents reads events log for a given bucket
func (l *DefaultLoggingClient) IterateOverBucketObjectEvents(ctxIn context.Context, backup *repository.Backup, bucketName string, timestampStart time.Time, iterationDeadline time.Time, consumeFunc func(obj []BucketObjectEvent, eventType ObjectEvent) error) (time.Time, error) {
	timestampEnd := backup.LastScheduledTime

	filter := fmt.Sprintf("logName=\"projects/%s/logs/cloudaudit.googleapis.com%%2Fdata_access\" AND "+
		"resource.type=\"gcs_bucket\" AND resource.labels.bucket_name=\"%s\" AND "+
		"protoPayload.methodName=(\"storage.objects.delete\" OR \"storage.objects.create\") AND  "+
		"NOT protoPayload.resourceName:\".temp-beam\" AND  "+
		"timestamp > \"%s\" AND timestamp < \"%s\"", l.srcProjectID, bucketName,
		timestampStart.Format(time.RFC3339Nano), timestampEnd.Format(time.RFC3339Nano))

	includePaths := prepareBucketPaths(backup.SinkOptions.Sink, backup.CloudStorageOptions.IncludePath)
	if 0 < len(includePaths) {
		filter += fmt.Sprintf(" AND (%s)", includePaths)
	}
	excludePaths := prepareBucketPaths(backup.SinkOptions.Sink, backup.CloudStorageOptions.ExcludePath)
	if 0 < len(excludePaths) {
		filter += fmt.Sprintf(" AND (NOT (%s))", excludePaths)
	}
	const PageSize = 1000
	pb := &loggingpb.ListLogEntriesRequest{
		ResourceNames: []string{fmt.Sprintf("projects/%s", l.srcProjectID)},
		OrderBy:       "timestamp asc", //this is necessary for integrity
		Filter:        filter,
		PageSize:      PageSize,
	}

	objectPattern := regexp.MustCompile(fmt.Sprintf("projects/_/buckets/%s/objects/(.*)", bucketName))
	it := l.client.ListLogEntries(ctxIn, pb)

	lastTimestamp := timestampStart
	iterationStartTime := time.Now()
	var eventsBatched []BucketObjectEvent
	lastEventType := Create

	const MaxBatchEvents = 2000
	const QuotaPeriodOneMinuteInSeconds = 60
	// logging quota for read request is 60 requests per minute per user, each page fetch 1000 elements
	const QuotaPerMinute = PageSize * 15
	loggingQuota := newLoggingQuota(QuotaPeriodOneMinuteInSeconds, QuotaPerMinute)
	for {
		if time.Now().After(iterationDeadline) {
			glog.Info("Iteration over objects deadline was met. Stop processing new GCS events.")
			break
		}
		if loggingQuota.IsReached() {
			loggingQuota.WaitUntilNextPeriod()
		}
		resp, err := it.Next()
		loggingQuota.Increment()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return lastTimestamp, errors.Wrap(err, "could not iterate over next event")
		}
		logEntryProtoPayload := resp.GetProtoPayload()
		if nil == logEntryProtoPayload {
			return lastTimestamp, errors.Errorf("protoPayload is nil for logEntry")
		}
		auditLog := audit.AuditLog{}
		err = ptypes.UnmarshalAny(logEntryProtoPayload, &auditLog)
		if err != nil {
			return lastTimestamp, errors.Wrapf(err, "could not UnmarshalAny log entry field protoPayload `%v` to ", logEntryProtoPayload.Value)
		}
		if auditLog.MethodName == "" || auditLog.ResourceName == "" {
			// MethodName && ResourceName are always present thus can't be empty
			return lastTimestamp, errors.Wrapf(err, "MethodName[%s] or ResourceName[%s] is empty", auditLog.MethodName, auditLog.ResourceName)
		}
		objectName := objectPattern.FindStringSubmatch(auditLog.ResourceName)
		if len(objectName) > 1 {
			var eventType ObjectEvent
			if auditLog.MethodName == "storage.objects.delete" {
				eventType = Delete
			} else if auditLog.MethodName == "storage.objects.create" {
				eventType = Create
			}

			eventTimestamp := time.Unix(resp.Timestamp.Seconds, int64(resp.Timestamp.Nanos)).UTC()

			event := BucketObjectEvent{
				ResourceName: auditLog.ResourceName,
				ObjectName:   objectName[len(objectName)-1],
				Timestamp:    eventTimestamp,
				Type:         eventType,
			}
			if event.Type != lastEventType || len(eventsBatched) > MaxBatchEvents {
				err = consumeFunc(eventsBatched, lastEventType)
				if err != nil {
					return lastTimestamp, errors.Wrap(err, "error during consuming event")
				}
				eventsBatched = []BucketObjectEvent{}
				lastTimestamp = eventTimestamp
				lastEventType = event.Type
			}
			eventsBatched = append(eventsBatched, event)
		}
	}
	if 0 < len(eventsBatched) {
		err := consumeFunc(eventsBatched, lastEventType)
		if err != nil {
			return lastTimestamp, errors.Wrap(err, "error during consuming event")
		}
		lastTimestamp = eventsBatched[len(eventsBatched)-1].Timestamp
	}
	if lastTimestamp == timestampStart {
		lastTimestamp = timestampEnd
	}
	if lastTimestamp.After(iterationStartTime) {
		lastTimestamp = iterationStartTime
	}

	return lastTimestamp, nil
}

// Close terminates terminates all resources in use
func (l *DefaultLoggingClient) Close() {
	l.client.Close()
}

/*
Track quota done for requests per minute.
*/
type requestPerPeriodQuota struct {
	maxRequestPerMinute uint64
	requestPerPeriod    uint64
	timePeriod          time.Duration
	quotaEnd            time.Time
}

func newLoggingQuota(periodInSeconds uint64, maxRequestPerPeriod uint64) *requestPerPeriodQuota {
	return &requestPerPeriodQuota{
		timePeriod:          time.Duration(periodInSeconds),
		maxRequestPerMinute: maxRequestPerPeriod,
		requestPerPeriod:    0,
		quotaEnd:            time.Now().Add(time.Second * time.Duration(periodInSeconds)),
	}
}

func (l *requestPerPeriodQuota) IsReached() bool {
	if time.Now().After(l.quotaEnd) {
		l.reset()
		return false
	}
	if l.requestPerPeriod < l.maxRequestPerMinute {
		return false
	}
	return true
}

func (l *requestPerPeriodQuota) Increment() {
	l.requestPerPeriod++
}

func (l *requestPerPeriodQuota) WaitUntilNextPeriod() {
	nanosecondsToNextQuotaPeriod := time.Duration(time.Until(l.quotaEnd).Nanoseconds())
	glog.Infof("Quota for read requests per minute reached. Waiting for the next quota period for %d nanoseconds. quota end %s",
		nanosecondsToNextQuotaPeriod, l.quotaEnd)
	time.Sleep(nanosecondsToNextQuotaPeriod)
	l.reset()
}

func (l *requestPerPeriodQuota) reset() {
	l.requestPerPeriod = 0
	l.quotaEnd = time.Now().Add(time.Second * l.timePeriod)
}

func prepareBucketPaths(bucket string, prefixes []string) (paths string) {
	if 0 < len(prefixes) {
		for index, prefix := range prefixes {
			paths += fmt.Sprintf("protoPayload.resourceName:\"projects/_/buckets/%s/objects/%s\"", bucket, prefix)
			if index < (len(prefixes) - 1) {
				paths += " OR "
			}
		}
	}
	return paths
}

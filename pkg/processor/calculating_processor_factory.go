package processor

import (
	"context"
	"fmt"

	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/ottogroup/penelope/pkg/service/billing"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

const oneGigiByteInBytes = 1073741824

// CalculatingProcessorFactory create Process for Calculating
type CalculatingProcessorFactory struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func NewCalculatingProcessorFactory(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) *CalculatingProcessorFactory {
	return &CalculatingProcessorFactory{
		backupProvider:      backupProvider,
		tokenSourceProvider: tokenSourceProvider,
	}
}

// DoMatchRequestType does request type match Calculating
func (c CalculatingProcessorFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
	return requestobjects.Calculating.EqualTo(requestType.String())
}

// CreateProcessor return instance of Operations for Calculating
func (c CalculatingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operations, error) {
	_, span := trace.StartSpan(ctxIn, "(*CalculatingProcessorFactory).CreateProcessor")
	defer span.End()

	return &calculatingProcessor{
		backupProvider:      c.backupProvider,
		tokenSourceProvider: c.tokenSourceProvider,
	}, nil
}

type calculatingProcessor struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

// Process request
func (c *calculatingProcessor) Process(ctxIn context.Context, args *Arguments) (*Result, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*calculatingProcessor).Process")
	defer span.End()

	var request *requestobjects.CalculateRequest
	if args.Request == nil {
		return nil, fmt.Errorf("nil request object for processing backup calculate request")
	}
	request, ok := args.Request.(*requestobjects.CalculateRequest)
	if !ok {
		return nil, fmt.Errorf("wrong request object for processing backup calculate request")
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Calculating, request.Project) {
		return nil, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Calculating.String(), args.Principal.User.Email, request.Project)
	}

	sourceProject := request.Project
	targetProject, err := c.backupProvider.GetSinkGCPProjectID(ctx, sourceProject)
	if err != nil {
		return nil, err
	}

	result := Result{}
	if repository.BigQuery.EqualTo(request.Type) {
		bigQueryCalculator, err := c.newBigQueryCalculator(ctx, sourceProject, targetProject)
		if err != nil {
			return nil, errors.Wrap(err, "newBigQueryCalculator failed")
		}
		calculateResponse, err := bigQueryCalculator.calculateCost(ctx, request)
		if err != nil {
			return nil, errors.Wrap(err, "bigQueryCalculator.calculateCost failed")
		}
		result.CalculateResponse = calculateResponse
	}
	if repository.CloudStorage.EqualTo(request.Type) {
		cloudStorageCalculator, err := c.newCloudStorageCalculator(ctx, targetProject)
		if err != nil {
			return nil, errors.Wrap(err, "newCloudStorageCalculator failed")
		}
		defer cloudStorageCalculator.storageClient.Close(ctx)
		calculateResponse, err := cloudStorageCalculator.calculateCost(ctx, request)
		if err != nil {
			return nil, errors.Wrap(err, "cloudStorageCalculator.calculateCost failed")
		}
		result.CalculateResponse = calculateResponse
	}
	return &result, nil
}

type baseCalculator struct {
	billingClient billing.Client
}

type cloudStorageCalculator struct {
	baseCalculator
	storageClient gcs.CloudStorageClient
}

type bigQueryCalculator struct {
	baseCalculator
	bigQueryClient bigquery.Client
}

func (c *calculatingProcessor) newCloudStorageCalculator(ctxIn context.Context, targetProjectID string) (*cloudStorageCalculator, error) {
	ctx, span := trace.StartSpan(ctxIn, "newCloudStorageCalculator")
	defer span.End()

	storageClient, err := gcs.NewCloudStorageClient(ctx, c.tokenSourceProvider, targetProjectID)
	if err != nil {
		return nil, errors.Wrap(err, "NewCloudStorageClient failed")
	}
	billingClient, err := billing.NewCloudBillingClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "NewCloudBillingClient failed")
	}
	CloudStorageCalculator := cloudStorageCalculator{baseCalculator: baseCalculator{billingClient: billingClient}, storageClient: storageClient}
	return &CloudStorageCalculator, nil
}

func (c *calculatingProcessor) newBigQueryCalculator(ctxIn context.Context, sourceProjectID, targetProjectID string) (*bigQueryCalculator, error) {
	ctx, span := trace.StartSpan(ctxIn, "newBigQueryCalculator")
	defer span.End()

	bigQueryClient, err := bigquery.NewBigQueryClient(ctx, c.tokenSourceProvider, sourceProjectID, targetProjectID)
	if err != nil {
		return nil, errors.Wrap(err, "NewBigQueryClient failed")
	}
	billingClient, err := billing.NewCloudBillingClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "NewCloudBillingClient failed")
	}
	BigQueryCalculator := bigQueryCalculator{baseCalculator: baseCalculator{billingClient: billingClient}, bigQueryClient: bigQueryClient}
	return &BigQueryCalculator, nil
}

func (c *cloudStorageCalculator) calculateCost(ctxIn context.Context, request *requestobjects.CalculateRequest) (*requestobjects.CalculatedResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*cloudStorageCalculator).calculateCost")
	defer span.End()

	response := requestobjects.CalculatedResponse{}
	storageSize, err := c.storageClient.BucketUsageInBytes(ctx, request.Project, request.GCSOptions.Bucket)
	if err != nil {
		return nil, errors.Wrap(err, "getTotalStorageSize failed")
	}
	response.Costs, err = c.calculateCosts(request, storageSize)
	return &response, err
}

func (c *bigQueryCalculator) calculateCost(ctxIn context.Context, request *requestobjects.CalculateRequest) (*requestobjects.CalculatedResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*bigQueryCalculator).calculateCost")
	defer span.End()

	response := requestobjects.CalculatedResponse{}
	storageSize, err := c.getTotalStorageSize(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "getTotalStorageSize failed")
	}
	response.Costs, err = c.calculateCosts(request, storageSize)
	return &response, err
}

func (c *bigQueryCalculator) getTotalStorageSize(ctxIn context.Context, request *requestobjects.CalculateRequest) (totalSize float64, err error) {
	ctx, span := trace.StartSpan(ctxIn, "(*bigQueryCalculator).getTotalStorageSize")
	defer span.End()

	if 0 < len(request.BigQueryOptions.Table) {
		for _, tableName := range request.BigQueryOptions.Table {
			table, err := c.bigQueryClient.GetTable(ctx, request.Project, request.BigQueryOptions.Dataset, tableName)
			if err != nil {
				return 0, errors.Wrap(err, fmt.Sprintf("GetTable failed for project %s and dataset %s and tableName %s", request.Project, request.BigQueryOptions.Dataset, tableName))
			}
			if table == nil {
				return 0, errors.New(fmt.Sprintf("table not exist %s for project %s and dataset %s", tableName, request.Project, request.BigQueryOptions.Dataset))
			}
			totalSize += table.SizeInBytes
		}
	} else {
		datasetSize, err := c.calculateBigQueryDatasetSize(ctx, request.Project, request.BigQueryOptions.Dataset)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("calculateBigQueryDatasetSize failed for project %s and dataset %s", request.Project, request.BigQueryOptions.Dataset))
		}
		totalSize += datasetSize
	}
	return totalSize, nil
}

func (c *bigQueryCalculator) calculateBigQueryDatasetSize(ctxIn context.Context, project string, dataset string) (datasetSize float64, err error) {
	ctx, span := trace.StartSpan(ctxIn, "(*bigQueryCalculator).calculateBigQueryDatasetSize")
	defer span.End()

	tables, err := c.bigQueryClient.GetTablesInDataset(ctx, project, dataset)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("GetTablesInDataset failed for project %s and dataset %s.", project, dataset))
	}
	for _, table := range tables {
		datasetSize += table.SizeInBytes
	}
	return datasetSize, nil
}

func (c *baseCalculator) calculateCosts(request *requestobjects.CalculateRequest, storageSizeInBytes float64) (costs []*requestobjects.Cost, err error) {
	storageClass := request.TargetOptions.StorageClass
	if storageClass == "" {
		storageClass = config.DefaultBucketStorageClass.MustGet()
	}
	var storageCosts []StorageConfiguration
	if request.TargetOptions.DualRegion == "" {
		storageCosts = append(storageCosts, getStorageCost(storageClass, request.TargetOptions.Region, false))
	} else {
		storageCosts = append(storageCosts,
			getStorageCost(storageClass, request.TargetOptions.Region, true),
			getStorageCost(storageClass, request.TargetOptions.DualRegion, true),
		)
	}

	var (
		minTTL                     int64
		storagePricePerGBMonth     float64
		earlyDeletePricePerGBMonth float64
		frequencyPerMonth          float64
		periods                    []int64
	)
	for _, cost := range storageCosts {
		if cost.StorageSKU != "" {
			price, err := c.billingClient.PricePerMonth(cost.StorageSKU)
			if err != nil {
				return costs, errors.Wrap(err, "pricePerMonth failed")
			}
			storagePricePerGBMonth += price
		}
		if cost.EarlyDeleteSKU != "" {
			price, err := c.billingClient.PricePerMonth(cost.EarlyDeleteSKU)
			if err != nil {
				return costs, errors.Wrap(err, "pricePerMonth failed")
			}
			earlyDeletePricePerGBMonth += price
		}
		if cost.MinTTL > minTTL {
			minTTL = cost.MinTTL
		}
	}

	if repository.Snapshot.EqualTo(request.Strategy) && request.SnapshotOptions.LifetimeInDays != 0 {
		periods = append(periods, int64(request.SnapshotOptions.LifetimeInDays))
	} else {
		periods = append(periods, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}...)
	}
	frequencyPerMonth = 1.0
	averageGregorianDaysInMonth := 365.2425 / 12
	if repository.Snapshot.EqualTo(request.Strategy) && 0 < request.SnapshotOptions.FrequencyInHours {
		frequencyPerMonth = 24 * averageGregorianDaysInMonth / float64(request.SnapshotOptions.FrequencyInHours)
	}

	var costFraction float64
	for _, period := range periods {
		costFraction = 0
		storageCost := requestobjects.Cost{
			Cost:        0.0,
			Currency:    "EUR",
			Name:        "Storage",
			Period:      period,
			SizeInBytes: int64(storageSizeInBytes),
		}
		if period < minTTL {
			fmt.Printf("Period: %d Storage: %f, EarlyDelete: %f Min TTL %d\n", period, storagePricePerGBMonth, earlyDeletePricePerGBMonth, minTTL)
			fmt.Println(float64(period))
			fmt.Println(float64(minTTL - period))

			costFraction += storagePricePerGBMonth * float64(period)
			costFraction += earlyDeletePricePerGBMonth * float64(minTTL-period)
		} else {
			fmt.Printf("Period: %d Storage: %f, EalryDelete: %f\n", period, storagePricePerGBMonth, earlyDeletePricePerGBMonth)
			costFraction += storagePricePerGBMonth * float64(period)
		}

		storageCost.Cost += costFraction * storageSizeInBytes / oneGigiByteInBytes * frequencyPerMonth
		costs = append(costs, &storageCost)
	}
	return costs, err
}

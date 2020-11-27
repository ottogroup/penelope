package processor

import (
    "context"
    "fmt"
    "github.com/pkg/errors"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/provider"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "github.com/ottogroup/penelope/pkg/service/billing"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "go.opencensus.io/trace"
    "math"
)

const oneGigiByteInBytes = 1073741824

// CalculatingProcessorFactory create Process for Calculating
type CalculatingProcessorFactory struct {
    backupProvider      provider.SinkGCPProjectProvider
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func NewCalculatingProcessorFactory(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) *CalculatingProcessorFactory {
    return &CalculatingProcessorFactory{
        backupProvider: backupProvider,
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
        backupProvider: c.backupProvider,
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
    cost := getStorageCost(storageClass, request.TargetOptions.Region)
    var (
        storagePricePerGBMonth     float64
        earlyDeletePricePerGBMonth float64
        frequencyPerMonth          float64
        periods                    []int64
    )
    if cost.StorageEAN != "" {
        storagePricePerGBMonth, err = pricePerMonth(c.billingClient, cost.StorageEAN)
        if err != nil {
            return costs, errors.Wrap(err, "pricePerMonth failed")
        }
    }
    if cost.EarlyDeleteEAN != "" {
        earlyDeletePricePerGBMonth, err = pricePerMonth(c.billingClient, cost.EarlyDeleteEAN)
        if err != nil {
            return costs, errors.Wrap(err, "pricePerMonth failed")
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
        if period < cost.MinTTL {
            costFraction += storagePricePerGBMonth * float64(period) / float64(cost.MinTTL)
            costFraction += earlyDeletePricePerGBMonth * float64(cost.MinTTL-period) / float64(cost.MinTTL)
        } else {
            costFraction += storagePricePerGBMonth * float64(period)
        }

        storageCost.Cost += costFraction * storageSizeInBytes / oneGigiByteInBytes * frequencyPerMonth
        costs = append(costs, &storageCost)
    }
    return costs, err
}

func pricePerMonth(client billing.Client, ean string) (float64, error) {
    sku, err := client.GetServiceSkuByEan(ean)
    if err != nil {
        return 0, errors.Wrap(err, fmt.Sprintf("GetServiceSkuByEan failed for ean %s", ean))
    }
    if sku != nil {
        if 0 < len(sku.PricingInfo) &&
            nil != sku.PricingInfo[0].PricingExpression &&
            0 < len(sku.PricingInfo[0].PricingExpression.TieredRates) &&
            nil != sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice {
            return float64(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos) / math.Pow(10, 9), nil
        }
    }
    return 0, errors.New(fmt.Sprintf("sku for ean %s not found", ean))
}

func getStorageCost(storageClass string, region string) storageCost {
    // source: https://developers.google.com/apis-explorer/#search/cloudbilling.services.skus.list/m/cloudbilling/v1/cloudbilling.services.skus.list?parent=services%252F95FF-2EF5-5EA1&currencyCode=EUR&_h=17&
    var costs = []storageCost{
        {Region: repository.EuropeWest1, StorageClass: repository.Regional, StorageEAN: "A703-5CB6-E0BF"},
        {Region: repository.EuropeWest1, StorageClass: repository.Nearline, StorageEAN: "D78D-ECDE-752A", EarlyDeleteEAN: "64AA-A2F3-3387", MinTTL: 1},
        {Region: repository.EuropeWest1, StorageClass: repository.Coldline, StorageEAN: "DB5F-944C-9031", EarlyDeleteEAN: "3BDD-5E66-FF01", MinTTL: 3},
        {Region: repository.EuropeWest3, StorageClass: repository.Regional, StorageEAN: "F272-7933-F065"},
        {Region: repository.EuropeWest3, StorageClass: repository.Nearline, StorageEAN: "4783-1B32-D7D2", EarlyDeleteEAN: "7FCC-957C-36E9", MinTTL: 1},
        {Region: repository.EuropeWest3, StorageClass: repository.Coldline, StorageEAN: "DCF3-6CFB-DC70", EarlyDeleteEAN: "70BC-7DCD-47E1", MinTTL: 3},
        {Region: repository.EuropeWest4, StorageClass: repository.Regional, StorageEAN: "89D8-0CF9-9F2E"},
        {Region: repository.EuropeWest4, StorageClass: repository.Nearline, StorageEAN: "A5D0-60CC-E116", EarlyDeleteEAN: "9AB3-E6C0-3726", MinTTL: 1},
        {Region: repository.EuropeWest4, StorageClass: repository.Coldline, StorageEAN: "2743-ACA0-4E7F", EarlyDeleteEAN: "6D21-A940-7268", MinTTL: 3},
    }

    for _, cost := range costs {
        if cost.StorageClass.EqualTo(storageClass) && cost.Region.EqualTo(region) {
            return cost
        }
    }
    return storageCost{}
}

type storageCost struct {
    StorageEAN     string
    EarlyDeleteEAN string
    MinTTL         int64
    Region         repository.Region
    StorageClass   repository.StorageClass
}

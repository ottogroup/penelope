package processor

import (
    "context"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    bq "github.com/ottogroup/penelope/pkg/service/bigquery"
    "fmt"
    "google.golang.org/api/cloudbilling/v1"
    "testing"
)

func TestCalculatingProcessor_Process_expect0(t *testing.T) {
    // Given
    sourceProjectID := "local-account"
    calculateRequest := requestobjects.CalculateRequest{}
    calculateRequest.Project = sourceProjectID
    calculateRequest.TargetOptions = requestobjects.TargetOptions{Region: repository.EuropeWest1.String(), StorageClass: repository.Regional.String()}
    calculateRequest.Type = repository.BigQuery.String()
    calculateRequest.BigQueryOptions = requestobjects.BigQueryOptions{Dataset: "Billing", Table: []string{"gcp_billing_export_v1_01E0E5_3EB3D2_2206EC$20190101"}}

    calculatorContext := givenATestBigQueryCalculatorContext()
    calculatorContext.BigQuery.fGetTable = &bq.Table{Name: "gcp_billing_export_v1_01E0E5_3EB3D2_2206EC$20190101", SizeInBytes: 0}
    calculatorContext.addPriceForStorage(9999, 0, calculateRequest.TargetOptions.StorageClass, calculateRequest.TargetOptions.Region)
    calcuator := bigQueryCalculator{bigQueryClient: &calculatorContext.BigQuery, baseCalculator: baseCalculator{billingClient: &calculatorContext.Billing}}
    // When
    calculateResponse, err := calcuator.calculateCost(context.Background(), &calculateRequest)
    // Then
    if err != nil {
        t.Errorf(fmt.Sprintf("calculateCost failed. Err %+v", err))
    }
    if calculateResponse == nil {
        t.Errorf("CalculateResponse is nil")
        return
    }
    if len(calculateResponse.Costs) != 12 {
        t.Errorf("CalculateResponse expected costs for whole year")
    }
    for _, cost := range calculateResponse.Costs {
        if cost.Cost != 0 {
            t.Errorf("CalculateResponse expected price to be 0")
        }
    }
}

func TestCalculatingProcessor_Process_expectOne(t *testing.T) {
    // Given
    sourceProjectID := "local-account"
    calculateRequest := requestobjects.CalculateRequest{}
    calculateRequest.Project = sourceProjectID
    calculateRequest.TargetOptions = requestobjects.TargetOptions{Region: repository.EuropeWest1.String(), StorageClass: repository.Regional.String()}
    calculateRequest.Type = repository.BigQuery.String()
    calculateRequest.Strategy = repository.Snapshot.String()
    calculateRequest.BigQueryOptions = requestobjects.BigQueryOptions{Dataset: "Billing", Table: []string{"gcp_billing_export_v1_01E0E5_3EB3D2_2206EC$20190101"}}
    calculateRequest.SnapshotOptions = requestobjects.SnapshotOptions{LifetimeInDays: 20}

    calculatorContext := givenATestBigQueryCalculatorContext()
    var oneHundredGigiByteInGB float64 = 100
    oneHundredGigiByteInBytes := oneHundredGigiByteInGB * oneGigiByteInBytes
    calculatorContext.BigQuery.fGetTable = &bq.Table{Name: "gcp_billing_export_v1_01E0E5_3EB3D2_2206EC$20190101", SizeInBytes: oneHundredGigiByteInBytes}
    var pricePerGgiByteInNanos int64 = 17618000
    calculatorContext.addPriceForStorage(pricePerGgiByteInNanos, 0, calculateRequest.TargetOptions.StorageClass, calculateRequest.TargetOptions.Region)
    calcuator := bigQueryCalculator{bigQueryClient: &calculatorContext.BigQuery, baseCalculator: baseCalculator{billingClient: &calculatorContext.Billing}}
    // When
    calculateResponse, err := calcuator.calculateCost(context.Background(), &calculateRequest)
    // Then
    if err != nil {
        t.Errorf(fmt.Sprintf("calculateCost failed. Err %+v", err))
    }
    if calculateResponse == nil {
        t.Errorf("CalculateResponse is nil")
        return
    }
    if len(calculateResponse.Costs) != 1 {
        t.Errorf("CalculateResponse expected one cost")
        return
    }
    expectedCost := float64(pricePerGgiByteInNanos) * 0.000000001 * float64(calculateRequest.SnapshotOptions.LifetimeInDays) * oneHundredGigiByteInGB
    cost := calculateResponse.Costs[0]
    if !floatEquals(expectedCost, cost.Cost) {
        t.Errorf("CalculateResponse expected price to be %f was %f", expectedCost, cost.Cost)
    }
}

type testBillingClient struct {
    SKU map[string]*cloudbilling.Sku
    Err error
}

func (t *testBillingClient) GetServiceSkuByEan(ean string) (*cloudbilling.Sku, error) {
    return t.SKU[ean], t.Err
}

type contextBigQueryCalculator struct {
    BigQuery testBigQueryClient
    Billing  testBillingClient
}

func (ctx *contextBigQueryCalculator) addPriceForStorage(priceInNanos int64, priceEarlyDeletionInNanos int64, storageClass string, storageRegion string) {
    cost := getStorageCost(storageClass, storageRegion)
    if cost.StorageEAN == "" {
        panic("ean is empty")
    }
    ctx.Billing.SKU[cost.StorageEAN] = &cloudbilling.Sku{
        PricingInfo: []*cloudbilling.PricingInfo{
            {
                PricingExpression: &cloudbilling.PricingExpression{
                    TieredRates: []*cloudbilling.TierRate{
                        {UnitPrice: &cloudbilling.Money{
                            CurrencyCode: "EUR",
                            Nanos:        priceInNanos,
                        }},
                    },
                },
            },
        },
    }
    if 0 < priceEarlyDeletionInNanos && cost.EarlyDeleteEAN == "" {
        panic("ean for early deletion is empty")
    }
    if 0 < priceEarlyDeletionInNanos {
        ctx.Billing.SKU[cost.EarlyDeleteEAN] = &cloudbilling.Sku{
            PricingInfo: []*cloudbilling.PricingInfo{
                {
                    PricingExpression: &cloudbilling.PricingExpression{
                        TieredRates: []*cloudbilling.TierRate{
                            {UnitPrice: &cloudbilling.Money{
                                CurrencyCode: "EUR",
                                Nanos:        priceEarlyDeletionInNanos,
                            }},
                        },
                    },
                },
            },
        }
    }
}

func givenATestBigQueryCalculatorContext() *contextBigQueryCalculator {
    bigQueryClient := testBigQueryClient{}
    billingClient := testBillingClient{}
    billingClient.SKU = make(map[string]*cloudbilling.Sku)
    return &contextBigQueryCalculator{
        BigQuery: bigQueryClient,
        Billing:  billingClient,
    }
}

func floatEquals(a, b float64) bool {
    var EPSILON = 0.000000000000001
    if (a-b) < EPSILON && (b-a) < EPSILON {
        return true
    }
    return false
}

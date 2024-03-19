package billing

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/ottogroup/penelope/pkg/config"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	gcpBilling "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"
)

// Client represent operation with the GCP billing
type Client interface {
	GetServiceSkuBySKUId(ean string) (*gcpBilling.Sku, error)
	PricePerMonth(skuid string) (float64, error)
}

// defaultCloudBillingClient implements Client
type defaultCloudBillingClient struct {
	client *gcpBilling.APIService
	ctx    context.Context
}

// NewCloudBillingClient crete new instance of Client
func NewCloudBillingClient(ctxIn context.Context) (Client, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewCloudBillingClient")
	defer span.End()

	o := []option.ClientOption{option.WithScopes("https://www.googleapis.com/auth/cloud-platform")}
	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		o = append(o, option.WithHTTPClient(http.DefaultClient))
	}
	client, err := gcpBilling.NewService(ctx, o...)
	if err != nil {
		return &defaultCloudBillingClient{}, fmt.Errorf("failed to create client: %v", err)
	}

	return &defaultCloudBillingClient{client: client, ctx: ctx}, nil
}

// GetServiceSkuBySKU get actual service SKU by skuID
func (c *defaultCloudBillingClient) GetServiceSkuBySKUId(skuID string) (*gcpBilling.Sku, error) {
	gcpCloudStorageName := "services/95FF-2EF5-5EA1"
	skus, err := c.client.Services.Skus.List(gcpCloudStorageName).CurrencyCode("EUR").Do()
	if err != nil {
		return nil, errors.Wrap(err, "ServicesList failed")
	}

	for _, sku := range skus.Skus {
		if sku.SkuId == skuID {
			return sku, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("SKU not found %s", skuID))
}

func (c *defaultCloudBillingClient) PricePerMonth(skuID string) (float64, error) {
	averageGregorianDaysInMonth := 365.2425 / 12

	sku, err := c.GetServiceSkuBySKUId(skuID)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("GetServiceSkuBySKUId failed for skuID %s", skuID))
	}
	if sku != nil {
		if 0 < len(sku.PricingInfo) &&
			nil != sku.PricingInfo[0].PricingExpression &&
			0 < len(sku.PricingInfo[0].PricingExpression.TieredRates) &&
			nil != sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice {
			return float64(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos) * math.Pow(10, 9) / float64(sku.PricingInfo[0].PricingExpression.BaseUnitConversionFactor) / math.Pow(2, 30) * 60 * 60 * 24 * averageGregorianDaysInMonth, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("sku for skuID %s not found", skuID))
}

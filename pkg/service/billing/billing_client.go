package billing

import (
    "context"
    "fmt"
    "github.com/pkg/errors"
    "github.com/ottogroup/penelope/pkg/config"
    "go.opencensus.io/trace"
    gcpBilling "google.golang.org/api/cloudbilling/v1"
    "google.golang.org/api/option"
    "net/http"
)

// Client represent operation with the GCP billing
type Client interface {
    GetServiceSkuByEan(ean string) (*gcpBilling.Sku, error)
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
    if config.UseDefaultHttpClient.GetBoolOrDefault(false)  {
        o = append(o, option.WithHTTPClient(http.DefaultClient))
    }
    client, err := gcpBilling.NewService(ctx, o...)
    if err != nil {
        return &defaultCloudBillingClient{}, fmt.Errorf("failed to create client: %v", err)
    }

    return &defaultCloudBillingClient{client: client, ctx: ctx}, nil
}

// GetServiceSkuByEan get actual service SKU by EAN
func (c *defaultCloudBillingClient) GetServiceSkuByEan(ean string) (*gcpBilling.Sku, error) {
    gcpCloudStorageName := "services/95FF-2EF5-5EA1"
    skus, err := c.client.Services.Skus.List(gcpCloudStorageName).CurrencyCode("EUR").Do()
    if err != nil {
        return nil, errors.Wrap(err, "ServicesList failed")
    }

    for _, sku := range skus.Skus {
        if sku.SkuId == ean {
            return sku, nil
        }
    }
    return nil, errors.New(fmt.Sprintf("SKU not found %s", ean))
}

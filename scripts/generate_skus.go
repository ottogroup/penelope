package main

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/cloudbilling/v1"
)

func main() {
	ctx := context.Background()
	cloudbillingService, err := cloudbilling.NewService(ctx)
	if err != nil {
		panic(err)
	}

	res, err := cloudbillingService.Services.Skus.List("services/95FF-2EF5-5EA1").CurrencyCode("EUR").Do()
	if err != nil {
		panic(err)
	}

	regions := []string{
		"europe-west1",
		"europe-west3",
		"europe-west4",
		"europe-west6",
		"europe-west6",
		"europe-west8",
		"europe-west9",
		"europe-west10",
		"europe-west12",
		"europe-central2",
		"europe-southwest1",
		"europe-north1",
	}

	classes := []string{
		"regional",
		"nearline",
		"coldline",
		"archive",
	}

	for _, region := range regions {
		for _, class := range classes {
			for _, dualRegion := range []bool{false, true} {
				storageSku := ""
				earlyDeleteSku := ""
				ttl := 0
				switch class {
				case "nearline":
					ttl = 1
				case "coldline":
					ttl = 3
				case "archive":
					ttl = 12
				}
				for _, sku := range res.Skus {
					if sku.Category.ResourceFamily == "Storage" &&
						sku.Category.UsageType == "OnDemand" &&
						len(sku.ServiceRegions) == 1 && sku.ServiceRegions[0] == region &&
						(strings.ToLower(sku.Category.ResourceGroup) == class+"storage" || sku.Category.ResourceGroup == "MultiRegionalStorage" && dualRegion && class == "regional") &&
						(strings.Contains(sku.Description, "Dual-region") && dualRegion || !strings.Contains(sku.Description, "Dual-region") && !dualRegion) {
						if strings.Contains(sku.Description, "(Early Delete)") {
							earlyDeleteSku = sku.SkuId
						} else {
							storageSku = sku.SkuId
						}
					}
				}
				other := ""
				if earlyDeleteSku != "" {
					other = fmt.Sprintf(", EarlyDeleteSKU: %q, MinTTL: %d", earlyDeleteSku, ttl)
				}
				if storageSku != "" {
					fmt.Printf("{Region: %q, StorageClass: %q, DualRegion: %v, StorageSKU: %q%s},\n", region, strings.ToUpper(class), dualRegion, storageSku, other)
				}
			}
		}
	}

	for _, class := range classes {
		storageSku := ""
		earlyDeleteSku := ""
		ttl := 0
		switch class {
		case "nearline":
			ttl = 1
		case "coldline":
			ttl = 3
		case "archive":
			ttl = 12
		}
		for _, sku := range res.Skus {
			if sku.Category.ResourceFamily == "Storage" &&
				sku.Category.UsageType == "OnDemand" &&
				len(sku.ServiceRegions) == 1 && sku.ServiceRegions[0] == "europe" &&
				(strings.ToLower(sku.Category.ResourceGroup) == class+"storage" || sku.Category.ResourceGroup == "MultiRegionalStorage" && class == "regional") {
				if strings.Contains(sku.Description, "(Early Delete)") {
					earlyDeleteSku = sku.SkuId
				} else {
					storageSku = sku.SkuId
				}
			}
		}
		other := ""
		if earlyDeleteSku != "" {
			other = fmt.Sprintf(", EarlyDeleteSKU: %q, MinTTL: %d", earlyDeleteSku, ttl)
		}
		fmt.Printf("{Region: %q, StorageClass: %q, DualRegion: %v, StorageSKU: %q%s},\n", "eu", strings.ToUpper(class), false, storageSku, other)
	}
}

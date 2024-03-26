package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/ottogroup/penelope/pkg/processor"
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

	regions := map[string]processor.Location{
		"europe-west1":      {Latitude: 50.4841084, Longitude: 3.7646014},
		"europe-west3":      {Latitude: 50.1177289, Longitude: 8.595807},
		"europe-west4":      {Latitude: 53.4342611, Longitude: 6.8033996},
		"europe-west8":      {Latitude: 45.4678811, Longitude: 9.1320676},
		"europe-west9":      {Latitude: 48.8602937, Longitude: 2.2821444},
		"europe-west10":     {Latitude: 52.510282, Longitude: 13.2942595},
		"europe-west12":     {Latitude: 45.0711719, Longitude: 7.633703},
		"europe-central2":   {Latitude: 52.2300896, Longitude: 20.9108091},
		"europe-southwest1": {Latitude: 40.440189, Longitude: -3.8079513},
		"europe-north1":     {Latitude: 60.5695263, Longitude: 27.1520617},
	}

	classes := []string{
		"regional",
		"nearline",
		"coldline",
		"archive",
	}

	for region, location := range regions {
		fmt.Printf("{\n")
		fmt.Printf("	Region: repository.Region(%q),\n", region)
		fmt.Printf("	Location: Location{\n")
		fmt.Printf("		Latitude:  %f,\n", location.Latitude)
		fmt.Printf("		Longitude: %f,\n", location.Longitude)
		fmt.Printf("	},\n")
		fmt.Printf("	StorageClasses: []RegionConfigurationStorageClass{\n")
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
					fmt.Printf("		{StorageClass: %q, DualRegion: %v, StorageSKU: %q%s},\n", strings.ToUpper(class), dualRegion, storageSku, other)
				}
			}
		}
		fmt.Printf("	},\n")
		fmt.Printf("},\n")
	}

	fmt.Printf("{\n")
	fmt.Printf("	Region: repository.Region(%q),\n", "eu")
	fmt.Printf("	StorageClasses: []RegionConfigurationStorageClass{\n")
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
		fmt.Printf("		{StorageClass: %q, DualRegion: %v, StorageSKU: %q%s},\n", strings.ToUpper(class), false, storageSku, other)
	}
	fmt.Printf("	},\n")
	fmt.Printf("},\n")

}

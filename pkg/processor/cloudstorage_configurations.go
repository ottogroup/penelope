package processor

import "github.com/ottogroup/penelope/pkg/repository"

type StorageConfiguration struct {
	StorageSKU     string
	EarlyDeleteSKU string
	MinTTL         int64
	DualRegion     bool
	Region         repository.Region
	StorageClass   repository.StorageClass
}

type RegionConfigurationStorageClass struct {
	StorageClass   repository.StorageClass
	DualRegion     bool
	StorageSKU     string
	EarlyDeleteSKU string
	MinTTL         int64
}

type RegionConfiguration struct {
	Region         repository.Region
	MultiRegion    bool
	Location       Location
	StorageClasses []RegionConfigurationStorageClass
}

// source: use go run scripts/generate_skus.go to generate the region location sku mapping
// https://developers.google.com/apis-explorer/#search/cloudbilling.services.skus.list/m/cloudbilling/v1/cloudbilling.services.skus.list?parent=services%252F95FF-2EF5-5EA1&currencyCode=EUR&_h=17&
var RegionConfigurations = []RegionConfiguration{
	{
		Region: repository.Region("europe-west4"),
		Location: Location{
			Latitude:  53.434261,
			Longitude: 6.803400,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "89D8-0CF9-9F2E"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "0548-E341-C883"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "A5D0-60CC-E116", EarlyDeleteSKU: "9AB3-E6C0-3726", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "9F11-CF06-6C09", EarlyDeleteSKU: "684E-F629-BFBD", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "2743-ACA0-4E7F", EarlyDeleteSKU: "6D21-A940-7268", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "3041-B9C2-F02A", EarlyDeleteSKU: "0181-CECA-98F3", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "BDF2-208A-3B0C", EarlyDeleteSKU: "7991-C7BF-8EBA", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "2301-2DCC-CD47", EarlyDeleteSKU: "5279-D321-062E", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-west8"),
		Location: Location{
			Latitude:  45.467881,
			Longitude: 9.132068,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "967B-9AE8-BB04"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "1800-5978-6B82"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "CA49-1DFC-B6AF", EarlyDeleteSKU: "2E93-A156-DF36", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "EBC2-A551-2247", EarlyDeleteSKU: "AAB6-39A6-1DEF", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "C1D5-B464-7C66", EarlyDeleteSKU: "C6D4-99D9-A78F", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "8BD6-1E2A-024D", EarlyDeleteSKU: "C73E-D7AC-C053", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "28C7-8797-6E71", EarlyDeleteSKU: "4525-728E-374B", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "9729-6081-2B55", EarlyDeleteSKU: "9BD8-691A-AE7B", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-central2"),
		Location: Location{
			Latitude:  52.230090,
			Longitude: 20.910809,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "109D-AA78-91BA"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "3E50-C6B4-D105"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "7446-C63C-1119", EarlyDeleteSKU: "9A31-06CB-DAA4", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "4B1F-1215-B3DE", EarlyDeleteSKU: "E672-7377-CBBA", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "3F00-23CF-9974", EarlyDeleteSKU: "542E-68D8-F5AC", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "4D09-6DB4-A3D7", EarlyDeleteSKU: "2DC6-FFE6-0049", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "A3BF-20BC-C5F3", EarlyDeleteSKU: "3117-0B29-40A5", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "1E4B-1611-A2BC", EarlyDeleteSKU: "C0B6-13A5-3232", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-north1"),
		Location: Location{
			Latitude:  60.569526,
			Longitude: 27.152062,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "8A7F-592F-3FE7"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "17FE-F1D5-27BA"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "9556-97BF-3B42", EarlyDeleteSKU: "097C-4088-7F5E", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "002B-81E7-32DC", EarlyDeleteSKU: "4973-CF7D-8C5C", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "2701-887F-4E66", EarlyDeleteSKU: "C7B3-B5AA-76F6", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "FF44-B6BB-8EB3", EarlyDeleteSKU: "8FB1-79B3-49A6", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "8F96-C717-3C2B", EarlyDeleteSKU: "C263-A596-0D53", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "F55D-740B-7F5B", EarlyDeleteSKU: "4A99-E439-94BB", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-west1"),
		Location: Location{
			Latitude:  50.484108,
			Longitude: 3.764601,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "A703-5CB6-E0BF"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "FDBF-9FEC-415F"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "D78D-ECDE-752A", EarlyDeleteSKU: "64AA-A2F3-3387", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "1D4D-401D-A201", EarlyDeleteSKU: "4B78-498E-1D79", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "DB5F-944C-9031", EarlyDeleteSKU: "3BDD-5E66-FF01", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "DE73-C395-1FEF", EarlyDeleteSKU: "34C9-29F4-70AC", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "1A2D-77C9-8F73", EarlyDeleteSKU: "7603-88E7-1A6E", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "069D-4E4A-CF05", EarlyDeleteSKU: "67DE-966A-5AFD", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-west3"),
		Location: Location{
			Latitude:  50.117729,
			Longitude: 8.595807,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "F272-7933-F065"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "BA45-D7D7-2F8E"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "4783-1B32-D7D2", EarlyDeleteSKU: "7FCC-957C-36E9", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "6E82-022D-E319", EarlyDeleteSKU: "E002-E53B-58A5", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "DCF3-6CFB-DC70", EarlyDeleteSKU: "70BC-7DCD-47E1", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "7886-16E5-8740", EarlyDeleteSKU: "8F73-6C51-735C", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "D9F7-D25F-B0EF", EarlyDeleteSKU: "0F22-5F50-0592", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "A6F4-AEF2-A993", EarlyDeleteSKU: "AD23-DE05-9836", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-west9"),
		Location: Location{
			Latitude:  48.860294,
			Longitude: 2.282144,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "5CEA-0E42-9E49"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "5158-BEE6-C24D"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "6A7E-88B2-EE62", EarlyDeleteSKU: "AB97-A3D6-5580", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "ADB6-51A3-28B0", EarlyDeleteSKU: "7AD0-3666-2E2E", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "D056-E841-A979", EarlyDeleteSKU: "166D-48FE-467C", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "5712-0B8C-E4DD", EarlyDeleteSKU: "2F84-6D71-FBF4", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "64CC-FF04-FD8A", EarlyDeleteSKU: "6DC3-F2C3-559A", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "C6A0-DADE-4F0E", EarlyDeleteSKU: "5FCC-6D81-8A2F", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-west10"),
		Location: Location{
			Latitude:  52.510282,
			Longitude: 13.294260,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "7F2E-E019-951F"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "BEA8-18F6-120A", EarlyDeleteSKU: "6361-43DA-4A55", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "7EE1-A6EA-1793", EarlyDeleteSKU: "C948-2456-998B", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "D1E2-71BA-61C9", EarlyDeleteSKU: "7444-90DF-C64B", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-west12"),
		Location: Location{
			Latitude:  45.071172,
			Longitude: 7.633703,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "126D-5D14-CE5B"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "2B1C-548C-DE4F", EarlyDeleteSKU: "CDD5-0247-0C8E", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "2083-620F-21F6", EarlyDeleteSKU: "CFAF-C3AB-127A", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "EFF5-EB22-3D38", EarlyDeleteSKU: "AB56-C9B6-8E24", MinTTL: 12},
		},
	},
	{
		Region: repository.Region("europe-southwest1"),
		Location: Location{
			Latitude:  40.440189,
			Longitude: -3.807951,
		},
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "5596-85FF-7F53"},
			{StorageClass: "REGIONAL", DualRegion: true, StorageSKU: "8EF1-47E4-652E"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "90B1-81AB-9A8B", EarlyDeleteSKU: "27BC-BFB8-42D5", MinTTL: 1},
			{StorageClass: "NEARLINE", DualRegion: true, StorageSKU: "BDF5-445D-55FB", EarlyDeleteSKU: "2E74-93E1-A4A5", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "9675-44B2-9CC6", EarlyDeleteSKU: "1BA1-CDDD-D5BC", MinTTL: 3},
			{StorageClass: "COLDLINE", DualRegion: true, StorageSKU: "F252-25C0-60F6", EarlyDeleteSKU: "8073-135B-C5B9", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "5ACD-E7DB-7BFA", EarlyDeleteSKU: "6A34-AEE7-B67A", MinTTL: 12},
			{StorageClass: "ARCHIVE", DualRegion: true, StorageSKU: "31B5-A00B-9A1E", EarlyDeleteSKU: "8CF2-39BA-4E88", MinTTL: 12},
		},
	},
	{
		Region:      repository.Region("eu"),
		MultiRegion: true,
		StorageClasses: []RegionConfigurationStorageClass{
			{StorageClass: "REGIONAL", DualRegion: false, StorageSKU: "EC40-8747-D6FF"},
			{StorageClass: "NEARLINE", DualRegion: false, StorageSKU: "4CF0-069A-15D9", EarlyDeleteSKU: "7BE5-EBE7-F791", MinTTL: 1},
			{StorageClass: "COLDLINE", DualRegion: false, StorageSKU: "6CCC-4CDD-383C", EarlyDeleteSKU: "8C3D-1047-DDD3", MinTTL: 3},
			{StorageClass: "ARCHIVE", DualRegion: false, StorageSKU: "5A75-E003-2CBF", EarlyDeleteSKU: "4E05-6445-0415", MinTTL: 12},
		},
	},
}

var StorageConfigurations = func() []StorageConfiguration {
	var result = make([]StorageConfiguration, 0)
	for _, regionConfig := range RegionConfigurations {
		for _, storageClass := range regionConfig.StorageClasses {
			result = append(result, StorageConfiguration{
				StorageSKU:     storageClass.StorageSKU,
				EarlyDeleteSKU: storageClass.EarlyDeleteSKU,
				MinTTL:         storageClass.MinTTL,
				DualRegion:     storageClass.DualRegion,
				Region:         regionConfig.Region,
				StorageClass:   storageClass.StorageClass,
			})
		}
	}
	return result
}()

var StorageClasses []repository.StorageClass = func() []repository.StorageClass {
	var storageClasses = make(map[repository.StorageClass]bool)
	for _, conf := range StorageConfigurations {
		storageClasses[conf.StorageClass] = true
	}

	var result = make([]repository.StorageClass, 0, len(storageClasses))
	for storageClass := range storageClasses {
		result = append(result, storageClass)
	}
	return result
}()

var Regions []repository.Region = func() []repository.Region {
	var regions = make(map[repository.Region]bool)
	for _, conf := range RegionConfigurations {
		regions[conf.Region] = true
	}

	var result = make([]repository.Region, 0, len(regions))
	for region := range regions {
		result = append(result, region)
	}
	return result
}()

func getStorageCost(storageClass string, region string, dualRegion bool) StorageConfiguration {
	for _, cost := range StorageConfigurations {
		if cost.StorageClass.EqualTo(storageClass) && cost.Region.EqualTo(region) && dualRegion == cost.DualRegion {
			return cost
		}
	}
	return StorageConfiguration{}
}

func getRegionConfiguration(region string) RegionConfiguration {
	for _, conf := range RegionConfigurations {
		if conf.Region.EqualTo(region) {
			return conf
		}
	}
	return RegionConfiguration{}
}

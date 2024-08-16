package requestobjects

import "strings"

// RequestType type of a request
type RequestType string

const (
	// Creating - create a new backup
	Creating RequestType = "Creating"
	// Getting - get backup information
	Getting RequestType = "Getting"
	// Listing - list backup
	Listing RequestType = "Listing"
	// Updating - change backup
	Updating RequestType = "Updating"
	// Restoring - preapre restore command for a backup
	Restoring RequestType = "Restoring"
	// Calculating - calculate prize for a backup
	Calculating RequestType = "Calculating"
	// Compliance - calculate compliance checks for a backup
	Compliance RequestType = "Compliance"
	// DatasetListing - list datasets avaiable for a User
	DatasetListing RequestType = "DatasetListing"
	// BucketListing - list buckets avaiable for a User
	BucketListing RequestType = "BuckeListing"
	// SourceProjectGet - get Source Project for given project ID
	SourceProjectGet RequestType = "SourceProjectGet"
	// Cleanup - cleanup trash can for a backup
	Cleanup RequestType = "Cleanup"
)

func (s RequestType) String() string {
	return string(s)
}

// EqualTo check if a given string match type
func (s RequestType) EqualTo(requestType string) bool {
	return strings.EqualFold(requestType, s.String())
}

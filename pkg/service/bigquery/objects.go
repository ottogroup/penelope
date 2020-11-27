package bigquery

import (
    bq "cloud.google.com/go/bigquery"
)

const (
	defaultAPIScope = "https://www.googleapis.com/auth/bigquery"
	cloudPlatformAPIScope = "https://www.googleapis.com/auth/cloud-platform"
)

// ExtractJobState State is one of a sequence of states that a Job progresses through as it is processed.
type ExtractJobState int

const (
    // StateUnspecified is the default JobIterator state.
    StateUnspecified ExtractJobState = iota
    // Pending is a state that describes that the job is pending.
    Pending
    // Running is a state that describes that the job is running.
    Running
    // Done is a state that describes that the job is done.
    Done
    // Failed is a state that describes that the job complete unsuccessfully.
    Failed
    // FailedQuotaExceeded for project exceeded 11 TB
    FailedQuotaExceeded
)

func (s ExtractJobState) String() string {
    return []string{"Unspecified", "Pending", "Running", "Done", "Failed"}[s]
}

func toJobState(status bq.State) ExtractJobState {
    switch status {
    case bq.StateUnspecified:
        return StateUnspecified
    case bq.Pending:
        return Pending
    case bq.Running:
        return Running
    case bq.Done:
        return Done
    }

    return StateUnspecified
}

package gcs

const (
	defaultAPIScope = "https://www.googleapis.com/auth/devstorage.full_control"
    metricAPIScope = "https://www.googleapis.com/auth/monitoring.read"
    cloudPlatformAPIScope = "https://www.googleapis.com/auth/cloud-platform"
)

// TransferJobState State is one of a sequence of states that a Job progresses through as it is processed.
type TransferJobState string

const (
    // StateUnspecified is the default JobIterator state.
    StateUnspecified TransferJobState = "Unspecified"
    // Pending is a state that describes that the job is pending.
    Pending TransferJobState = "Pending"
    // Running is a state that describes that the job is running.
    Running TransferJobState = "Running"
    // Done is a state that describes that the job is done.
    Done TransferJobState = "Done"
    // Failed is a state that describes that the job complete unsuccessfully.
    Failed TransferJobState = "Failed"
)

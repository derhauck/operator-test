package v1

type ConditionType string

const (
	// TypeAvailableObserver represents the status of the Deployment reconciliation
	TypeAvailableObserver ConditionType = "Available"
	// TypeDegradedObserver represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	TypeDegradedObserver ConditionType = "Degraded"
	// TypeStatusFailure represents the last failure status of the pod checking the endpoint
	TypeStatusFailure ConditionType = "StatusFailure"
	// TypeStatusSuccess represents the last successful status of the pod checking the endpoint
	TypeStatusSuccess ConditionType = "StatusSuccess"
)

package types

const (
	PropertyCreated = "created"
	PropertyUpdated = "updated"
)

type NamespacedName struct {
	Namespace string
	Name      string
}

type Item struct {
	NamespacedName
	HandlerID string
	Result    PolicyReportResult
	Source    interface{}
}

func ItemFor(handlerID string, namespace string, name string, result PolicyReportResult, source interface{}) *Item {
	return &Item{
		HandlerID: handlerID,
		NamespacedName: NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		Result: result,
		Source: source,
	}
}

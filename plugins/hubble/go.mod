module github.com/bakito/policy-report-publisher-plugin-hubble

go 1.25.3

replace github.com/bakito/policy-report-publisher-shared v0.0.0 => ../../shared

require (
	github.com/bakito/policy-report-publisher-shared v0.0.0
	github.com/cilium/cilium v1.18.5
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3
	google.golang.org/grpc v1.78.0
)

require (
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

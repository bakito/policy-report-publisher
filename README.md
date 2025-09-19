# Policy Report Publisher

Policy Report Publisher is a Kubernetes utility designed to listen for security-related events and network policy violations from multiple sources (adapters) in your cluster, convert them into PolicyReports, and publish them to the Kubernetes API. This enables integration with policy engines and dashboards such as Kyverno or the Policy Reporter.

## Features

- **Adapter-based event collection**: Supports collecting and converting events from different sources (currently [Cilium Hubble](https://github.com/cilium/hubble) for network events and [KubeArmor](https://github.com/kubearmor/kubearmor) for runtime security).
- **PolicyReport generation**: Creates and updates Kubernetes PolicyReport CRs to track violations and alerts.
- **Prometheus metrics**: Exposes metrics on processed items per adapter.
- **Leader election support**: Optionally runs as a leader in a multi-replica setup.
- **Graceful shutdown**: Handles OS signals and shuts down cleanly.
- **Highly configurable via environment variables**.

## Adapters

- **Hubble**: Listens for dropped egress flows (network traffic blocked by Cilium policies), converting them into PolicyReport results indicating failed egress attempts.
- **KubeArmor**: Listens for container security alerts, converting them into PolicyReport results tied to the affected pod.

## Usage

### Prerequisites

- Kubernetes cluster with [Kyverno](https://kyverno.io/) CRDs installed for PolicyReports.
- Optional: Cilium with Hubble Relay for network visibility, KubeArmor for runtime security events.

### Deployment

You can build and run the publisher as a container:

```shell
docker build -t bakito/policy-report-publisher .
docker run -e HUBBLE_SERVICE_NAME=<hubble-relay-address> \
           -e KUBEARMOR_SERVICE_NAME=<kubearmor-address> \
           -e LOG_REPORTS=true \
           bakito/policy-report-publisher
```

#### Environment Variables

- `HUBBLE_SERVICE_NAME`: gRPC address to the Hubble relay service (enables Hubble adapter).
- `KUBEARMOR_SERVICE_NAME`: gRPC address to the KubeArmor service (enables KubeArmor adapter).
- `LOG_REPORTS`: If set, enables logging of processed reports.
- `LEADER_ELECTION_NS`: Namespace to use for leader election (optional, enables HA).

### RBAC & CRD

The publisher will attempt to create/update PolicyReport resources. Ensure your deployment has the necessary RBAC permissions:

- `get;list;watch` on Pods
- `get;list;watch;create;update;patch` on PolicyReports

### Makefile Tasks

- `make rbac`: Generate RBAC manifests for PolicyReportPublisher.
- `make lint`: Run Go linting.
- `make tidy`: Clean up Go modules.

## Architecture

1. On startup, the publisher checks which adapters are enabled via environment variables.
2. Each enabled adapter runs in its own goroutine, watching for relevant security/network events.
3. Events are converted into `PolicyReportResult` objects and sent to a central channel.
4. The report handler consumes these events, updating or creating PolicyReport CRs for the corresponding pods.

## Example: Hubble Adapter

- Watches for dropped egress flows.
- Extracts destination, protocol, and source pod info.
- Generates PolicyReport results with severity "high" and category "egress".

## Example: KubeArmor Adapter

- Watches for runtime security alerts.
- Maps KubeArmor severity (1-10) to PolicyReport severity levels.
- Populates PolicyReport results with policy name, rule, and message.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## Maintainer

- [bakito](https://github.com/bakito)

## References

- [Kyverno PolicyReport CRD](https://github.com/kubernetes-sigs/wg-policy-prototypes/tree/master/policy-report)
- [Cilium Hubble](https://github.com/cilium/hubble)
- [KubeArmor](https://github.com/kubearmor/kubearmor)
- 

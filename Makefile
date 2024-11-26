# Include toolbox tasks
include ./.toolbox.mk

rbac: tb.controller-gen ## Generate RBAC resources
	$(TB_CONTROLLER_GEN) crd rbac:roleName=policy-report-publisher paths="./pkg/..." output:crd:artifacts:config=config/crd/bases

lint: tb.golangci-lint
	$(TB_GOLANGCI_LINT) run --fix

# go mod tidy
tidy:
	go mod tidy

all: tidy rbac lint

pf-hubble:
	kubectl port-forward -n kube-system svc/hubble-relay 32766:443
pf-hubble-insecure:
	kubectl port-forward -n kube-system svc/hubble-relay 32766:80
pf-kubearmor:
	kubectl port-forward -n kubearmor svc/kubearmor 32767:32767

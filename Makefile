# Include toolbox tasks
include ./.toolbox.mk

rbac: tb.controller-gen ## Generate RBAC resources
	$(TB_CONTROLLER_GEN) crd rbac:roleName=policy-report-publisher paths="./internal/..." output:crd:artifacts:config=config/crd/bases

lint: tb.golangci-lint
	@echo "Lint Main"
	@$(TB_GOLANGCI_LINT) run --fix
	@echo "Lint Shared"
	@cd shared && $(TB_GOLANGCI_LINT) run --fix --config ../.golangci.yaml
	@for dir in plugins/*/; do \
		plugin_name=$$(basename $$dir); \
		echo "Lint Plugin $$plugin_name"; \
		(cd $$dir && $(TB_GOLANGCI_LINT) run --fix --config ../../.golangci.yaml); \
	done



# go mod tidy
tidy:
	@echo "Go mod tidy Main"
	@go mod tidy
	@echo "Go mod tidy Shared"
	@cd shared && go mod tidy
	@for dir in plugins/*/; do \
		plugin_name=$$(basename $$dir); \
		echo "Go mod tidy Plugin $$plugin_name"; \
		(cd $$dir && go mod tidy); \
	done
all: tidy rbac lint

pf-hubble:
	kubectl port-forward -n kube-system svc/hubble-relay 32766:443
pf-hubble-insecure:
	kubectl port-forward -n kube-system svc/hubble-relay 32766:80
pf-kubearmor:
	kubectl port-forward -n kubearmor svc/kubearmor 32767:32767

.PHONY: plugins

# Build each plugin as standalone binary
plugins:
	@mkdir -p bin/plugins
	@for dir in plugins/*/; do \
		plugin_name=$$(basename $$dir); \
		echo "Building Plugin $$plugin_name"; \
		(cd $$dir && go build -o ../../bin/plugins/$${plugin_name}.so); \
	done
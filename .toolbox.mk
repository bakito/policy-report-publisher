## toolbox - start
## Generated with https://github.com/bakito/toolbox

## Current working directory
TB_LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
TB_LOCALBIN ?= $(TB_LOCALDIR)/bin
$(TB_LOCALBIN):
	mkdir -p $(TB_LOCALBIN)

## Tool Binaries
TB_CONTROLLER_GEN ?= $(TB_LOCALBIN)/controller-gen
TB_GOLANGCI_LINT ?= $(TB_LOCALBIN)/golangci-lint

## Tool Versions
# renovate: packageName=sigs.k8s.io/controller-tools/cmd/controller-gen
TB_CONTROLLER_GEN_VERSION ?= v0.17.3
# renovate: packageName=github.com/golangci/golangci-lint/v2/cmd/golangci-lint
TB_GOLANGCI_LINT_VERSION ?= v2.1.1

## Tool Installer
.PHONY: tb.controller-gen
tb.controller-gen: $(TB_CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(TB_CONTROLLER_GEN): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/controller-gen || GOBIN=$(TB_LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(TB_CONTROLLER_GEN_VERSION)
.PHONY: tb.golangci-lint
tb.golangci-lint: $(TB_GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(TB_GOLANGCI_LINT): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/golangci-lint || GOBIN=$(TB_LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(TB_GOLANGCI_LINT_VERSION)

## Reset Tools
.PHONY: tb.reset
tb.reset:
	@rm -f \
		$(TB_LOCALBIN)/controller-gen \
		$(TB_LOCALBIN)/golangci-lint

## Update Tools
.PHONY: tb.update
tb.update: tb.reset
	toolbox makefile --renovate -f $(TB_LOCALDIR)/Makefile \
		sigs.k8s.io/controller-tools/cmd/controller-gen@github.com/kubernetes-sigs/controller-tools \
		github.com/golangci/golangci-lint/v2/cmd/golangci-lint
## toolbox - end
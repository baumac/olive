BIN := $(CURDIR)/bin
VACUUM_VERSION := v0.33.2
VACUUM := $(BIN)/vacuum
SPECS := specs/api/public-openapi.yaml

.PHONY: install-tools lint lint-spec

install-tools: $(VACUUM)

$(VACUUM):
	GOBIN=$(BIN) go install github.com/daveshanley/vacuum@$(VACUUM_VERSION)

lint: lint-spec

lint-spec: $(VACUUM)
	$(VACUUM) lint --details --fail-severity warn --ruleset specs/api/vacuum-ruleset.yaml $(SPECS)

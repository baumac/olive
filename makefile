BIN := $(CURDIR)/bin
SPEC := specs/api/public-openapi.yaml

VACUUM_VERSION := v0.33.2
VACUUM := $(BIN)/vacuum

OAPI_CODEGEN_VERSION := v2.8.2
OAPI_CODEGEN := $(BIN)/oapi-codegen

OPENAPI_GENERATOR_IMAGE := openapitools/openapi-generator-cli:v7.27.0

.PHONY: install-tools lint lint-spec generate generate-sdk-go generate-sdk-ts check-generate test-sdk test-sdk-go test-sdk-ts

install-tools: $(VACUUM) $(OAPI_CODEGEN)

$(VACUUM):
	GOBIN=$(BIN) go install github.com/daveshanley/vacuum@$(VACUUM_VERSION)

$(OAPI_CODEGEN):
	GOBIN=$(BIN) go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)

lint: lint-spec check-generate

lint-spec: $(VACUUM)
	$(VACUUM) lint --details --fail-severity warn --ruleset specs/api/vacuum-ruleset.yaml $(SPEC)

generate: generate-sdk-go generate-sdk-ts

generate-sdk-go: $(OAPI_CODEGEN)
	cd sdk/go && $(OAPI_CODEGEN) -config oapi-codegen.yaml $(CURDIR)/$(SPEC)

# The generator does not delete the files of removed schemas.
generate-sdk-ts:
	rm -rf sdk/typescript/src
	docker run --rm -u "$$(id -u):$$(id -g)" -v "$(CURDIR):/local" -w /local $(OPENAPI_GENERATOR_IMAGE) generate \
		--generator-name typescript-fetch \
		--input-spec $(SPEC) \
		--output sdk/typescript/src \
		--template-dir sdk/typescript/templates \
		--global-property apiDocs=false,modelDocs=false \
		--additional-properties supportsES6=true,importFileExtension=.js,useSingleRequestParameter=true,withInterfaces=true,legacyDiscriminatorBehavior=false

check-generate: generate
	git diff --exit-code -- sdk
	@test -z "$$(git ls-files --others --exclude-standard -- sdk)" || { git ls-files --others --exclude-standard -- sdk; echo "New generated files are not in git. Run make generate and add them."; exit 1; }

test-sdk: test-sdk-go test-sdk-ts

test-sdk-go:
	cd sdk/go && go test ./...

test-sdk-ts:
	npm ci
	npm test --workspace @reviews/sdk

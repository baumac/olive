BIN := $(CURDIR)/bin
SPEC := specs/api/public-openapi.yaml

VACUUM_VERSION := v0.33.2
VACUUM := $(BIN)/vacuum

OAPI_CODEGEN_VERSION := v2.8.2
OAPI_CODEGEN := $(BIN)/oapi-codegen

OPENAPI_GENERATOR_IMAGE := openapitools/openapi-generator-cli:v7.27.0

APP_BIN := $(BIN)/reviews-app
APP_DATA := app/pb_data
UI_DIST := app/internal/webui/dist

# npm ci runs again only when the lockfile changes.
NODE_MODULES := node_modules/.package-lock.json

.PHONY: install-tools lint lint-spec lint-go lint-ui generate generate-sdk-go generate-sdk-ts check-generate test test-sdk test-sdk-go test-sdk-ts test-app test-ui test-e2e test-e2e-ui build-sdk-ts build-ui build-app run-app dev-ui

install-tools: $(VACUUM) $(OAPI_CODEGEN)

$(VACUUM):
	GOBIN=$(BIN) go install github.com/daveshanley/vacuum@$(VACUUM_VERSION)

$(OAPI_CODEGEN):
	GOBIN=$(BIN) go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)

lint: lint-spec lint-go lint-ui check-generate

lint-spec: $(VACUUM)
	$(VACUUM) lint --details --fail-severity warn --ruleset specs/api/vacuum-ruleset.yaml $(SPEC)

lint-go:
	go vet ./app/... ./app/sdk/go/...
	@test -z "$$(gofmt -l app)" || { gofmt -l app; echo "Run gofmt -w on these files."; exit 1; }

lint-ui: build-sdk-ts
	npm run typecheck --workspace @reviews/ui
	npm run typecheck --workspace @reviews/tests

generate: generate-sdk-go generate-sdk-ts

generate-sdk-go: $(OAPI_CODEGEN)
	cd app/sdk/go && $(OAPI_CODEGEN) -config oapi-codegen.yaml $(CURDIR)/$(SPEC)

# The generator does not delete the files of removed schemas.
generate-sdk-ts:
	rm -rf app/sdk/typescript/src
	docker run --rm -u "$$(id -u):$$(id -g)" -v "$(CURDIR):/local" -w /local $(OPENAPI_GENERATOR_IMAGE) generate \
		--generator-name typescript-fetch \
		--input-spec $(SPEC) \
		--output app/sdk/typescript/src \
		--template-dir app/sdk/typescript/templates \
		--global-property apiDocs=false,modelDocs=false \
		--additional-properties supportsES6=true,importFileExtension=.js,useSingleRequestParameter=true,withInterfaces=true,legacyDiscriminatorBehavior=false

check-generate: generate
	git diff --exit-code -- app/sdk
	@test -z "$$(git ls-files --others --exclude-standard -- app/sdk)" || { git ls-files --others --exclude-standard -- app/sdk; echo "New generated files are not in git. Run make generate and add them."; exit 1; }

test: test-sdk test-app test-ui

test-sdk: test-sdk-go test-sdk-ts

test-sdk-go:
	cd app/sdk/go && go test ./...

test-sdk-ts: $(NODE_MODULES)
	npm test --workspace @reviews/sdk

test-app:
	cd app && go test ./...

test-ui: build-sdk-ts
	npm test --workspace @reviews/ui

# The end-to-end tests start ./bin/reviews-app. See decision 00006.
test-e2e: build-app
	npm test --workspace @reviews/tests

test-e2e-ui: build-app
	npm run test:ui --workspace @reviews/tests

$(NODE_MODULES): package-lock.json
	npm ci

# The UI imports the built SDK from app/sdk/typescript/dist.
build-sdk-ts: $(NODE_MODULES)
	npm run build --workspace @reviews/sdk

# vite build empties the folder, so the target makes .gitkeep again. The Go
# embed needs a file in the folder.
build-ui: build-sdk-ts
	npm run build --workspace @reviews/ui
	touch $(UI_DIST)/.gitkeep

build-app: build-ui
	go build -o $(APP_BIN) ./app

run-app:
	go run ./app serve --dir $(APP_DATA)

# Run make run-app at the same time. The Vite dev server sends /v1 to it.
dev-ui: build-sdk-ts
	npm run dev --workspace @reviews/ui

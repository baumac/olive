BIN := $(CURDIR)/bin

# TOOLS_GO builds the tools. Its version changes the embedded spec of the Go
# SDK. A new version gets a new folder of tools.
TOOLS_GO := go1.27.3
TOOLS_BIN := $(BIN)/tools-$(TOOLS_GO)
SPEC := specs/api/public-openapi.yaml

# renovate: depName=github.com/daveshanley/vacuum
VACUUM_VERSION := v0.33.2
VACUUM := $(TOOLS_BIN)/vacuum

# renovate: depName=github.com/oapi-codegen/oapi-codegen/v2
OAPI_CODEGEN_VERSION := v2.8.2
OAPI_CODEGEN := $(TOOLS_BIN)/oapi-codegen

# renovate: depName=golang.org/x/vuln
GOVULNCHECK_VERSION := v1.9.0
GOVULNCHECK := $(TOOLS_BIN)/govulncheck

OPENAPI_GENERATOR_IMAGE := openapitools/openapi-generator-cli:v7.27.0
RENOVATE_IMAGE := ghcr.io/renovatebot/renovate:44.239.1
RENOVATE_REPORT := $(BIN)/renovate-report.json

COVERAGE_APP := $(BIN)/coverage-app.out
COVERAGE_MIN_APP := 90

APP_BIN := $(BIN)/reviews-app
APP_DATA := app/api/pb_data
UI_DIST := app/api/internal/webui/dist

# npm ci runs again only when the lockfile changes.
NODE_MODULES := node_modules/.package-lock.json

.PHONY: install-tools install-browsers install-browser-deps check-ci-matrix lint lint-spec lint-go lint-ui lint-eslint security-scan generate generate-sdk-go generate-sdk-ts check-generate test test-sdk test-sdk-go test-sdk-ts test-app test-race test-ui test-coverage test-coverage-app test-coverage-ui test-e2e test-e2e-ui build-sdk-ts build-ui build-app build-image run-app dev-ui renovate-local renovate

install-tools: $(VACUUM) $(OAPI_CODEGEN) $(GOVULNCHECK)

$(VACUUM):
	GOBIN=$(TOOLS_BIN) GOTOOLCHAIN=$(TOOLS_GO) go install github.com/daveshanley/vacuum@$(VACUUM_VERSION)

$(OAPI_CODEGEN):
	GOBIN=$(TOOLS_BIN) GOTOOLCHAIN=$(TOOLS_GO) go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)

$(GOVULNCHECK):
	GOBIN=$(TOOLS_BIN) GOTOOLCHAIN=$(TOOLS_GO) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

lint: lint-spec lint-go lint-ui lint-eslint check-generate check-ci-matrix

lint-spec: $(VACUUM)
	$(VACUUM) lint --details --fail-severity warn --ruleset specs/api/vacuum-ruleset.yaml $(SPEC)

lint-go:
	go vet ./app/api/... ./specs/api/sdk/server/go/...
	@test -z "$$(gofmt -l app/api specs/api/sdk/server/go)" || { gofmt -l app/api specs/api/sdk/server/go; echo "Run gofmt -w on these files."; exit 1; }

lint-ui: build-sdk-ts
	npm run typecheck --workspace @reviews/ui
	npm run typecheck --workspace @reviews/tests

lint-eslint: $(NODE_MODULES)
	npm run lint --workspace @reviews/ui
	npm run lint --workspace @reviews/tests

# security-scan is not part of lint. A new advisory in the database can fail
# it without a code change. govulncheck reports only the code that the
# modules call.
security-scan: $(GOVULNCHECK)
	$(GOVULNCHECK) -C app/api ./...
	$(GOVULNCHECK) -C specs/api/sdk/server/go ./...

generate: generate-sdk-go generate-sdk-ts

generate-sdk-go: $(OAPI_CODEGEN)
	cd specs/api/sdk/server/go && $(OAPI_CODEGEN) -config oapi-codegen.yaml $(CURDIR)/$(SPEC)

# The generator does not delete the files of removed schemas.
generate-sdk-ts:
	rm -rf specs/api/sdk/client/typescript/src
	docker run --rm -u "$$(id -u):$$(id -g)" -v "$(CURDIR):/local" -w /local $(OPENAPI_GENERATOR_IMAGE) generate \
		--generator-name typescript-fetch \
		--input-spec $(SPEC) \
		--output specs/api/sdk/client/typescript/src \
		--template-dir specs/api/sdk/client/typescript/templates \
		--global-property apiDocs=false,modelDocs=false \
		--additional-properties supportsES6=true,importFileExtension=.js,useSingleRequestParameter=true,withInterfaces=true,legacyDiscriminatorBehavior=false

GENERATED := specs/api/sdk/server/go/reviewsapi.gen.go specs/api/sdk/client/typescript/src
CHECK_DIR := .check-generate

# check-generate keeps a copy of the generated files, generates them again,
# and compares. Edits to the hand-written SDK files do not change the result.
check-generate:
	rm -rf $(CHECK_DIR)
	mkdir -p $(CHECK_DIR)
	tar -cf - $(GENERATED) | tar -xf - -C $(CHECK_DIR)
	$(MAKE) --no-print-directory generate
	@diff -r $(CHECK_DIR)/specs/api/sdk/server/go/reviewsapi.gen.go specs/api/sdk/server/go/reviewsapi.gen.go && \
		diff -r $(CHECK_DIR)/specs/api/sdk/client/typescript/src specs/api/sdk/client/typescript/src || \
		{ echo "The generated SDKs do not match the spec. Run make generate."; exit 1; }
	rm -rf $(CHECK_DIR)
	@test -z "$$(git ls-files --others --exclude-standard -- $(GENERATED))" || { git ls-files --others --exclude-standard -- $(GENERATED); echo "New generated files are not in git. Run make generate and add them."; exit 1; }

test: test-sdk test-app test-ui

test-sdk: test-sdk-go test-sdk-ts

test-sdk-go:
	cd specs/api/sdk/server/go && go test ./...

test-sdk-ts: $(NODE_MODULES)
	npm test --workspace @reviews/sdk

# The race detector makes the tests of the app about 7 times slower. The
# default timeout of go test is 10 minutes. The CI job stops after 20 minutes.
GO_TEST_TIMEOUT := 15m

# test-app has no race detector. test-race and test-coverage-app have it. See
# decision 00010.
test-app:
	cd app/api && go test -timeout $(GO_TEST_TIMEOUT) ./...

test-race:
	cd app/api && go test -race -timeout $(GO_TEST_TIMEOUT) ./...

test-ui: build-sdk-ts
	npm test --workspace @reviews/ui

test-coverage: test-coverage-app test-coverage-ui

# The target fails when the Go tests cover less than COVERAGE_MIN_APP percent
# of the statements of the app. It uses the race detector like test-race.
# The coverage leaves out internal/testapp. Only the tests use it.
test-coverage-app:
	mkdir -p $(BIN)
	cd app/api && go test -race -timeout $(GO_TEST_TIMEOUT) \
		-coverpkg=$$(go list ./... | grep -v /internal/testapp$$ | paste -sd, -) -coverprofile=$(COVERAGE_APP) ./...
	@cd app/api && go tool cover -func=$(COVERAGE_APP) | awk -v min=$(COVERAGE_MIN_APP) \
		'/^total:/ { sub("%", "", $$3); printf "Go coverage: %s%%. The minimum is %s%%.\n", $$3, min; if ($$3 + 0 < min) exit 1 }'

# app/ui/vite.config.ts has the minimum of the UI.
test-coverage-ui: build-sdk-ts
	npm run coverage --workspace @reviews/ui

# The end-to-end tests start ./bin/reviews-app. See decision 00010.
# PROJECT runs one Playwright project, such as make test-e2e PROJECT=webkit.
# E2E_FILE runs the spec files that match it, such as
# E2E_FILE=e2e/auth.spec.ts. Make also reads it from the environment.
E2E_ARGS = $(strip $(if $(PROJECT),--project=$(PROJECT)) $(E2E_FILE))
test-e2e: build-app
	npm test --workspace @reviews/tests $(if $(E2E_ARGS),-- $(E2E_ARGS))

# BROWSER installs one browser, such as make install-browsers BROWSER=firefox.
install-browsers: $(NODE_MODULES)
	cd tests && npx playwright install $(BROWSER)

# install-browser-deps installs the system packages of the browsers. It
# needs sudo on Linux.
install-browser-deps: $(NODE_MODULES)
	cd tests && npx playwright install-deps $(BROWSER)

# check-ci-matrix fails when the e2e matrix of CI misses a Playwright project.
check-ci-matrix: $(NODE_MODULES)
	node tests/check-ci-matrix.mjs

test-e2e-ui: build-app
	npm run test:ui --workspace @reviews/tests

$(NODE_MODULES): package-lock.json
	npm ci

# The UI imports the built SDK from specs/api/sdk/client/typescript/dist.
build-sdk-ts: $(NODE_MODULES)
	npm run build --workspace @reviews/sdk

# vite build empties the folder, so the target makes .gitkeep again. The Go
# embed needs a file in the folder.
build-ui: build-sdk-ts
	npm run build --workspace @reviews/ui
	touch $(UI_DIST)/.gitkeep

build-app: build-ui
	go build -o $(APP_BIN) ./app/api

# The docker job of CI and developers build the image the same way. See
# decision 00010.
build-image:
	docker build -f deploy/Dockerfile -t reviews-app:local .

run-app:
	go run ./app/api serve --dir $(APP_DATA)

# Run make run-app at the same time. The Vite dev server sends /v1 to it.
dev-ui: build-sdk-ts
	npm run dev --workspace @reviews/ui

# renovate-local changes no file and opens no pull request. A git worktree
# has its git folder out of the work folder. Docker gets the two folders.
# Without GITHUB_COM_TOKEN, Renovate skips the lookups of the GitHub Actions.
renovate-local: GIT_COMMON_DIR = $(shell git rev-parse --path-format=absolute --git-common-dir)
renovate-local:
	mkdir -p $(BIN)
	docker run --rm -u "$$(id -u):$$(id -g)" -v "$(CURDIR):$(CURDIR)" -v "$(GIT_COMMON_DIR):$(GIT_COMMON_DIR)" -w "$(CURDIR)" \
		-e GITHUB_COM_TOKEN -e LOG_LEVEL $(RENOVATE_IMAGE) renovate \
		--platform=local --dry-run=lookup --report-type=file --report-path=$(RENOVATE_REPORT)
	@echo "The report with the updates is in $(RENOVATE_REPORT)."

# renovate opens or updates the pull request on GitHub. The Renovate workflow
# runs it. RENOVATE_REPOSITORIES has the repository, such as owner/name.
renovate:
	@test -n "$$RENOVATE_TOKEN" || { echo "Set RENOVATE_TOKEN. See decision 00010."; exit 1; }
	@test -n "$$RENOVATE_REPOSITORIES" || { echo "Set RENOVATE_REPOSITORIES, such as owner/name."; exit 1; }
	docker run --rm -e RENOVATE_TOKEN -e RENOVATE_REPOSITORIES -e RENOVATE_ONBOARDING=false -e LOG_LEVEL \
		$(RENOVATE_IMAGE) renovate

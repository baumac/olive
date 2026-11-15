# syntax=docker/dockerfile:1
# The build context is the root of the repository. See decision 00022.

FROM node:22.23.5-alpine3.24 AS ui
WORKDIR /src
COPY package.json package-lock.json ./
COPY app/sdk/typescript/package.json app/sdk/typescript/
COPY app/ui/package.json app/ui/
# The root package.json names this workspace. npm ci needs its package.json.
COPY tests/package.json tests/
RUN npm ci --workspace @reviews/sdk --workspace @reviews/ui
COPY app/sdk/typescript app/sdk/typescript
COPY app/ui app/ui
RUN npm run build --workspace @reviews/sdk && npm run build --workspace @reviews/ui

FROM golang:1.27.4-alpine3.24 AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY app/go.mod app/go.sum app/
COPY app/sdk/go/go.mod app/sdk/go/go.sum app/sdk/go/
RUN cd app && go mod download
COPY app app
COPY --from=ui /src/app/internal/webui/dist app/internal/webui/dist
# PocketBase uses a SQLite driver in Go. The build needs no C compiler.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/reviews-app ./app

FROM alpine:3.24.4
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 reviews \
    && mkdir /pb_data \
    && chown reviews:reviews /pb_data
COPY --from=build /out/reviews-app /usr/local/bin/reviews-app
USER reviews
VOLUME /pb_data
EXPOSE 8090
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD wget -q -O /dev/null http://127.0.0.1:8090/api/health || exit 1
ENTRYPOINT ["reviews-app"]
CMD ["serve", "--http=0.0.0.0:8090", "--dir=/pb_data", "--encryptionEnv=PB_ENCRYPTION_KEY"]

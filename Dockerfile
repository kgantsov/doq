FROM node:16.3.0-alpine3.13 AS frontend

WORKDIR /doq/
COPY ./ ./
WORKDIR /doq/admin_ui
RUN npm install
RUN npm run build

RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
# -0 means no compression.  Needed because go's
# tz loader doesn't handle compressed data.
RUN zip -r -0 /zoneinfo.zip .

FROM --platform=$BUILDPLATFORM golang:1.22.2 AS builder

WORKDIR /src

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage bind mounts to go.sum and go.mod to avoid having to copy them into
# the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x
COPY --from=frontend /doq/admin_ui/dist/ ./cmd/server/

# This is the architecture you're building for, which is passed in by the builder.
# Placing it here allows the previous steps to be cached across architectures.
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app ./cmd/server

FROM alpine

ENV UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser
USER appuser

ENV ZONEINFO=/zoneinfo.zip
COPY --from=frontend /zoneinfo.zip /

COPY --from=builder /app /
COPY --from=frontend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# CMD ["/app --port 8780"]

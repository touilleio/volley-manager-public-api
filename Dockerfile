FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.24 AS builder

ARG TARGETOS
ARG TARGETARCH

ARG GIT_COMMIT
ARG BUILD_DATE
ARG VERSION

WORKDIR /src

COPY go.* .

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go mod download

COPY . .

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o volley-manager-public-api \
    -ldflags "-X github.com/touilleio/volley-manager-public-api/internal/buildinfo.GitCommit=${GIT_COMMIT} \
			-X github.com/touilleio/volley-manager-public-api/internal/buildinfo.BuildDate=${BUILD_DATE} \
			-X github.com/touilleio/volley-manager-public-api/internal/buildinfo.Version=${VERSION}" \
    .

FROM --platform=$BUILDPLATFORM alpine:3.24
RUN apk add --no-cache tzdata

# Games snapshot dir; named volumes inherit this ownership, so nobody can write to them.
RUN mkdir -p /data && chown nobody:nobody /data

USER nobody

ENTRYPOINT ["/volley-manager-public-api"]

EXPOSE 8080

COPY --from=builder /src/volley-manager-public-api .
COPY ./static /static

FROM --platform=$BUILDPLATFORM golang:1.20-alpine3.18 as builder

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
    -ldflags "-X github.com/sqooba/go-common/version.GitCommit=${GIT_COMMIT} \
    			-X github.com/sqooba/go-common/version.BuildDate=${BUILD_DATE} \
    			-X github.com/sqooba/go-common/version.Version=${VERSION}" \
    .

FROM --platform=$BUILDPLATFORM alpine:3.18
COPY --from=builder /src/volley-manager-public-api .
COPY ./static /static

USER nobody

ENTRYPOINT ["/volley-manager-public-api"]

EXPOSE 8080

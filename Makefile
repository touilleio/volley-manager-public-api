
VERSION=v1.0.0
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOLINT=golangci-lint run
BUILD_PLATFORM=linux/amd64
PACKAGE_PLATFORM=$(BUILD_PLATFORM)
VERSION_MAJOR=$(shell echo $(VERSION) | cut -f1 -d.)
VERSION_MINOR=$(shell echo $(VERSION) | cut -f2 -d.)
GO_PACKAGE_PREFIX=touilleio/volley-manager-public-api
DOCKER_REGISTRY=
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')

all: ensure build package

ensure:
	env GOOS=linux $(GOCMD) mod download

clean:
	$(GOCLEAN)

lint:
	$(GOLINT) ...

build:
	env GOOS=linux CGO_ENABLED=0 \
		$(GOBUILD) \
		-ldflags "-X github.com/sqooba/go-common/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} \
			-X github.com/sqooba/go-common/version.BuildDate=${BUILD_DATE} \
			-X github.com/sqooba/go-common/version.Version=${VERSION}" \
		-o volley-manager-public-api .

package:
	docker buildx build -f Dockerfile \
		--platform $(BUILD_PLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=${GIT_COMMIT}${GIT_DIRTY} \
		-t ${DOCKER_REGISTRY}${GO_PACKAGE_PREFIX}:$(VERSION) \
		.

test:
	$(GOTEST) ./...

release:
	docker push ${DOCKER_REGISTRY}${GO_PACKAGE_PREFIX}:$(VERSION)

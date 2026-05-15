PROVIDER = provider-concourse
BINARY = $(PROVIDER)
IMG ?= ghcr.io/maximilianbraun/$(PROVIDER):latest

GO := go
LOCALBIN ?= $(shell pwd)/bin
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.20.1

.PHONY: all
all: generate build

.PHONY: generate
generate: controller-gen
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."
	"$(CONTROLLER_GEN)" crd paths="./apis/..." output:crd:artifacts:config=package/crds

.PHONY: manifests
manifests: generate

.PHONY: build
build:
	$(GO) build -o bin/$(BINARY) ./cmd/provider/

.PHONY: test
test:
	$(GO) test ./... -coverprofile cover.out

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push:
	docker push $(IMG)

.PHONY: dev
dev: generate build
	./bin/$(BINARY) --debug

.PHONY: clean
clean:
	rm -rf bin/ package/crds/

$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN="$(LOCALBIN)" $(GO) install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

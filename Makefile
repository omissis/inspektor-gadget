TAG := `git describe --tags --always`
VERSION :=

CONTAINER_RUNTIME ?= docker
CONTAINER_REPO ?= ghcr.io/inspektor-gadget/inspektor-gadget

GADGET_CONTAINERS = \
	gadget-default-container \
	gadget-core-container

IMAGE_TAG ?= $(shell ./tools/image-tag branch)

LDFLAGS := "-X main.version=$(VERSION) \
-X main.gadgetimage=$(CONTAINER_REPO):$(IMAGE_TAG) \
-extldflags '-static'"

LOCAL_GADGET_TARGETS = \
	local-gadget-linux-amd64 \
	local-gadget-linux-arm64

LIVENESS_PROBE ?= true

MINIKUBE ?= $(shell pwd)/bin/minikube-$(MINIKUBE_VERSION)
MINIKUBE_DRIVER ?= docker
MINIKUBE_VERSION ?= v1.27.0
MINIKUBE_SETUP_TARGETS = \
	minikube-setup-docker \
	minikube-setup-containerd \
	minikube-setup-cri-o

KUBERNETES_VERSION ?= v1.24.6
KUBERNETES_DISTRIBUTION ?= ""

KUBECTL_GADGET_TARGETS = \
	kubectl-gadget-linux-amd64 \
	kubectl-gadget-linux-arm64 \
	kubectl-gadget-darwin-amd64 \
	kubectl-gadget-darwin-arm64 \
	kubectl-gadget-windows-amd64

GOHOSTOS ?= $(shell go env GOHOSTOS)
GOHOSTARCH ?= $(shell go env GOHOSTARCH)

KUBERNETES_ARCHITECTURE ?= $(GOHOSTARCH)

ENABLE_BTFGEN ?= false

BPFTOOL ?= bpftool
ARCH ?= $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/' | sed 's/ppc64le/powerpc/' | sed 's/mips.*/mips/')

LOCAL_GADGET_INTEGRATION_TEST_TARGETS = \
	local-gadget-integration-tests-docker \
	local-gadget-integration-tests-containerd \
	local-gadget-integration-tests-cri-o

# Adds a '-dirty' suffix to version string if there are uncommitted changes
changes := $(shell git status --porcelain)
ifeq ($(changes),)
	VERSION := $(TAG)
else
	VERSION := $(TAG)-dirty
endif

pvpath := $(shell command -v pv 2>/dev/null || true)
ifeq ($(pvpath),)
	PV :=
else
	PV := | $(pvpath)
endif

# export variables that are used in Makefile.btfgen as well.
export BPFTOOL ARCH

include crd.mk
include tests.mk

.DEFAULT_GOAL := build
.PHONY: build
build: manifests generate kubectl-gadget gadget-default-container

.PHONY: all
all: build local-gadget

# make does not allow implicit rules (with '%') to be phony so let's use
# the 'phony-explicit' dependency to make implicit rules inherit the phony
# attribute
.PHONY: phony-explicit
phony-explicit:

ebpf-objects:
	docker run -it --rm \
		--name ebpf-object-builder \
		--user $(shell id -u):$(shell id -g) \
		-v $(shell pwd):/work ghcr.io/inspektor-gadget/inspektor-gadget-ebpf-builder

ebpf-objects-outside-docker:
	TARGET=arm64 go generate ./...
	TARGET=amd64 go generate ./...

# local-gadget
.PHONY: list-local-gadget-targets
list-local-gadget-targets:
	@echo $(LOCAL_GADGET_TARGETS)

.PHONY: build-local-gadget-all build-local-gadget
build-local-gadget-all: $(LOCAL_GADGET_TARGETS) build-local-gadget

build-local-gadget: build-local-gadget-linux-$(GOHOSTARCH)
	cp local-gadget-linux-$(GOHOSTARCH) local-gadget

build-local-gadget-%: phony-explicit
	echo Building local-gadget-$* && \
	export GOOS=$(shell echo $* |cut -f1 -d-) GOARCH=$(shell echo $* |cut -f2 -d-) && \
	docker buildx build -t local-gadget-$*-builder -f Dockerfiles/local-gadget.Dockerfile \
		--build-arg GOOS=linux --build-arg GOARCH=$${GOARCH} --build-arg VERSION=$(VERSION) . && \
	docker run --rm --entrypoint cat local-gadget-$*-builder local-gadget-$* > local-gadget-$* && \
	chmod +x local-gadget-$*

.PHONY: list-kubectl-gadget-targets
list-kubectl-gadget-targets:
	@echo $(KUBECTL_GADGET_TARGETS)

.PHONY: kubectl-gadget-all kubectl-gadget
kubectl-gadget-all: $(KUBECTL_GADGET_TARGETS) kubectl-gadget

kubectl-gadget: kubectl-gadget-$(GOHOSTOS)-$(GOHOSTARCH)
	cp kubectl-gadget-$(GOHOSTOS)-$(GOHOSTARCH) kubectl-gadget

kubectl-gadget-%: phony-explicit
	export GO111MODULE=on CGO_ENABLED=0 && \
	export GOOS=$(shell echo $* |cut -f1 -d-) GOARCH=$(shell echo $* |cut -f2 -d-) && \
	go build -ldflags $(LDFLAGS) \
		-o kubectl-gadget-$${GOOS}-$${GOARCH} \
		github.com/inspektor-gadget/inspektor-gadget/cmd/kubectl-gadget

.PHONY: install/kubectl-gadget
install/kubectl-gadget: kubectl-gadget-$(GOHOSTOS)-$(GOHOSTARCH)
	mkdir -p ~/.local/bin/
	cp kubectl-gadget-$(GOHOSTOS)-$(GOHOSTARCH) ~/.local/bin/kubectl-gadget

gadget-container-all: $(GADGET_CONTAINERS)

gadget-%-container:
	docker buildx build -t $(CONTAINER_REPO):$(IMAGE_TAG)$(if $(findstring core,$*),-core,) -f Dockerfiles/gadget-$*.Dockerfile \
		--build-arg ENABLE_BTFGEN=$(ENABLE_BTFGEN) .

push-gadget-%-container:
	docker push $(CONTAINER_REPO):$(IMAGE_TAG)$(if $(findstring core,$*),-core,)

# kubectl-gadget container image
.PHONY: kubectl-gadget-container
kubectl-gadget-container:
	docker buildx build -t kubectl-gadget -f Dockerfiles/kubectl-gadget.Dockerfile \
	--build-arg IMAGE_TAG=$(IMAGE_TAG) .

# tests
.PHONY: test
test:
	go test -test.v ./...

.PHONY: controller-tests
controller-tests: kube-apiserver etcd kubectl
	ACK_GINKGO_DEPRECATIONS=1.16.4 \
	TEST_ASSET_KUBE_APISERVER=$(KUBE_APISERVER_BIN) \
	TEST_ASSET_ETCD=$(ETCD_BIN) \
	TEST_ASSET_KUBECTL=$(KUBECTL_BIN) \
	go test -test.v ./pkg/controllers/... -controller-test

.PHONY: gadgets-unit-tests
gadgets-unit-tests:
	go test -test.v -exec sudo ./pkg/gadgets/...

.PHONY: local-gadget-tests
local-gadget-tests:
	# Compile and execute in separate commands because Go might not be
	# available in the root environment
	go test -c ./pkg/local-gadget-manager \
		-tags withebpf
	sudo ./local-gadget-manager.test -test.v -root-test $${LOCAL_GADGET_TESTS_PARAMS}
	rm -f ./local-gadget-manager.test

# INTEGRATION_TESTS_PARAMS can be used to pass additional parameters locally e.g
# INTEGRATION_TESTS_PARAMS="-run TestExecsnoop -v -no-deploy-ig -no-deploy-spo" make integration-tests
.PHONY: inspektor-gadget-integration-tests local-gadget-integration-tests
inspektor-gadget-integration-tests: kubectl-gadget
	KUBECTL_GADGET="$(shell pwd)/kubectl-gadget" \
		go test ./integration/inspektor-gadget/... \
			-integration \
			-timeout 30m \
			-k8s-distro $(KUBERNETES_DISTRIBUTION) \
			-k8s-arch $(KUBERNETES_ARCHITECTURE) \
			-image $(CONTAINER_REPO):$(IMAGE_TAG) \
			$$INTEGRATION_TESTS_PARAMS

local-gadget-integration-tests-all: $(LOCAL_GADGET_INTEGRATION_TEST_TARGETS) test

local-gadget-integration-tests: local-gadget-integration-tests-$(CONTAINER_RUNTIME)

local-gadget-integration-tests-%: build-local-gadget
	@export MINIKUBE_PROFILE=minikube-$* && \
	echo "Checking minikube with profile $${MINIKUBE_PROFILE} is running ..." && \
	$(MINIKUBE) status -p $${MINIKUBE_PROFILE} -f {{.APIServer}} >/dev/null || (echo "Error: $${MINIKUBE_PROFILE} not running, exiting ..." && exit 1) && \
	echo "Preparing minikube with profile $${MINIKUBE_PROFILE} for testing ..." && \
	$(MINIKUBE) cp local-gadget-linux-${GOHOSTARCH} $${MINIKUBE_PROFILE}:/bin/local-gadget >/dev/null && \
	$(MINIKUBE) ssh sudo chmod +x /bin/local-gadget && \
	go test -c -o local-gadget-integration.test ./integration/local-gadget && \
	$(MINIKUBE) cp local-gadget-integration.test $${MINIKUBE_PROFILE}:/bin/local-gadget-integration.test >/dev/null && \
	$(MINIKUBE) ssh sudo chmod +x /bin/local-gadget-integration.test && \
	rm local-gadget-integration.test && \
	$(MINIKUBE) -p $${MINIKUBE_PROFILE} ssh "sudo ln -sf /var/lib/minikube/binaries/$(KUBERNETES_VERSION)/kubectl /bin/kubectl" && \
	$(MINIKUBE) -p $${MINIKUBE_PROFILE} ssh "sudo ln -sf /etc/kubernetes/admin.conf /root/.kube/config" && \
	echo "Running test in minikube with profile $${MINIKUBE_PROFILE} ..." && \
	$(MINIKUBE) -p $${MINIKUBE_PROFILE} ssh "sudo local-gadget-integration.test -test.v -integration -container-runtime $* $${INTEGRATION_TESTS_PARAMS}"

.PHONY: generate-documentation lint
generate-documentation:
	go run -tags docs cmd/gen-doc/gen-doc.go -repo $(shell pwd)

lint:
# This version number must be kept in sync with CI workflow lint one.
# XDG_CACHE_HOME is necessary to avoid this type of errors:
# ERRO Running error: context loading failed: failed to load packages: failed to load with go/packages: err: exit status 1: stderr: failed to initialize build cache at /.cache/go-build: mkdir /.cache: permission denied
# Process 15167 has exited with status 3
# While GOLANGCI_LINT_CACHE is used to store golangci-lint cache.
	docker run --rm --env XDG_CACHE_HOME=/tmp/xdg_home_cache \
		--env GOLANGCI_LINT_CACHE=/tmp/golangci_lint_cache \
		--user $(shell id -u):$(shell id -g) -v $(shell pwd):/app -w /app \
		golangci/golangci-lint:v1.49.0 golangci-lint run --fix

# minikube
.PHONY: minikube-download
minikube-download:
	mkdir -p bin
	test -e bin/minikube-$(MINIKUBE_VERSION) || \
	(cd bin && curl -Lo ./minikube-$(MINIKUBE_VERSION) https://github.com/kubernetes/minikube/releases/download/$(MINIKUBE_VERSION)/minikube-$(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH))
	chmod +x bin/minikube-$(MINIKUBE_VERSION)

.PHONY: minikube-clean
minikube-clean:
	$(MINIKUBE) delete -p minikube-docker
	$(MINIKUBE) delete -p minikube-containerd
	$(MINIKUBE) delete -p minikube-cri-o
	rm -rf bin

.PHONY: minikube-setup-all minikube-setup
minikube-setup-all: $(MINIKUBE_SETUP_TARGETS)

minikube-setup: minikube-setup-$(CONTAINER_RUNTIME)

minikube-setup-%: minikube-download
	$(MINIKUBE) status -p minikube-$* -f {{.APIServer}} >/dev/null || \
	$(MINIKUBE) start -p minikube-$* --driver=$(MINIKUBE_DRIVER) --kubernetes-version=$(KUBERNETES_VERSION) --container-runtime=$* --wait=all

.PHONY: minikube-inspektor-gadget-install
minikube-inspektor-gadget-install: gadget-default-container kubectl-gadget
	@echo "Image on the host:"
	docker image list --format "table {{.ID}}\t{{.Repository}}:{{.Tag}}\t{{.Size}}" |grep $(CONTAINER_REPO):$(IMAGE_TAG)
	@echo
	# Unfortunately, minikube-cache and minikube-image have bugs in older
	# versions. And new versions of minikube don't support all eBPF
	# features. So we have to keep "docker-save|docker-load" when
	# available.
	if $(MINIKUBE) docker-env >/dev/null 2>&1 ; then \
		docker save $(CONTAINER_REPO):$(IMAGE_TAG) $(PV) | (eval $$($(MINIKUBE) docker-env | grep =) && docker load) ; \
	else \
		$(MINIKUBE) image load $(CONTAINER_REPO):$(IMAGE_TAG) ; \
	fi
	@echo "Image in Minikube:"
	$(MINIKUBE) image ls --format=table | grep "$(CONTAINER_REPO)\s*|\s*$(IMAGE_TAG)" || \
		(echo "Image $(CONTAINER_REPO)\s*|\s*$(IMAGE_TAG) was not correctly loaded into Minikube" && false)
	@echo
	# Remove all resources created by Inspektor Gadget.
	./kubectl-gadget undeploy || true
	./kubectl-gadget deploy --liveness-probe=$(LIVENESS_PROBE) \
		--image-pull-policy=Never
	kubectl rollout status daemonset -n gadget gadget --timeout 30s
	@echo "Image used by the gadget pod:"
	kubectl get pod -n gadget -o yaml|grep imageID:

.PHONY: btfgen
btfgen:
	+make -f Makefile.btfgen

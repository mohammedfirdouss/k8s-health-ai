.PHONY: test build build-diagctl run install kind-up docker-build deploy smoke-readonly

test:
	go test ./...

# Quick check that the CRD API group is registered (needs kubectl + applied CRD).
smoke-readonly:
	./hack/smoke-readonly.sh

build:
	go build -o bin/manager ./cmd/manager

build-diagctl:
	go build -o bin/diagctl ./cmd/diagctl

run: build
	LLM_PROVIDER=mock ./bin/manager

install:
	./hack/install.sh

kind-up:
	kind create cluster --config hack/kind-cluster.yaml --name k8s-health-ai || true
	kubectl cluster-info --context kind-k8s-health-ai

docker-build:
	docker build -t k8s-health-ai-manager:latest .

deploy: docker-build
	kubectl apply -f config/manager/namespace.yaml
	kubectl apply -f config/rbac/service_account.yaml
	kubectl apply -f config/rbac/cluster_role.yaml
	kubectl apply -f config/rbac/cluster_role_binding.yaml
	kubectl apply -f config/manager/deployment.yaml
	kind load docker-image k8s-health-ai-manager:latest --name k8s-health-ai 2>/dev/null || true

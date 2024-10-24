KIND            := kindest/node:v1.27.3
KIND_CLUSTER    := local-dev-cluster

dev-up:
	kind create cluster \
		--image $(KIND) \
		--name $(KIND_CLUSTER) \
		--config kind-config.yaml

	kubectl wait --timeout=120s --namespace=local-path-storage --for=condition=Available deployment/local-path-provisioner

dev-down:
	kind delete cluster --name $(KIND_CLUSTER)

skaffold-dev:
	skaffold dev --port-forward -f skaffold-dev.yaml

dev: dev-up skaffold-dev dev-down
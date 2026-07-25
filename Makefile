NAMESPACE ?= go-cloud
CHART ?= chart/go-cloud

run:
	go run ./src/cmd/api/main.go

test:
	go test -v ./...

build:
	docker build -t go-cloud:latest .

helm-dep:
	helm dependency build $(CHART)

helm-install:
	kubectl create namespace $(NAMESPACE) --dry-run=client -o json | kubectl apply -f -
	helm upgrade --install go-cloud $(CHART) --namespace $(NAMESPACE) --create-namespace

helm-upgrade:
	helm upgrade go-cloud $(CHART) --namespace $(NAMESPACE)

deploy: build helm-install
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=go-cloud -n $(NAMESPACE) --timeout=120s
	kubectl get pods -n $(NAMESPACE)

test-run: k6-run

k6-run:
	helm test go-cloud --namespace $(NAMESPACE)

helm-status:
	helm status go-cloud --namespace $(NAMESPACE)

helm-history:
	helm history go-cloud --namespace $(NAMESPACE)

clean: helm-uninstall

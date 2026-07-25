NAMESPACE ?= go-cloud

run:
	go run ./src/cmd/api/main.go

test:
	go test -v ./...

build:
	docker build -t go-cloud:latest .

deploy:
	kubectl create namespace $(NAMESPACE) --dry-run=client -o json | kubectl apply -f -
	kubectl apply -f deployment/postgres-secret.yaml
	kubectl apply -f deployment/postgres-sts.yaml
	kubectl apply -f deployment/postgres-svc.yaml
	kubectl apply -f deployment/redis-deployment.yaml
	kubectl apply -f deployment/redis-svc.yaml
	kubectl apply -f deployment/app-deployment.yaml
	kubectl apply -f deployment/app-svc.yaml
	kubectl apply -f deployment/ingress.yaml
	kubectl apply -f deployment/monitoring/
	kubectl wait --for=condition=ready pod -l app=go-cloud -n $(NAMESPACE) --timeout=120s
	kubectl get pods -n $(NAMESPACE)

deploy-all: build deploy

k6-run:
	kubectl apply -f deployment/k6-job.yaml
	kubectl wait --for=condition=complete job/k6-load-test -n $(NAMESPACE) --timeout=120s
	kubectl logs job/k6-load-test -n $(NAMESPACE)
	kubectl delete -f deployment/k6-job.yaml 2>/dev/null || true

k6-logs:
	kubectl logs job/k6-load-test -n $(NAMESPACE)

clean:
	kubectl delete -f deployment/k6-job.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/monitoring/ -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/ingress.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/app-deployment.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/app-svc.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/redis-deployment.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/redis-svc.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/postgres-sts.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/postgres-svc.yaml -n $(NAMESPACE) 2>/dev/null || true
	kubectl delete -f deployment/postgres-secret.yaml -n $(NAMESPACE) 2>/dev/null || true

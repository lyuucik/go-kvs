run:
	go run ./src/cmd/api/main.go

test:
	go test -v ./...

build:
	docker build -t go-cloud:latest .

deploy:
	kubectl apply -f deployment/postgres-secret.yaml
	kubectl apply -f deployment/postgres-sts.yaml
	kubectl apply -f deployment/postgres-svc.yaml
	kubectl apply -f deployment/redis-deployment.yaml
	kubectl apply -f deployment/redis-svc.yaml
	kubectl apply -f deployment/app-deployment.yaml
	kubectl apply -f deployment/app-svc.yaml
	kubectl apply -f deployment/ingress.yaml
	kubectl apply -f deployment/monitoring/

deploy-all: build deploy

clean:
	kubectl delete -f deployment/monitoring/ 2>/dev/null || true
	kubectl delete -f deployment/ingress.yaml 2>/dev/null || true
	kubectl delete -f deployment/app-deployment.yaml 2>/dev/null || true
	kubectl delete -f deployment/app-svc.yaml 2>/dev/null || true
	kubectl delete -f deployment/redis-deployment.yaml 2>/dev/null || true
	kubectl delete -f deployment/redis-svc.yaml 2>/dev/null || true
	kubectl delete -f deployment/postgres-sts.yaml 2>/dev/null || true
	kubectl delete -f deployment/postgres-svc.yaml 2>/dev/null || true
	kubectl delete -f deployment/postgres-secret.yaml 2>/dev/null || true

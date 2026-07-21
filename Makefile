run:
	OAUTH2_REDIRECT_URL=http://127.0.0.1:8080/auth/google/callback go run main.go

mongo:
	docker compose up

test:
	go test ./model/
	go test ./repository/

build:
	go build -o budget main.go

deploy: build
	scp ./budget julien@budget.desrosiers.org:budget/budget

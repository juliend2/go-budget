run:
	go run main.go

mongo:
	docker compose up

test:
	go test ./model/
	go test ./repository/


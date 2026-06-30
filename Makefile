all:
	go run main.go

test:
	go test ./model/
	go test ./repository/

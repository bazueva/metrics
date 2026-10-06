test-cover:
	go test $$(go list ./... | grep -v '/mock') -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1
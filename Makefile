test-cover:
	go test $$(go list ./... | grep -v '/mock') -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

generate-keys:
	mkdir -p keys
	openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out keys/private.pem
	openssl pkey -in keys/private.pem -pubout -out keys/public.pem
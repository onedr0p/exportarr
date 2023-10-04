-include .env

build:
	docker build . -t exportarr:local
run:
	docker rm --force exportarr || echo ""
	docker run --name exportarr \
		-e PORT=9707 \
		-e URL="${APP_URL}" \
		-e APIKEY="${APP_API_KEY}" \
		-e LOG_LEVEL="debug" \
		-p 9707:9707 \
		-d exportarr:local ${APP_NAME}
test:
	go test -v -race -covermode atomic -coverprofile=covprofile ./...
tidy:
	go mod tidy
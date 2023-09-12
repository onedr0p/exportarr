-include .env

docker:
	docker build . -t exportarr:local
run:
	docker run --name exportarr \
		-e PORT=9707 \
		-e URL="${APP_URL}" \
		-e APIKEY="${APP_API_KEY}" \
		-p 9707:9707 \
		-d exportarr:local ${APP_NAME}
test:
	go test -v -race -covermode atomic -coverprofile=covprofile ./...

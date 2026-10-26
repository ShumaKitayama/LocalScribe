.PHONY: test test-integration wails-dev wails-build

test:
	go test ./...

test-integration:
	go test -tags integration ./...

wails-dev:
	wails dev

wails-build:
	cd frontend && npm run build
	wails build

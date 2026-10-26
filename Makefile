.PHONY: test test-integration test-frontend wails-dev wails-build package-macos

test:
	go test ./...
	$(MAKE) test-frontend

test-frontend:
	cd frontend && npm test

test-integration:
	go test -tags integration ./...

wails-dev:
	wails dev

wails-build:
	cd frontend && npm run build
	wails build

package-macos:
	scripts/package-macos.sh

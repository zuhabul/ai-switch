.PHONY: test test-v1 test-v2 build-v2

test: test-v1 test-v2

test-v1:
	./tests/managed_codex_rotation_test.sh
	./tests/managed_codex_rotation_matrix_test.sh
	./tests/seamless_proxy_rotation_test.sh
	./tests/status_ui_test.sh

test-v2:
	cd v2 && go test ./...

build-v2:
	cd v2 && go build ./cmd/aiswitch ./cmd/aiswitchd

.PHONY: test test-v1 test-v2 test-v2-e2e build-v2

test: test-v1 test-v2 test-v2-e2e

test-v1:
	./tests/managed_codex_rotation_test.sh
	./tests/managed_codex_rotation_matrix_test.sh
	./tests/seamless_proxy_rotation_test.sh
	./tests/status_ui_test.sh

test-v2:
	cd v2 && go test ./...

test-v2-e2e:
	./tests/v2_end_to_end_test.sh

build-v2:
	cd v2 && go build ./cmd/aiswitch ./cmd/aiswitchd

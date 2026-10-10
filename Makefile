BINARY_NAME := guardd

.PHONY: build test vet lint release docs-check print-go-directive

build:
	mkdir -p bin
	go build -o bin/$(BINARY_NAME) ./cmd/guardd
	# Note: client/ is a library package (no main); only cmd/guardd produces a binary.

test:
	go test ./...

vet:
	go vet ./...

lint:
	@if [ -f .golangci.yml ] || [ -f .golangci.yaml ]; then golangci-lint run; else echo "no golangci-lint config; skipping"; fi

release:
	goreleaser release --snapshot --clean

# docs-check replays the README's documented commands so doc drift (a README
# example that no longer matches the binary, or a stale Go version claim)
# fails at make time instead of sitting unnoticed (would have caught README-2).
# All no-key runs execute under env -i with HOME pointed at a temp dir, so
# guardd's .env key discovery (~/.hermes/.env, ~/9router-deploy/.env.shared)
# cannot leak a live key into the check — the run must fail closed on its own.
.PHONY: docs-check
docs-check: build
	@set -e; \
	GO_DIRECTIVE="$$($(MAKE) -s print-go-directive)"; \
	echo "docs-check: go directive = $$GO_DIRECTIVE"; \
	TMPHOME=$$(mktemp -d); \
	had_root_guardd=0; if [ -f ./guardd ]; then had_root_guardd=1; fi; \
	trap 'rm -rf "$$TMPHOME"; if [ "$$had_root_guardd" != 1 ]; then rm -f ./guardd; fi' EXIT; \
	git_dir="$$(git rev-parse --git-dir 2>/dev/null || true)"; \
	if [ -n "$$git_dir" ] && [ -f "$$git_dir/README.md" ]; then \
	  README_CHECK="$$git_dir/README.md"; \
	else \
	  README_CHECK="README.md"; \
	fi; \
	echo "docs-check: checking README at $$README_CHECK"; \
	GO_MINOR=$$(printf '%s' "$$GO_DIRECTIVE" | cut -d. -f1-2); \
	if grep -Fq -- "$$GO_DIRECTIVE" "$$README_CHECK"; then \
	  echo "docs-check: full go directive $$GO_DIRECTIVE found in README"; \
	else \
	  echo "docs-check NOTE: README does not carry the patch-precise go directive '$$GO_DIRECTIVE' (README-2 open) — enforcing minor-version match until it lands"; \
	  README_GO=$$(grep -E 'Go[^0-9]*[0-9]+\.[0-9]+' "$$README_CHECK" | grep -oE '[0-9]+\.[0-9]+' | sort -u); \
	  if ! printf '%s\n' "$$README_GO" | grep -Fxq -- "$$GO_MINOR"; then \
	    echo "docs-check FAIL: README's Go versions [$$README_GO] do not match go.mod's go $$GO_DIRECTIVE (stale Go version claim — README-2 class)"; \
	    exit 1; \
	  fi; \
	  echo "docs-check: go directive (minor $$GO_MINOR) OK"; \
	fi; \
	sed -n '/^```bash$$/,/^```$$/p' "$$README_CHECK" | \
	  sed '1d;$$d' | grep -F 'printf' | grep -F 'guardd' | head -1 > "$$TMPHOME/readme_cmd.sh"; \
	if [ ! -s "$$TMPHOME/readme_cmd.sh" ]; then \
	  echo "docs-check FAIL: no 'printf ... | guardd' invocation found in README.md bash fence"; \
	  exit 1; \
	fi; \
	sed -n '/^```bash$$/,/^```$$/p' "$$README_CHECK" | \
	  sed '1d;$$d' | grep -F 'go build' | head -1 > "$$TMPHOME/readme_build.sh"; \
	if [ ! -s "$$TMPHOME/readme_build.sh" ]; then \
	  echo "docs-check FAIL: no 'go build' line found in README.md bash fence"; \
	  exit 1; \
	fi; \
	echo "docs-check: replaying README build: $$(cat "$$TMPHOME/readme_build.sh")"; \
	/bin/sh -c "$$(cat "$$TMPHOME/readme_build.sh")" \
	  || { echo "docs-check FAIL: README's documented build command failed"; exit 1; }; \
	echo "docs-check: replaying README command: $$(cat "$$TMPHOME/readme_cmd.sh")"; \
	awk '/^```json$$/{f=1;next} f&&/^```$$/{exit} f' "$$README_CHECK" > "$$TMPHOME/readme_expected.json"; \
	if [ ! -s "$$TMPHOME/readme_expected.json" ]; then \
	  echo "docs-check FAIL: no json fence found in README.md"; \
	  exit 1; \
	fi; \
	OUT="$$TMPHOME/actual.json"; RC=0; \
	env -i HOME="$$TMPHOME" PATH="$$PATH" GUARD_EGRESS_ENABLED=true \
	  /bin/sh -c "$$(cat "$$TMPHOME/readme_cmd.sh")" > "$$OUT" 2>"$$TMPHOME/stderr.txt" || RC=$$?; \
	if [ "$$RC" -ne 2 ]; then \
	  echo "docs-check FAIL: README invocation expected rc=2 (fail-closed), got rc=$$RC"; \
	  cat "$$TMPHOME/stderr.txt"; \
	  exit 1; \
	fi; \
	if ! jq -S . "$$TMPHOME/readme_expected.json" > "$$TMPHOME/expected.sorted.json" 2>/dev/null; then \
	  sed -E 's/([0-9a-z"\}\]])[[:space:]]*\/\/[[:space:]].*/\1/' "$$TMPHOME/readme_expected.json" > "$$TMPHOME/readme_expected.stripped.json"; \
	  jq -S . "$$TMPHOME/readme_expected.stripped.json" > "$$TMPHOME/expected.sorted.json" 2>/dev/null \
	    || { echo "docs-check FAIL: could not parse README fail-closed JSON block"; exit 1; }; \
	fi; \
	jq -S . "$$OUT" > "$$TMPHOME/actual.sorted.json" \
	  || { echo "docs-check FAIL: guardd emitted invalid JSON"; cat "$$OUT"; exit 1; }; \
	if ! diff -u "$$TMPHOME/expected.sorted.json" "$$TMPHOME/actual.sorted.json"; then \
	  echo "docs-check FAIL: guardd output does not match README's fail-closed JSON (doc drift — README-2 class)"; \
	  exit 1; \
	fi; \
	echo "docs-check: fail-closed JSON matches README"; \
	printf '' | env -i HOME="$$TMPHOME" PATH="$$PATH" GUARD_EGRESS_ENABLED=true ./bin/$(BINARY_NAME) > "$$TMPHOME/empty.json" 2>"$$TMPHOME/empty.stderr" && RC=0 || RC=$$?; \
	if [ "$$RC" -ne 2 ]; then \
	  echo "docs-check FAIL: empty-stdin invocation expected rc=2 (fail-closed), got rc=$$RC"; \
	  cat "$$TMPHOME/empty.stderr"; \
	  exit 1; \
	fi; \
	jq -e '.decision == "block" and .errored == true and (.reason | contains("fail-closed"))' "$$TMPHOME/empty.json" > /dev/null \
	  || { echo "docs-check FAIL: empty-stdin run is not the fail-closed block verdict"; cat "$$TMPHOME/empty.json"; exit 1; }; \
	echo "docs-check: empty-stdin fail-closed rc=2 OK"; \
	echo "docs-check: PASS"
print-go-directive:
	@echo 1.26.6

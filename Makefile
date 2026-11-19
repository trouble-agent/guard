BINARY_NAME := guardd
VERSION ?= v0.2.0

.PHONY: build test vet lint release docs-check print-go-directive

build:
	mkdir -p bin
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/guardd
	# Note: client/ is a library package (no main); only cmd/guardd produces a binary.

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

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
	command -v jq >/dev/null 2>&1 \
	  || { echo "docs-check FAIL: jq was not found on PATH — this target shells out to jq (four times) to compare verdict JSON. Install jq and re-run (see README Prerequisites, GUARD-DF-004)."; exit 1; }; \
	GO_DIRECTIVE="$$( $(MAKE) -s print-go-directive )"; \
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
	echo "docs-check: checking connectors/README.md line anchors (DOC-16/DOC-40)"; \
	printf '%s\n' \
	  'main.go:146-147|flag.String("source"' \
	  'main.go:189|io.ReadAll(os.Stdin)' \
	  'main.go:196|WithTimeout(context.Background(), 120*time.Second)' \
	  'main.go:206-207|if res.Errored {' \
	  'main.go:241-251|handleHealth := func' \
	  'main.go:256-257|"unauthorized: missing or invalid "' \
	  'main.go:260|"POST only"' \
	  'main.go:269-270|if len(body) > maxBodyBytes {' \
	  'main.go:273-274|json.NewDecoder(bytes.NewReader(body))' \
	  'main.go:278|"content is required"' \
	  'main.go:281|WithTimeout(r.Context(), 120*time.Second)' \
	  'main.go:350-366|func resolveToken(' \
	  'main.go:352-353|case envVal != "":' \
	  'main.go:373-387|func readTokenFile(' \
	  'guard_client.py:46|DEFAULT_ENDPOINT = os.environ.get("GUARD_ENDPOINT"' \
	  'guard_client.py:208-211|CONSTRAINTS_FAIL_OPEN: Mapping[str, bool] = {' \
	  'guard_client.py:217-224|timeout: float = 120.0,' \
	  'guard_client.py:223|os.environ.get("GUARD_TOKEN"' \
	  'guard_client.py:262-263|except Exception as exc:' \
	  'guard_client.py:265-280|def _on_error(' \
	  'guard_client.py:311|"--endpoint",' \
	  'guard_client.py:323|"--timeout",' \
	  'guard_client.py:349|"--fail-open",' \
	  'guard_client.py:359-362|def _verdict_exit_code(' \
	  > "$$TMPHOME/anchor_table.txt"; \
	if [ ! -s "$$TMPHOME/anchor_table.txt" ]; then \
	  echo "docs-check FAIL: anchor table is empty — the check would run on nothing (DOC-16)"; \
	  exit 1; \
	fi; \
	TABLE_TOKENS=$$(cut -d'|' -f1 "$$TMPHOME/anchor_table.txt"); \
	README_ANCHORS=$$(grep -oE '(main\.go|guard_client\.py):[0-9]+(-[0-9]+)?' connectors/README.md | sort -u); \
	if [ -z "$$README_ANCHORS" ]; then \
	  echo "docs-check FAIL: no main.go:/guard_client.py: anchors found in connectors/README.md — check ran on nothing (DOC-16)"; \
	  exit 1; \
	fi; \
	anchor_fail=0; \
	for tok in $$README_ANCHORS; do \
	  if ! printf '%s\n' "$$TABLE_TOKENS" | grep -Fxq -- "$$tok"; then \
	    echo "docs-check FAIL: connectors/README.md cites $$tok but the anchor table carries no expected subject for it — add '$$tok|<substring the cited line must contain>' (DOC-40)"; \
	    anchor_fail=1; \
	  fi; \
	done; \
	for tok in $$TABLE_TOKENS; do \
	  if ! grep -Fq -- "$$tok" connectors/README.md; then \
	    echo "docs-check FAIL: anchor table entry $$tok is no longer cited in connectors/README.md — drop it or re-point the citation (DOC-40)"; \
	    anchor_fail=1; \
	  fi; \
	done; \
	while IFS='|' read -r anchor_tok anchor_subject; do \
	  [ -n "$$anchor_tok" ] || continue; \
	  anchor_file=$${anchor_tok%%:*}; \
	  anchor_range=$${anchor_tok#*:}; \
	  case "$$anchor_file" in \
	    main.go) anchor_path=cmd/guardd/main.go ;; \
	    guard_client.py) anchor_path=connectors/python/guard_client.py ;; \
	    *) echo "docs-check FAIL: anchor table entry $$anchor_tok names an ungated file (only main.go and guard_client.py carry subjects)"; anchor_fail=1; continue ;; \
	  esac; \
	  if [ ! -f "$$anchor_path" ]; then \
	    echo "docs-check FAIL: anchor $$anchor_tok -> $$anchor_path is missing"; \
	    anchor_fail=1; continue; \
	  fi; \
	  file_lines=$$(wc -l < "$$anchor_path"); \
	  anchor_start=$${anchor_range%%-*}; \
	  anchor_end=$${anchor_range#*-}; \
	  if [ "$$anchor_range" = "$$anchor_end" ]; then anchor_end=""; fi; \
	  anchor_line=$$(sed -n "$${anchor_start}p" "$$anchor_path"); \
	  anchor_start_text="$$anchor_line"; \
	  if [ -z "$$anchor_line" ]; then \
	    echo "docs-check FAIL: anchor $$anchor_tok -> $$anchor_path line $$anchor_start is EMPTY/absent ($$file_lines lines)"; \
	    anchor_fail=1; continue; \
	  fi; \
	  anchor_head=$$(printf '%s' "$$anchor_line" | sed 's/^[[:space:]]*//'); \
	  anchor_is_comment=0; \
	  case "$$anchor_file" in \
	    main.go) case "$$anchor_line" in *'//'*[![:space:]]*) anchor_is_comment=1 ;; esac ;; \
	    guard_client.py) case "$$anchor_head" in '#'*) anchor_is_comment=1 ;; esac ;; \
	  esac; \
	  if [ "$$anchor_is_comment" -ne 0 ]; then \
	    echo "docs-check FAIL: anchor $$anchor_tok -> $$anchor_path line $$anchor_start is a comment, not the cited behaviour: '$$anchor_line'"; \
	    anchor_fail=1; continue; \
	  fi; \
	  case "$$anchor_line" in \
	    ""|*[![:space:]]*) ;; \
	    *) \
	      echo "docs-check FAIL: anchor $$anchor_tok -> $$anchor_path line $$anchor_start is whitespace/blank, not the cited behaviour: '$$anchor_line'"; \
	      anchor_fail=1; continue;; \
	  esac; \
	  anchor_no_space=$$(printf '%s' "$$anchor_line" | tr -d '[:space:]'); \
	  case "$$anchor_line" in \
	    [[:space:]]*[{}][[:space:]]*) \
	      case "$$anchor_no_space" in \
	        '{'|'}') \
	          echo "docs-check FAIL: anchor $$anchor_tok -> $$anchor_path line $$anchor_start is a lone brace line, not the cited behaviour: '$$anchor_line'"; \
	          anchor_fail=1; continue;; \
	      esac;; \
	    '{'|'}') \
	      echo "docs-check FAIL: anchor $$anchor_tok -> $$anchor_path line $$anchor_start is a lone brace line, not the cited behaviour: '$$anchor_line'"; \
	      anchor_fail=1; continue;; \
	  esac; \
	  case "$$anchor_line" in \
	    *"$$anchor_subject"*) ;; \
	    *) \
	      echo "docs-check FAIL: anchor $$anchor_tok -> $$anchor_path line $$anchor_start does not carry the expected subject '$$anchor_subject' (line is: '$$anchor_line')"; \
	      anchor_fail=1; continue;; \
	  esac; \
	  if [ -n "$$anchor_end" ]; then \
	    if [ "$$anchor_start" -gt "$$anchor_end" ]; then \
	      echo "docs-check FAIL: anchor $$anchor_tok range goes backwards"; \
	      anchor_fail=1; continue; \
	    fi; \
	    if [ "$$anchor_end" -gt "$$file_lines" ]; then \
	      echo "docs-check FAIL: anchor $$anchor_tok range end $$anchor_end past EOF ($$file_lines lines)"; \
	      anchor_fail=1; continue; \
	    fi; \
	    anchor_line=$$(sed -n "$${anchor_end}p" "$$anchor_path"); \
	    if [ -z "$$anchor_line" ]; then \
	      echo "docs-check FAIL: anchor $$anchor_tok range end $$anchor_end is empty"; \
	      anchor_fail=1; continue; \
	    fi; \
	  fi; \
	  echo "docs-check: anchor $$anchor_tok ok: $$anchor_start: $$anchor_start_text$${anchor_end:+ ... $$anchor_end: $$(sed -n "$${anchor_end}p" "$$anchor_path")}"; \
	done < "$$TMPHOME/anchor_table.txt"; \
	if [ "$$anchor_fail" -ne 0 ]; then \
	  echo "docs-check FAIL: stale/incoherent connectors/README.md line anchors (DOC-16/DOC-40 — re-point the citation and update its table subject)"; \
	  exit 1; \
	fi; \
	echo "docs-check: PASS"
print-go-directive:
	@awk '/^go /{print $$2}' go.mod

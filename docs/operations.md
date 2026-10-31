# guardd operator monitoring and recovery

This runbook covers the HTTP service mode of `guardd`. `/check` sends submitted content to the hosted Jev classifier when egress is enabled; treat request bodies as potentially sensitive and configure the egress setting deliberately.

## 1. Safe startup

Build the service binary from the repository root:

```sh
make build
```

Use a dedicated operator token supplied through the environment, and bind explicitly to loopback:

```sh
GUARD_TOKEN='replace-with-a-long-random-secret' \
GUARD_EGRESS_ENABLED=false \
./bin/guardd -serve 127.0.0.1:8768
```

For an owner-only token file instead:

```sh
install -m 0600 /dev/null /run/guardd/operator-token
# Populate the file using your secret-management mechanism.
GUARD_EGRESS_ENABLED=false ./bin/guardd -serve 127.0.0.1:8768 \
  -tokenfile /run/guardd/operator-token
```

Do not pass a production secret with `-token`: command-line arguments can be visible through `ps` and `/proc/<pid>/cmdline`. The server refuses to start without a token.

## 2. Token precedence and file permissions

The effective token is selected in this order: non-empty `GUARD_TOKEN`, then `-tokenfile`, then `-token`. If no token is supplied, serving fails closed. The `-token` form is supported but discouraged because it exposes the secret in the process list.

A token file must have no group or other permission bits. Use mode `0400` or `0600`; modes such as `0640` are refused. For example:

```sh
chmod 0600 /run/guardd/operator-token
stat -c '%a %n' /run/guardd/operator-token
```

`GUARD_TOKEN` takes precedence over `-tokenfile` if both are set. The token is read at startup; change credentials by updating the environment/file and restarting the process.

## 3. Bind address and exposure

An address with an omitted host, such as `:8768`, is normalized to `127.0.0.1:8768`. This loopback default avoids exposing the service on every interface. An explicit host overrides it, so only use a non-loopback address when network exposure is intentional and protected by appropriate network controls.

Safe examples:

```sh
./bin/guardd -serve :8768                 # binds 127.0.0.1:8768
./bin/guardd -serve 127.0.0.1:8768        # explicit loopback
```

## 4. Liveness monitoring

Both `GET /healthz` and `GET /health` are unauthenticated liveness endpoints. They return HTTP 200 and JSON with `"status":"ok"`. The body also reports `egress_enabled`; inspect it to confirm the process-start configuration, but do not treat liveness as proof that a classifier call will succeed.

```sh
curl -sS -i http://127.0.0.1:8768/healthz
curl -sS -i http://127.0.0.1:8768/health
```

A successful liveness response establishes that the HTTP handler is responding. It does not establish that the external classifier is reachable or that `/check` has valid credentials.

## 5. `/check` authentication and status ladder

`/check` accepts `POST` and requires the configured token in `X-Operator-Token`. Authentication is checked before method or body validation, so an unauthenticated request receives 401 even if its method or JSON is otherwise invalid.

| Request condition | Expected response |
| --- | --- |
| Missing or invalid `X-Operator-Token` | `401 Unauthorized` |
| Valid token, method other than `POST` | `405 Method Not Allowed` |
| Valid token and `POST`, malformed JSON | `400 Bad Request` |
| Valid token and `POST`, empty or omitted `content` | `400 Bad Request` |
| Valid token and `POST`, non-empty `content` | `200 OK` with a JSON `Result` body |

Example:

```sh
curl -sS -i -X POST \
  -H 'X-Operator-Token: replace-with-a-long-random-secret' \
  -H 'Content-Type: application/json' \
  -d '{"content":"message to classify"}' \
  http://127.0.0.1:8768/check
```

HTTP 200 means the request was processed and a `Result` was returned; inspect its `errored`, `decision`, `route`, and `reason` fields. It does not by itself mean live classification succeeded.

## 6. Classifier egress behavior

`GUARD_EGRESS_ENABLED` defaults to `true` when unset. With egress enabled, content submitted for classification may be sent to the hosted Jev API outside this host. Set it to `false` to disable external classification; recognized false values also include `0`, `no`, and `off` (case-insensitive, with surrounding whitespace ignored). This setting is read at process start, so a change requires a restart.

When disabled, `guardd` has no local classifier. `/check` still returns HTTP 200 with an in-band errored `Result` explaining that egress is disabled; it does not make a live classifier call. With the default fail-closed policy the error result blocks/quarantines. Confirm the active mode from the startup warning and `egress_enabled` in the health response.

## 7. Policy startup behavior

Without a policy path, the built-in default policy is used. `GUARD_POLICY=/path/to/policy.json` selects a policy file; the `-policy /path/to/policy.json` flag takes precedence if both are provided. A named policy file that is missing, unreadable, invalid JSON, or has invalid score bands causes startup to fail. The process does not silently fall back to the default policy: correct the file/path before restarting.

```sh
GUARD_POLICY=/etc/guard/policy.json GUARD_TOKEN='…' \
  ./bin/guardd -serve 127.0.0.1:8768
```

## 8. Log interpretation

- `guardd: token loaded from GUARD_TOKEN` or `guardd: token loaded from tokenfile` identifies the selected credential source. A `-token` startup emits a warning that the secret is exposed through the process list.
- `WARNING: classifier egress is DISABLED; ...` confirms that classification is disabled and `/check` will return an in-band error result. The enabled warning explicitly states that submitted content can leave the host.
- `guardd listening on ...` is emitted only after the TCP bind succeeds. If the address is already in use, expect a `serve: listen tcp ...` failure and no listening banner; check for another process or choose a free port.
- Authentication refusals return 401 to the caller. The handler does not log each refusal in `main.go`; use an upstream access log, if configured, for request-level auditing. Do not infer that a refusal was logged by guardd itself.
- Classification outcomes are returned in each `/check` JSON `Result`, not as a documented per-request guardd log line. Inspect the response fields (`errored`, `decision`, `route`, `reason`, and policy metadata) when diagnosing an outcome.

## 9. Bounded recovery and restart

Use this short, bounded sequence; avoid restart loops:

1. Check `GET /healthz` once. If it returns 200, record `egress_enabled` and continue to the authenticated request check. If it fails, inspect the service manager's status and recent logs.
2. Confirm the process is configured with a token and that a token file has mode `0400` or `0600`. Verify the request header uses the same token source selected by precedence.
3. For `/check` 400/405 responses, correct the method or request body instead of restarting. For a 200 response with `errored:true`, read `reason`; with egress disabled, this is expected and restarting without changing configuration will not enable classification.
4. For a failed startup, inspect the first fatal error. Fix an invalid `GUARD_POLICY` file or bind conflict before another attempt. Do not repeatedly restart with unchanged inputs.
5. Restart once using the service manager or the original launch command. If managing the process directly, send `TERM`, wait up to 10 seconds, then use `KILL` only if it remains. Re-check the listening log line and both health endpoint and authenticated `/check` after restart.
6. If the process still fails, preserve its logs and configuration context (never copy the token itself) and escalate to the service owner. Do not broaden the bind address or disable fail-closed behavior as a troubleshooting shortcut.

## Source anchors

These line references are exact for the source revision this runbook was checked against; re-check them when `cmd/guardd/main.go` changes.

- `cmd/guardd/main.go:120-153` — command flags, egress default selection, policy resolution, token resolution, and serving-mode wiring.
- `cmd/guardd/main.go:207-247` — health endpoints, authentication/method/body checks, and `Result` response.
- `cmd/guardd/main.go:250-290` — token requirement, bind normalization, egress warnings, and post-bind listening log.
- `cmd/guardd/main.go:303-347` — token precedence and owner-only token-file permission validation.
- `cmd/guardd/main.go:375-395` and `cmd/guardd/main.go:418-470` — policy selection, parsing, and validation failure behavior.

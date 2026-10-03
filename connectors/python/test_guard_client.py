"""Offline tests for the Python connector (stub server, no network, no key)."""

import dataclasses
import io
import json
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

import pytest

import guard_client
from guard_client import Guard, GuardError, ROUTE_QUARANTINE, ROUTE_REVIEW, Verdict


class _Stub(BaseHTTPRequestHandler):
    response: dict = {}
    status = 200

    def do_POST(self):  # noqa: N802
        length = int(self.headers.get("Content-Length", 0))
        self.rfile.read(length)
        body = json.dumps(_Stub.response).encode()
        self.send_response(_Stub.status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):  # silence
        pass


class _TokenStub(BaseHTTPRequestHandler):
    """Mimics guardd's /check: 401 unless X-Operator-Token matches."""

    response: dict = {"route": "deliver", "decision": "allow"}
    token: str = ""

    def do_POST(self):  # noqa: N802
        length = int(self.headers.get("Content-Length", 0))
        self.rfile.read(length)
        if self.headers.get("X-Operator-Token") != _TokenStub.token:
            body = b"unauthorized: missing or invalid X-Operator-Token"
            self.send_response(401)
            self.send_header("Content-Type", "text/plain")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        body = json.dumps(_TokenStub.response).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):  # silence
        pass


@pytest.fixture()
def stub():
    srv = HTTPServer(("127.0.0.1", 0), _Stub)
    t = threading.Thread(target=srv.serve_forever, daemon=True)
    t.start()
    yield f"http://127.0.0.1:{srv.server_port}"
    srv.shutdown()


@pytest.fixture()
def token_stub(monkeypatch):
    monkeypatch.delenv("GUARD_TOKEN", raising=False)  # ambient env must not leak in
    srv = HTTPServer(("127.0.0.1", 0), _TokenStub)
    t = threading.Thread(target=srv.serve_forever, daemon=True)
    t.start()
    yield f"http://127.0.0.1:{srv.server_port}"
    srv.shutdown()


def test_parses_verdict(stub):
    _Stub.response = {
        "route": "quarantine",
        "decision": "block",
        "risk_level": "high",
        "score": 0.98,
        "constraints": {"allow_tools": False, "allow_secrets": False},
        "normalizations": ["rot13"],
        "load_bearing_normalizations": ["rot13"],
    }
    v = Guard(endpoint=stub).check("x")
    assert v.route == ROUTE_QUARANTINE
    assert not v.deliver
    assert not v.allows_tools
    assert v.normalizations == ["rot13"]
    assert v.load_bearing_normalizations == ["rot13"]


def test_parses_verdict_without_load_bearing(stub):
    # Old server / benign traffic: field absent -> empty list, no error.
    _Stub.response = {
        "route": "deliver",
        "decision": "allow",
        "risk_level": "low",
        "constraints": {"allow_tools": True},
        "normalizations": ["rot13"],
    }
    v = Guard(endpoint=stub).check("x")
    assert v.load_bearing_normalizations == []


def test_deliver_true_for_benign(stub):
    _Stub.response = {
        "route": "deliver",
        "decision": "allow",
        "risk_level": "low",
        "constraints": {"allow_tools": True},
    }
    assert Guard(endpoint=stub).is_safe("hello")


def test_fail_closed_raises():
    g = Guard(endpoint="http://127.0.0.1:1", timeout=1, fail_mode="closed")
    with pytest.raises(GuardError):
        g.check("x")


def test_fail_open_outage_preserves_capabilities():
    # GUARD-DF-001: an outage must not silently strip tools/secrets from every
    # task. Capabilities are PRESERVED and flagged (errored/decision='error').
    g = Guard(endpoint="http://127.0.0.1:1", timeout=1, fail_mode="open")
    v = g.check("x")
    assert v.allows_tools
    assert v.allows_secrets
    assert v.constraints == Guard.CONSTRAINTS_FAIL_OPEN


def test_fail_open_outage_verdict_shape():
    # The outage is distinguishable from a real block-all policy: route=review
    # (deliver False) + decision='error' + errored=True.
    g = Guard(endpoint="http://127.0.0.1:1", timeout=1, fail_mode="open")
    v = g.check("x")
    assert v.errored
    assert v.route == ROUTE_REVIEW
    assert not v.deliver
    assert v.decision == "error"
    assert v.risk_level == "medium"
    assert "guard_error" in v.reason
    assert v.raw.get("fail_mode") == "open"


def test_constraints_denial_still_applies_to_normal_verdicts(stub):
    # The strip pattern must stay correct for REAL policy verdicts: a
    # non-errored verdict that denies tools still yields allows_tools False.
    _Stub.response = {
        "route": "review",
        "decision": "block",
        "risk_level": "high",
        "constraints": {"allow_tools": False},
    }
    v = Guard(endpoint=stub).check("x")
    assert not v.errored
    assert not v.allows_tools
    assert not v.deliver


def test_rejects_bad_fail_mode():
    with pytest.raises(ValueError):
        Guard(fail_mode="whatever")


# --- token auth (REVIEW-GUARD-002): guardd's /check requires X-Operator-Token ---


def test_check_without_token_gets_401(token_stub):
    _TokenStub.token = "s3cret"
    g = Guard(endpoint=token_stub)
    with pytest.raises(GuardError) as ei:
        g.check("x")
    assert "401" in str(ei.value)


def test_check_with_token_arg_sends_header(token_stub):
    _TokenStub.token = "s3cret"
    v = Guard(endpoint=token_stub, token="s3cret").check("x")
    assert v.deliver


def test_check_with_token_env_fallback(token_stub, monkeypatch):
    monkeypatch.setenv("GUARD_TOKEN", "env-secret")
    _TokenStub.token = "env-secret"
    v = Guard(endpoint=token_stub).check("x")
    assert v.deliver


def test_token_arg_wins_over_env(token_stub, monkeypatch):
    monkeypatch.setenv("GUARD_TOKEN", "env-secret")
    _TokenStub.token = "arg-secret"
    v = Guard(endpoint=token_stub, token="arg-secret").check("x")
    assert v.deliver
    monkeypatch.setenv("GUARD_TOKEN", "arg-secret")
    _TokenStub.token = "arg-secret"
    g = Guard(endpoint=token_stub, token="")  # empty string falls back to env
    assert g.check("x").deliver


# --- CLI entry point (QA-GUARD-003): pip console script `guard` --------------


def test_module_exposes_main():
    # The console-script contract: `[project.scripts] guard = "guard_client:main"`.
    assert callable(guard_client.main)


def test_to_dict_roundtrip(stub):
    # The CLI prints to_dict(); from_dict(to_dict()) must preserve every
    # verdict field. `raw` is excluded: it is the ORIGINAL wire document and
    # to_dict deliberately spreads it flat instead of re-nesting it.
    _Stub.response = {
        "route": "quarantine",
        "decision": "block",
        "risk_level": "high",
        "score": 0.98,
        "constraints": {"allow_tools": False, "allow_secrets": False},
        "normalizations": ["rot13"],
        "load_bearing_normalizations": ["rot13"],
    }
    v = Guard(endpoint=stub).check("x")
    v2 = Verdict.from_dict(v.to_dict())
    for f in dataclasses.fields(Verdict):
        if f.name == "raw":
            continue
        assert getattr(v2, f.name) == getattr(v, f.name), f"field {f.name} diverged"


def test_to_dict_spreads_server_extra_fields(stub):
    # Unknown server fields survive to the CLI output (flat raw spread).
    _Stub.response = {"route": "deliver", "decision": "allow", "future_field": 7}
    v = Guard(endpoint=stub).check("x")
    assert v.to_dict()["future_field"] == 7


def test_main_deliver_exit_zero(stub, capsys, monkeypatch):
    _Stub.response = {"route": "deliver", "decision": "allow", "risk_level": "low"}
    monkeypatch.setattr("sys.stdin", io.StringIO(""))  # unused: content is argv
    rc = guard_client.main(["--endpoint", stub, "ignored payload"])
    assert rc == 0
    out = json.loads(capsys.readouterr().out)
    assert out["route"] == "deliver"
    assert out["decision"] == "allow"


def test_main_reads_stdin_when_no_content(stub, capsys, monkeypatch):
    # Endpoint via GUARD_ENDPOINT env (the console script's no-flag path).
    _Stub.response = {"route": "deliver", "decision": "allow", "risk_level": "low"}
    monkeypatch.setenv("GUARD_ENDPOINT", stub)
    monkeypatch.setattr("sys.stdin", io.StringIO("piped content"))
    rc = guard_client.main([])
    assert rc == 0
    out = json.loads(capsys.readouterr().out)
    assert out["route"] == "deliver"


def test_main_exit_codes_branch_by_route(stub, capsys):
    _Stub.response = {
        "route": "quarantine",
        "decision": "block",
        "risk_level": "high",
    }
    assert guard_client.main(["--endpoint", stub, "x"]) == 2
    assert json.loads(capsys.readouterr().out)["route"] == "quarantine"

    _Stub.response = {"route": "review", "decision": "flag", "risk_level": "medium"}
    assert guard_client.main(["--endpoint", stub, "x"]) == 1

    out = json.loads(capsys.readouterr().out)  # every code still prints the JSON
    assert out["route"] == "review"


def test_main_fail_closed_outage_exit_three(capsys):
    g_args = ["--endpoint", "http://127.0.0.1:1", "--timeout", "1", "x"]
    assert guard_client.main(g_args) == 3
    err = capsys.readouterr().err
    assert "guard check failed" in err
    assert capsys.readouterr().out == ""


def test_main_fail_open_outage_exit_one(capsys):
    g_args = [
        "--endpoint",
        "http://127.0.0.1:1",
        "--timeout",
        "1",
        "--fail-open",
        "x",
    ]
    assert guard_client.main(g_args) == 1
    out = json.loads(capsys.readouterr().out)
    assert out["errored"] is True
    assert out["route"] == "review"
    assert out["constraints"]["allow_tools"] is True


def test_main_empty_content_errors(capsys):
    with pytest.raises(SystemExit) as ei:
        guard_client.main([""])
    assert ei.value.code == 2
    assert "no content" in capsys.readouterr().err

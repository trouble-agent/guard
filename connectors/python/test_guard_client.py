"""Offline tests for the Python connector (stub server, no network, no key)."""

import json
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

import pytest

from guard_client import Guard, GuardError, ROUTE_QUARANTINE, ROUTE_REVIEW


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


@pytest.fixture()
def stub():
    srv = HTTPServer(("127.0.0.1", 0), _Stub)
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


def test_fail_open_returns_review_with_no_capabilities():
    g = Guard(endpoint="http://127.0.0.1:1", timeout=1, fail_mode="open")
    v = g.check("x")
    assert v.route == ROUTE_REVIEW
    assert v.errored
    assert not v.allows_tools
    assert not v.allows_secrets


def test_rejects_bad_fail_mode():
    with pytest.raises(ValueError):
        Guard(fail_mode="whatever")

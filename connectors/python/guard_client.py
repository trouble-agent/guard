"""guard_client — the Python connector for the guard service.

For task-router (Python) and anything else that is not Go. Stdlib only: one
import, one call, the way a parameterised query is one call.

    from guard_client import Guard

    g = Guard()                       # reads GUARD_ENDPOINT, else 127.0.0.1:8768
    v = g.check(body, source="task-router")
    if not v.deliver:
        # honour v.route and v.constraints — do not treat the body as trusted
        ...

The contract is the same Result shape as the Go package and the CLI, so a
verdict is portable across every surface.
"""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from typing import Any, Mapping, Optional

DEFAULT_ENDPOINT = os.environ.get("GUARD_ENDPOINT", "http://127.0.0.1:8768")

# Routes (stable identifiers — bind to these, not to the raw decision).
ROUTE_DELIVER = "deliver"
ROUTE_REVIEW = "review"
ROUTE_QUARANTINE = "quarantine"


@dataclass
class Verdict:
    """One guard outcome. Mirrors the Go guard.Result shape."""

    route: str = ROUTE_DELIVER
    decision: str = "allow"
    risk_level: str = "low"
    reason: str = ""
    attack_class: str = ""
    score: float = 0.0
    policy: str = ""
    provider: str = ""
    model: str = ""
    errored: bool = False
    constraints: Mapping[str, bool] = field(default_factory=dict)
    normalizations: list = field(default_factory=list)
    # Transforms whose decoded variant actually drove the verdict (absent/empty
    # otherwise). GUARD-DF-002: `normalizations` is what RAN (rot13 always);
    # this is what MATTERED.
    load_bearing_normalizations: list = field(default_factory=list)
    cost_usd: float = 0.0
    duration_ms: int = 0
    raw: Mapping[str, Any] = field(default_factory=dict)

    @property
    def deliver(self) -> bool:
        """True only when the message may go straight to the normal handler."""
        return self.route == ROUTE_DELIVER

    @property
    def allows_tools(self) -> bool:
        return bool(self.constraints.get("allow_tools"))

    @property
    def allows_secrets(self) -> bool:
        return bool(self.constraints.get("allow_secrets"))

    @classmethod
    def from_dict(cls, d: Mapping[str, Any]) -> "Verdict":
        return cls(
            route=d.get("route", ROUTE_DELIVER),
            decision=d.get("decision", "allow"),
            risk_level=d.get("risk_level", "low"),
            reason=d.get("reason", ""),
            attack_class=d.get("attack_class", ""),
            score=d.get("score", 0.0),
            policy=d.get("policy", ""),
            provider=d.get("provider", ""),
            model=d.get("model", ""),
            errored=bool(d.get("errored", False)),
            constraints=d.get("constraints", {}) or {},
            normalizations=list(d.get("normalizations", []) or []),
            load_bearing_normalizations=list(d.get("load_bearing_normalizations", []) or []),
            cost_usd=d.get("cost_usd", 0.0),
            duration_ms=d.get("duration_ms", 0),
            raw=d,
        )


class GuardError(RuntimeError):
    """Raised under fail_mode='closed' when the service cannot be reached."""


class Guard:
    """Client for a guardd instance.

    token:
        Shared secret sent as the X-Operator-Token header on /check (guardd's
        operator-token auth). When empty, GUARD_TOKEN from the environment is
        used; guardd answers 401 to /check without a valid token.

    fail_mode:
        'closed' (default) — a transport/HTTP error raises GuardError. The
            caller must not proceed; nothing is delivered on a failed check.
        'open' — a failed check returns a REVIEW verdict that PRESERVES
            capabilities and flags the outage: route='review',
            decision='error', errored=True, and
            constraints=CONSTRAINTS_FAIL_OPEN (allow_tools/allow_secrets
            still granted). An outage therefore degrades to "a human looks"
            without silently stripping tools/secrets from every task. An
            errored verdict ALWAYS gates unattended delivery — consumers must
            branch on `errored` and never auto-deliver, whatever constraints
            say.

    GUARD-DF-001: before this, an outage returned constraints={} — the same
    shape a real block-all policy produces — so the documented strip pattern
    (empty tools/secrets when a capability is not granted) crippled every task
    for the whole outage with no signal a consumer was likely to check.
    """

    # Granted on an errored (fail-open outage) verdict so the outage does not
    # silently cripple tasks. `errored=True` + route='review' still gate
    # delivery; these capabilities are explicitly flagged, not trusted.
    CONSTRAINTS_FAIL_OPEN: Mapping[str, bool] = {
        "allow_tools": True,
        "allow_secrets": True,
    }

    def __init__(
        self,
        endpoint: Optional[str] = None,
        token: Optional[str] = None,
        timeout: float = 120.0,
        fail_mode: str = "closed",
    ) -> None:
        if fail_mode not in ("closed", "open"):
            raise ValueError("fail_mode must be 'closed' or 'open'")
        self.endpoint = (endpoint or DEFAULT_ENDPOINT).rstrip("/")
        self.token = token if token else os.environ.get("GUARD_TOKEN", "")
        self.timeout = timeout
        self.fail_mode = fail_mode

    def check(
        self,
        content: str,
        source: str = "",
        channel: str = "",
        meta: Optional[Mapping[str, str]] = None,
        msg_id: str = "",
    ) -> Verdict:
        """Classify one message. Never returns a bare success on failure."""
        payload = {
            "id": msg_id,
            "source": source,
            "channel": channel,
            "content": content,
            "meta": dict(meta or {}),
        }
        try:
            headers = {"Content-Type": "application/json"}
            if self.token:
                headers["X-Operator-Token"] = self.token
            req = urllib.request.Request(
                self.endpoint + "/check",
                data=json.dumps(payload).encode("utf-8"),
                headers=headers,
                method="POST",
            )
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                return Verdict.from_dict(json.loads(resp.read().decode("utf-8")))
        except Exception as exc:  # transport, timeout, HTTP status, bad JSON
            return self._on_error(exc)

    def _on_error(self, exc: Exception) -> Verdict:
        if self.fail_mode == "open":
            # GUARD-DF-001: preserve capabilities and flag the outage. An
            # errored verdict always routes to REVIEW (deliver False) and
            # carries decision='error' so consumers can tell an outage from a
            # real block-all policy — `constraints` alone cannot.
            return Verdict(
                route=ROUTE_REVIEW,
                decision="error",
                risk_level="medium",
                reason=f"guard_error (connector fail-open): {exc}",
                errored=True,
                constraints=dict(self.CONSTRAINTS_FAIL_OPEN),
                raw={"error": str(exc), "fail_mode": "open"},
            )
        raise GuardError(f"guard check failed: {exc}") from exc

    # --- convenience: the one-liner for call sites that only need a boolean ---

    def is_safe(self, content: str, **kw: Any) -> bool:
        """True only when the message may be delivered unchanged."""
        return self.check(content, **kw).deliver

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
            cost_usd=d.get("cost_usd", 0.0),
            duration_ms=d.get("duration_ms", 0),
            raw=d,
        )


class GuardError(RuntimeError):
    """Raised under fail_mode='closed' when the service cannot be reached."""


class Guard:
    """Client for a guardd instance.

    fail_mode:
        'closed' (default) — a transport/HTTP error raises GuardError. The
            caller must not proceed; nothing is delivered on a failed check.
        'open' — a failed check returns a REVIEW verdict with every capability
            withheld, so an outage degrades to "a human looks", never to
            "no protection" or "work stops".
    """

    def __init__(
        self,
        endpoint: Optional[str] = None,
        timeout: float = 120.0,
        fail_mode: str = "closed",
    ) -> None:
        if fail_mode not in ("closed", "open"):
            raise ValueError("fail_mode must be 'closed' or 'open'")
        self.endpoint = (endpoint or DEFAULT_ENDPOINT).rstrip("/")
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
            req = urllib.request.Request(
                self.endpoint + "/check",
                data=json.dumps(payload).encode("utf-8"),
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                return Verdict.from_dict(json.loads(resp.read().decode("utf-8")))
        except Exception as exc:  # transport, timeout, HTTP status, bad JSON
            return self._on_error(exc)

    def _on_error(self, exc: Exception) -> Verdict:
        if self.fail_mode == "open":
            return Verdict(
                route=ROUTE_REVIEW,
                decision="allow",
                risk_level="medium",
                reason=f"guard_error (connector fail-open): {exc}",
                errored=True,
                constraints={},  # nothing granted
                raw={"error": str(exc)},
            )
        raise GuardError(f"guard check failed: {exc}") from exc

    # --- convenience: the one-liner for call sites that only need a boolean ---

    def is_safe(self, content: str, **kw: Any) -> bool:
        """True only when the message may be delivered unchanged."""
        return self.check(content, **kw).deliver

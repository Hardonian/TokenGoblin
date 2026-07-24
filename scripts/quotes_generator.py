#!/usr/bin/env python3

from __future__ import annotations

from dataclasses import dataclass

PACKAGES = [
    {
        "id": "starter",
        "name": "Starter",
        "price": 0,
        "period": "/ month",
        "limits": ["1 tenant", "10,000 events/mo", "CSV export"],
        "support": "Email support",
    },
    {
        "id": "pro",
        "name": "Pro",
        "price": 29,
        "period": "/ month",
        "limits": [
            "5 tenants",
            "100,000 events/mo",
            "Forecasting",
            "Routing recommendations",
        ],
        "support": "Priority support",
    },
    {
        "id": "enterprise",
        "name": "Enterprise",
        "price": 99,
        "period": "/ month",
        "limits": [
            "Unlimited tenants",
            "Unlimited events",
            "Audit trail + RBAC",
            "SLA",
        ],
        "support": "Dedicated support",
    },
]


@dataclass(frozen=True)
class Package:
    id: str
    name: str
    price: int
    period: str
    limits: list[str]
    support: str


def build_packages() -> list[Package]:
    return [Package(**item) for item in PACKAGES]


def quote_text(pkg: Package, currency: str = "USD") -> str:
    price_display = (
        "Free"
        if pkg.price == 0
        else f"{currency} {pkg.price}{pkg.period}"
    )
    lines = [f"[{pkg.name}] {price_display}"]
    lines.extend(f" - {item}" for item in pkg.limits)
    lines.append(f" - Support: {pkg.support}")
    return "\n".join(lines)


if __name__ == "__main__":
    for pkg in build_packages():
        print(quote_text(pkg))

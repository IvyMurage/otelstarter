"""otelstarter: business-aware domain events on top of OpenTelemetry.

from otelstarter import observability

observability.event("order.created", domain="orders", outcome="success")
"""

from otelstarter import observability

__all__ = ["observability"]
__version__ = "0.1.0"

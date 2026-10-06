# Quickstart: a Python app with traces, logs, metrics and business events

About 10 minutes, plus a one-time 1 GB download. At the end you can see your app's requests as traces, its logs linked to them, its metrics, and your own business events, in a local Grafana.

## You need

- **Go 1.25+**, to install otelstarter.
- **Docker** with Compose v2, for the local Collector and Grafana.
- **Python 3.10 or newer** for your app. If `python3 --version` says 3.9 or older, install a newer one, for example with [pyenv](https://github.com/pyenv/pyenv): `pyenv install 3.12`, then `pyenv local 3.12` in your app's folder.

## 1. Install otelstarter

```bash
go install github.com/IvyMurage/otelstarter/cmd/otelstarter@latest
```

Go puts the program in `~/go/bin`. If your shell then says `otelstarter: command not found`, add that folder to your `PATH`:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc    # or ~/.bashrc
source ~/.zshrc
otelstarter version
```

## 2. An app to instrument

Use your own FastAPI, Flask or Django app, or create this small one:

```bash
mkdir ~/otel-demo && cd ~/otel-demo
printf 'fastapi\nuvicorn\nhttpx\n' > requirements.txt
```

`main.py`:

```python
import logging
import httpx
from fastapi import FastAPI

log = logging.getLogger("shop")
app = FastAPI()

@app.post("/orders")
async def create_order():
    async with httpx.AsyncClient() as client:
        stock = await client.get("http://127.0.0.1:8000/inventory")
    log.info("order placed, stock=%s", stock.json()["available"])
    return {"order_id": "ord_1", "status": "created"}

@app.get("/inventory")
async def inventory():
    return {"available": 42}
```

## 3. Add OpenTelemetry

```bash
otelstarter analyze          # what otelstarter detects
otelstarter init --dry-run   # what it would change
otelstarter init             # answer y
```

`init` creates `otel.env` and adds pinned packages to `requirements.txt`. Install them in a virtual environment:

```bash
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
otelstarter validate
```

Expect one problem: *Nothing is listening at http://localhost:4318*. The next step starts the Collector. Fix anything else `validate` reports first.

## 4. Run it

```bash
otelstarter run -- uvicorn main:app
```

This starts the local stack, then your app with instrumentation. In a second terminal, send some traffic:

```bash
for i in $(seq 10); do curl -s -X POST localhost:8000/orders; echo; done
```

Open **http://localhost:3000**, then **Explore**:

| Pick | Query | You see |
|------|-------|---------|
| Tempo | Search, service `otel-demo` | Each `POST /orders`, with the httpx call and `GET /inventory` inside it |
| Loki | `{service_name="otel-demo"}` | The "order placed" logs, each with a `trace_id` |
| Prometheus | `http_server_duration_milliseconds_count` | Request counts per route. Metrics arrive about once a minute |

## 5. Record a business event

In `main.py`, import the SDK, which `init` already added:

```python
from otelstarter import observability
```

and inside `create_order`, before `return`:

```python
    observability.event("order.created", domain="orders", outcome="success",
                        entity_type="order", entity_id="ord_1")
```

Press Ctrl+C, run step 4 again, and send traffic. Each event now prints a line in the terminal:

```text
[otelstarter] event order.created domain=orders outcome=success entity_type=order entity_id=ord_1 trace_id=…
```

In Loki, `{service_name="otel-demo"} |= "order.created"` finds them. Copy an event's `trace_id` into Tempo to see the request it belongs to.

## 6. When something is wrong

- **`otelstarter validate`** checks the setup and says how to fix each problem.
- **`otelstarter doctor`** adds an environment report you can paste into an issue, with secrets hidden.
- **If the app crashes,** `run` keeps the stack up so you can look at what happened in Grafana. Press Ctrl+C when done.
- **A badly named event,** like `ordercreated`, logs a warning pointing at your line.

## Next

- [analyze](analyze.md), [init](init.md), [run](run.md), [validate and doctor](validate.md)
- [Python SDK](../sdk/python/README.md) and the [domain event model](DOMAIN_EVENTS.md)

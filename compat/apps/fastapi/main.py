"""The same four routes as compat/apps/express. Nothing here knows about otelstarter."""

import logging
import os

import psycopg2
from fastapi import FastAPI, Request

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("shop")
app = FastAPI()


@app.get("/health")
def health():
    return {"ok": True}


@app.get("/orders/{order_id}")
def order(order_id: int):
    # The email is a literal on purpose: otelstarter must remove it from the span.
    with psycopg2.connect(os.environ["DATABASE_URL"]) as conn, conn.cursor() as cur:
        cur.execute(f"SELECT {order_id} AS id, 'ann@shop.rw' AS email")
        row = cur.fetchone()
    return {"id": row[0], "email": row[1]}


@app.get("/boom")
def boom():
    raise RuntimeError("boom")


@app.post("/echo")
async def echo(request: Request):
    body = await request.json()
    log.info("echo received", extra={"password": body.get("password"), "n": body.get("n")})
    return body

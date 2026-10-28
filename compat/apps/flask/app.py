"""The same four routes as compat/apps/express. Nothing here knows about otelstarter."""

import logging
import os

import psycopg2
from flask import Flask, jsonify, request

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("shop")
app = Flask(__name__)


@app.get("/health")
def health():
    return jsonify(ok=True)


@app.get("/orders/<int:order_id>")
def order(order_id):
    # The email is a literal on purpose: otelstarter must remove it from the span.
    with psycopg2.connect(os.environ["DATABASE_URL"]) as conn, conn.cursor() as cur:
        cur.execute(f"SELECT {order_id} AS id, 'ann@shop.rw' AS email")
        row = cur.fetchone()
    return jsonify(id=row[0], email=row[1])


@app.get("/boom")
def boom():
    raise RuntimeError("boom")


@app.post("/echo")
def echo():
    body = request.get_json()
    log.info("echo received", extra={"password": body.get("password"), "n": body.get("n")})
    return jsonify(body)


if __name__ == "__main__":
    app.run(port=int(os.environ["PORT"]), use_reloader=False)

# Compatibility check

Does an app behave exactly the same with otelstarter as without it? `run.py` answers that for real apps, and CI runs it on every pull request.

| App | Framework | Runs as |
|-----|-----------|---------|
| `express` | Express 5, pg, pino | `node server.js` (CommonJS) |
| `express-esm` | the same, as ES modules | `node server.js` (`"type": "module"`) |
| `nestjs11` | NestJS 11, pg, pino | `node dist/main.js` (compiled to CommonJS) |
| `nestjs12` | NestJS 12, the latest | the same |
| `flask` | Flask 3, psycopg2 | `python app.py` |
| `flask-gunicorn` | the same | `gunicorn -w 2` |
| `fastapi` | FastAPI, psycopg2 | `uvicorn` |
| `fastapi-workers` | the same | `uvicorn --workers 2` |

Every app has the same routes: `/health`, `/orders/42` (a database query with an email address in the SQL), `/boom` (throws), an unknown page, and `POST /echo`, which logs a password with the app's own logger.

## What it checks

Each app is set up the way a user would: dependencies installed, `otelstarter init --yes`, then the packages init asks for (the SDKs come from this checkout). Then it runs three times:

1. **Without otelstarter:** the reference answers.
2. **With otelstarter and a Collector:**
   - every answer, including the 500 and the 404, is the same as without otelstarter;
   - SIGTERM stops the app within 10 s;
   - the Collector received traces naming the route, the app's log line, and metrics;
   - neither the email address nor the password reached the Collector.
3. **With otelstarter and the Collector down:** the same answers, and SIGTERM still stops the app within 10 s.

The Collector is a small OTLP/HTTP receiver inside `run.py`, so no Docker is needed apart from Postgres.

It also prints what otelstarter costs per app: startup time, memory of all the app's processes, and the median time of 100 requests.

## Running it

Needs Go, Node.js 22.14+, Python 3.10+ and a Postgres:

```bash
docker run -d --rm --name compat-pg -e POSTGRES_PASSWORD=pw -p 127.0.0.1:5432:5432 postgres:16-alpine
DATABASE_URL=postgres://postgres:pw@127.0.0.1:5432/postgres python3 compat/run.py            # all apps
DATABASE_URL=postgres://postgres:pw@127.0.0.1:5432/postgres python3 compat/run.py flask      # one app
```

Apps are set up in `compat/.work/`. Delete it to start from scratch.

## Adding an app

Add a folder under `apps/` with the same routes, and an entry in `APPS` in `run.py`. Pin exact dependency versions, so the result only changes when a version is changed on purpose.

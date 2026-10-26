// The same four routes in every compat app. Nothing here knows about otelstarter.
const express = require('express');
const { Pool } = require('pg');
const log = require('pino')();

const pool = new Pool({ connectionString: process.env.DATABASE_URL });
const app = express();
app.use(express.json());

app.get('/health', (req, res) => res.json({ ok: true }));
app.get('/orders/:id', async (req, res) => {
  // The email is a literal on purpose: otelstarter must remove it from the span.
  const { rows } = await pool.query(`SELECT ${Number(req.params.id)} AS id, 'ann@shop.rw' AS email`);
  res.json(rows[0]);
});
app.get('/boom', () => {
  throw new Error('boom');
});
app.post('/echo', (req, res) => {
  log.info({ password: req.body.password, n: req.body.n }, 'echo received');
  res.json(req.body);
});

app.listen(Number(process.env.PORT), () => log.info('listening'));

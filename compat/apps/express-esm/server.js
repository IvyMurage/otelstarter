// express/server.js, written as an ES module.
import express from 'express';
import pg from 'pg';
import pino from 'pino';

const log = pino();
const pool = new pg.Pool({ connectionString: process.env.DATABASE_URL });
const app = express();
app.use(express.json());

app.get('/health', (req, res) => res.json({ ok: true }));
app.get('/orders/:id', async (req, res) => {
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

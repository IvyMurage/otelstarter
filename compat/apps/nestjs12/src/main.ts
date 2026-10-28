// The same four routes as compat/apps/express, as a NestJS app compiled to
// CommonJS (the Nest default).
import 'reflect-metadata';
import { Body, Controller, Get, Module, Param, Post } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import { Pool } from 'pg';
import pino from 'pino';

const log = pino();
const pool = new Pool({ connectionString: process.env.DATABASE_URL });

@Controller()
class AppController {
  @Get('health')
  health() {
    return { ok: true };
  }

  @Get('orders/:id')
  async order(@Param('id') id: string) {
    const { rows } = await pool.query(`SELECT ${Number(id)} AS id, 'ann@shop.rw' AS email`);
    return rows[0];
  }

  @Get('boom')
  boom(): never {
    throw new Error('boom');
  }

  @Post('echo')
  echo(@Body() body: { password?: string; n?: number }) {
    log.info({ password: body.password, n: body.n }, 'echo received');
    return body;
  }
}

@Module({ controllers: [AppController] })
class AppModule {}

async function main() {
  const app = await NestFactory.create(AppModule, { logger: false });
  await app.listen(Number(process.env.PORT));
  log.info('listening');
}
void main();

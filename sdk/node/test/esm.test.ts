import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { isESMEntry } from '../src/esm';

function project(type: string | undefined): string {
  const dir = mkdtempSync(path.join(os.tmpdir(), 'otelstarter-esm-'));
  writeFileSync(path.join(dir, 'package.json'), JSON.stringify(type ? { type } : {}));
  mkdirSync(path.join(dir, 'src'));
  return dir;
}

test('isESMEntry follows Node’s own rules', () => {
  const esm = project('module');
  const cjs = project(undefined);
  assert.equal(isESMEntry(path.join(esm, 'src', 'server.js')), true, 'type: module, found one folder up');
  assert.equal(isESMEntry(path.join(esm, 'server.cjs')), false, '.cjs is always CommonJS');
  assert.equal(isESMEntry(path.join(cjs, 'server.js')), false, 'no type: CommonJS');
  assert.equal(isESMEntry(path.join(cjs, 'server.mjs')), true, '.mjs is always an ES module');
  assert.equal(isESMEntry(path.join(cjs, 'main.ts')), false, 'other extensions: left to their own loader');
  assert.equal(isESMEntry(undefined), false, 'node -e');
});

// A fake library under node_modules, patched by a tiny instrumentation
// through the same import-in-the-middle hook the SDK registers.
test('modules loaded with import are patched in an ES-module app', { skip: !('isInternalThread' in require('node:worker_threads')) }, async () => {
  const dir = project('module');
  const lib = path.join(dir, 'node_modules', 'greeter');
  mkdirSync(lib, { recursive: true });
  writeFileSync(path.join(lib, 'package.json'), JSON.stringify({ name: 'greeter', version: '1.0.0', main: 'index.js' }));
  writeFileSync(path.join(lib, 'index.js'), "exports.greet = () => 'hello';");
  writeFileSync(path.join(dir, 'app.js'), "import { greet } from 'greeter'; console.log(greet());");

  const esm = path.join(__dirname, '..', 'src', 'esm.js');
  const preload = path.join(dir, 'preload.cjs');
  writeFileSync(preload, `
    if (!require('node:worker_threads').isInternalThread) {
      const { InstrumentationBase, InstrumentationNodeModuleDefinition } = require(${JSON.stringify(require.resolve('@opentelemetry/instrumentation'))});
      class Greeter extends InstrumentationBase {
        constructor() { super('greeter-test', '1.0.0', {}); }
        init() {
          return [new InstrumentationNodeModuleDefinition('greeter', ['*'], (m) => {
            this._wrap(m, 'greet', () => () => 'patched');
            return m;
          })];
        }
      }
      const g = new Greeter();
      require(${JSON.stringify(esm)}).hookImports(['greeter']);
      g.enable();
    }`);

  const child = spawn(process.execPath, ['--require', preload, path.join(dir, 'app.js')], { cwd: dir });
  let out = '';
  child.stdout.on('data', (d) => (out += d));
  child.stderr.on('data', (d) => (out += d));
  await once(child, 'exit');
  assert.equal(out.trim(), 'patched', out);
});

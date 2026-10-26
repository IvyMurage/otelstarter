import { metrics } from '@opentelemetry/api';
import { logs } from '@opentelemetry/api-logs';
import { OTLPLogExporter } from '@opentelemetry/exporter-logs-otlp-proto';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-proto';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-proto';
import { registerInstrumentations, type Instrumentation } from '@opentelemetry/instrumentation';
import {
  defaultResource, detectResources, envDetector, hostDetector, osDetector, processDetector,
} from '@opentelemetry/resources';
import { BatchLogRecordProcessor, LoggerProvider } from '@opentelemetry/sdk-logs';
import { MeterProvider, PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { BatchSpanProcessor, type SpanProcessor } from '@opentelemetry/sdk-trace-base';
import { NodeTracerProvider } from '@opentelemetry/sdk-trace-node';
import { listFromEnv } from './config';
import { hookImports, moduleNames } from './esm';
import { PrivacyLogRecordProcessor, PrivacySpanProcessor } from './processor';

/**
 * Starts OpenTelemetry with only what the app needs: the three OTLP exporters
 * (traces, metrics, logs over HTTP/protobuf) and the instrumentations named in
 * OTEL_NODE_ENABLED_INSTRUMENTATIONS. Everything else is configured by the
 * standard OTEL_* environment variables in otel.env.
 */

/** Always loaded: incoming/outgoing HTTP, fetch, runtime metrics, pino logs. */
export const BASE_INSTRUMENTATIONS = ['http', 'undici', 'runtime-node', 'pino'];

export interface Started {
  instrumentations: string[];
  missing: string[];
  shutdown(): Promise<void>;
}

export function start(env: NodeJS.ProcessEnv = process.env): Started {
  if (env.OTEL_SDK_DISABLED === 'true') {
    return { instrumentations: [], missing: [], shutdown: () => Promise.resolve() };
  }

  // No cloud detectors: probing cloud metadata servers outside a cloud only
  // slows startup and prints a warning.
  const resource = defaultResource().merge(
    detectResources({ detectors: [envDetector, hostDetector, osDetector, processDetector] }),
  );

  const spanProcessors: SpanProcessor[] = exporterEnabled('OTEL_TRACES_EXPORTER', env)
    ? [new PrivacySpanProcessor(new BatchSpanProcessor(new OTLPTraceExporter()))] // clean, then export
    : [];
  const tracerProvider = new NodeTracerProvider({ resource, spanProcessors });
  tracerProvider.register();

  const meterProvider = new MeterProvider({
    resource,
    readers: exporterEnabled('OTEL_METRICS_EXPORTER', env)
      ? [new PeriodicExportingMetricReader({
          exporter: new OTLPMetricExporter(),
          // The reader does not read this variable itself; 60 s is the standard default.
          exportIntervalMillis: positiveInt(env.OTEL_METRIC_EXPORT_INTERVAL) ?? 60_000,
        })]
      : [],
  });
  metrics.setGlobalMeterProvider(meterProvider);

  const loggerProvider = new LoggerProvider({
    resource,
    processors: exporterEnabled('OTEL_LOGS_EXPORTER', env)
      ? [new PrivacyLogRecordProcessor(new BatchLogRecordProcessor({ exporter: new OTLPLogExporter() }))] // clean, then export
      : [],
  });
  logs.setGlobalLoggerProvider(loggerProvider);

  const wanted = listFromEnv('OTEL_NODE_ENABLED_INSTRUMENTATIONS', env) ?? BASE_INSTRUMENTATIONS;
  const names = [...new Set([...BASE_INSTRUMENTATIONS, ...wanted])];
  const { loaded, missing } = loadInstrumentations(names);
  const instances = loaded.map((l) => l.instance);
  // Before the app's first import, so modules loaded with import are patched too.
  hookImports(moduleNames(instances), env);
  registerInstrumentations({ instrumentations: instances, tracerProvider, meterProvider, loggerProvider });

  let stopped: Promise<void> | undefined;
  return {
    instrumentations: loaded.map((l) => l.name),
    missing,
    shutdown: () =>
      (stopped ??= Promise.allSettled([tracerProvider.shutdown(), meterProvider.shutdown(), loggerProvider.shutdown()])
        .then(() => undefined)),
  };
}

function positiveInt(value: string | undefined): number | undefined {
  const n = Number(value);
  return Number.isInteger(n) && n > 0 ? n : undefined;
}

function exporterEnabled(variable: string, env: NodeJS.ProcessEnv): boolean {
  return (env[variable] ?? 'otlp').toLowerCase() !== 'none';
}

/**
 * Loads @opentelemetry/instrumentation-<name> for each name. A package that is
 * not installed is reported, not fatal: observability must never stop the app.
 */
export function loadInstrumentations(names: string[]): { loaded: { name: string; instance: Instrumentation }[]; missing: string[] } {
  const loaded: { name: string; instance: Instrumentation }[] = [];
  const missing: string[] = [];
  for (const name of names) {
    let mod: Record<string, unknown>;
    try {
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      mod = require(`@opentelemetry/instrumentation-${name}`) as Record<string, unknown>;
    } catch {
      missing.push(name);
      continue;
    }
    // Each instrumentation package exports one class ending in "Instrumentation".
    const Class = Object.entries(mod).find(([key, value]) => key.endsWith('Instrumentation') && typeof value === 'function')?.[1] as
      | (new (config?: object) => Instrumentation)
      | undefined;
    if (!Class) {
      missing.push(name);
      continue;
    }
    // pino: trace and span IDs come from otelstarter's own logger fields.
    const config = name === 'pino' ? { disableLogCorrelation: true } : {};
    loaded.push({ name, instance: new Class(config) });
  }
  return { loaded, missing };
}

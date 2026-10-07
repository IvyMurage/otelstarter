/** Settings read from the standard OpenTelemetry environment variables (otel.env). */

export function serviceName(env: NodeJS.ProcessEnv = process.env): string {
  return env.OTEL_SERVICE_NAME || 'unknown_service';
}

/** deployment.environment.name from OTEL_RESOURCE_ATTRIBUTES, or "". */
export function environmentName(env: NodeJS.ProcessEnv = process.env): string {
  for (const pair of (env.OTEL_RESOURCE_ATTRIBUTES ?? '').split(',')) {
    const [key, value] = pair.split('=');
    if (key?.trim() === 'deployment.environment.name' || key?.trim() === 'deployment.environment') {
      return (value ?? '').trim();
    }
  }
  return '';
}

/** Comma-separated list from an environment variable, or undefined when unset. */
export function listFromEnv(name: string, env: NodeJS.ProcessEnv = process.env): string[] | undefined {
  const raw = env[name];
  if (raw === undefined || raw.trim() === '') return undefined;
  return raw.split(',').map((s) => s.trim()).filter(Boolean);
}

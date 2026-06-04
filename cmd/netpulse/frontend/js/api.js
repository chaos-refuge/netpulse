// ===== NetPulse API Layer =====

const API = {
  STATUS:  '/api/status',
  PING:    '/api/detect/ping',
  DNS:     '/api/detect/dns',
  TRACE:   '/api/detect/trace',
  DIAGNOSE:'/api/diagnose',
  DNS_SET: '/api/optimize/dns',
  FLUSH:   '/api/optimize/flush',
  PROXY:   '/api/optimize/proxy',
  PRESETS: '/api/optimize/presets',
};

async function api(url, body) {
  const opts = { method: body ? 'POST' : 'GET', headers: {} };
  if (body) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const r = await fetch(url, opts);
  return r.json();
}

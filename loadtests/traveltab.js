import http from 'k6/http';
import { check, sleep } from 'k6';

const profile = __ENV.PROFILE || 'smoke';
const baseURL = (__ENV.BASE_URL || 'http://127.0.0.1:8081').replace(/\/$/, '');
const cities = ['Lisbon', 'Porto', 'Faro'];
const profiles = {
  smoke: { executor: 'shared-iterations', vus: 1, iterations: 3, maxDuration: '30s' },
  load: { executor: 'ramping-vus', startVUs: 0, stages: [
    { duration: '30s', target: 10 }, { duration: '2m', target: 10 }, { duration: '30s', target: 0 },
  ] },
  stress: { executor: 'ramping-vus', startVUs: 0, stages: [
    { duration: '30s', target: 10 }, { duration: '1m', target: 10 },
    { duration: '30s', target: 25 }, { duration: '1m', target: 25 },
    { duration: '30s', target: 50 }, { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ] },
};
if (!Object.prototype.hasOwnProperty.call(profiles, profile)) {
  throw new Error('PROFILE must be smoke, load, or stress');
}
if (!/^https?:\/\//.test(baseURL)) throw new Error('BASE_URL must start with http:// or https://');

export const options = {
  scenarios: { journey: { ...profiles[profile], gracefulStop: '10s' } },
  userAgent: 'TravelTab-loadtest/1.0', // Exercise real visit tracking (not a bot UA).
  maxRedirects: 0,
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: [{ threshold: 'rate<0.01', abortOnFail: true, delayAbortEval: '10s' }],
    'http_req_duration{endpoint:home}': ['p(95)<500'],
    'http_req_duration{endpoint:search}': ['p(95)<500'],
    'http_req_duration{endpoint:itinerary}': ['p(95)<1000'],
  },
};

function requestParams(endpoint, fragment = false) {
  return {
    timeout: '5s',
    tags: { endpoint, name: endpoint },
    headers: fragment ? { 'HX-Request': 'true' } : {},
  };
}

function valid(response, marker, endpoint) {
  return check(response, {
    [`${endpoint}: HTTP 200`]: (r) => r.status === 200,
    [`${endpoint}: expected content`]: (r) => typeof r.body === 'string' && r.body.includes(marker),
  });
}

export function setup() {
  // Warm templates sequentially before concurrent users render for the first time.
  const response = http.get(`${baseURL}/`, requestParams('warmup'));
  if (!valid(response, 'TravelTab', 'warmup')) throw new Error('Home page preflight failed');
  for (const city of cities) {
    const result = http.get(`${baseURL}/process-form/?city_name=${encodeURIComponent(city)}`, requestParams('warmup', true));
    if (!valid(result, 'weather-display-container', 'warmup')) throw new Error(`Search preflight failed: ${city}`);
  }
}

export default function () {
  const city = cities[(__VU + __ITER - 1) % cities.length];
  const home = http.get(`${baseURL}/`, requestParams('home'));
  valid(home, 'TravelTab', 'home');
  const search = http.get(`${baseURL}/process-form/?city_name=${encodeURIComponent(city)}`, requestParams('search', true));
  valid(search, 'weather-display-container', 'search');
  const itinerary = http.post(`${baseURL}/generate-itinerary`, {
    city_itenary: city,
    start_date: '2026-10-01',
    end_date: '2026-10-03',
    categories: 'tourism.sights',
  }, requestParams('itinerary', true));
  valid(itinerary, 'itinerary-map', 'itinerary');
  sleep(1);
}

export function handleSummary(data) {
  const output = __ENV.SUMMARY_PATH || `loadtests/results/${profile}-summary.json`;
  const lines = [`TravelTab ${profile}: ${baseURL}`];
  for (const [name, metric] of Object.entries(data.metrics)) {
    if (metric.thresholds) {
      for (const [threshold, result] of Object.entries(metric.thresholds)) {
        lines.push(`${result.ok ? 'PASS' : 'FAIL'} ${name} ${threshold}`);
      }
    }
  }
  lines.push(`Full metrics: ${output}`);
  return {
    stdout: `${lines.join('\n')}\n`,
    [output]: JSON.stringify({ profile, baseURL, recordedAt: new Date().toISOString(), ...data }, null, 2),
  };
}

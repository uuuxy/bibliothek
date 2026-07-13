import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 11,
  duration: '10s',
};

// Zugangsdaten ausschließlich über k6-Umgebungsvariablen (z.B. k6 run -e TEST_PASSWORD=... loadtest.js)
const TEST_EMAIL = __ENV.TEST_EMAIL || 'pflasch@philipp-reis-schule.de';
const TEST_PASSWORD = __ENV.TEST_PASSWORD;

// Setup wird einmalig vor allen VUs ausgeführt
export function setup() {
  if (!TEST_PASSWORD) {
    throw new Error('TEST_PASSWORD environment variable is required (k6 run -e TEST_PASSWORD=...)');
  }
  const loginRes = http.post('http://localhost:8084/login', JSON.stringify({
    email: TEST_EMAIL,
    password: TEST_PASSWORD
  }), {
    headers: { 'Content-Type': 'application/json' }
  });

  if (loginRes.status !== 200) {
    console.error(`Login failed: ${loginRes.status} ${loginRes.body}`);
  }

  // k6 speichert Cookies im Cookie-Jar, wir können sie auch manuell extrahieren
  let cookies = {};
  if (loginRes.cookies) {
    for (const name of Object.keys(loginRes.cookies)) {
      cookies[name] = loginRes.cookies[name][0].value;
    }
  }

  return { cookies };
}

export default function bulkReceiveTest(data) {
  const url = 'http://localhost:8084/api/bestellungen/bulk-receive'; 
  
  const payload = JSON.stringify({
    exemplar_ids: ['1001', '1002', '1003', '1004', '1005'] 
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': data.cookies['csrf_token'] || '',
    },
    cookies: data.cookies,
  };

  const res = http.post(url, payload, params);

  check(res, {
    'ist Status 200 (Erfolg)': (r) => r.status === 200,
    'Server nicht abgestürzt (kein 500)': (r) => r.status !== 500,
  });
}

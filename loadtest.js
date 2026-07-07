import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 11,
  duration: '10s',
};

// Setup wird einmalig vor allen VUs ausgeführt
export function setup() {
  const loginRes = http.post('http://localhost:8084/login', JSON.stringify({
    email: 'pflasch@philipp-reis-schule.de',
    password: 'peterfxy23'
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

export default function (data) {
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

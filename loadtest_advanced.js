import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    ausleihe_opac: {
      executor: 'constant-vus',
      vus: 5,
      duration: '6h',
      exec: 'ausleiheAndOpac',
    },
    backoffice: {
      executor: 'constant-vus',
      vus: 2,
      duration: '6h',
      exec: 'backofficeTasks',
    },
    inventur: {
      executor: 'constant-vus',
      vus: 2,
      duration: '6h',
      exec: 'inventurTasks',
    },
    buchhalter: {
      executor: 'constant-vus',
      vus: 1,
      duration: '6h',
      exec: 'buchhalterTasks',
    },
  },
};

const TEST_EMAIL = __ENV.TEST_EMAIL || 'pflasch@philipp-reis-schule.de';
const TEST_PASSWORD = __ENV.TEST_PASSWORD;
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8084';

const SUPPLIER_ID = 'dbf8042f-7ae6-493b-8a0b-e6c58547e877';
const TITEL_ID = 'dedc8a2f-f79e-46d6-aa81-923508cfdbaa';

export function setup() {
  if (!TEST_PASSWORD) {
    throw new Error('TEST_PASSWORD required');
  }

  const loginRes = http.post(`${BASE_URL}/login`, JSON.stringify({
    email: TEST_EMAIL, password: TEST_PASSWORD
  }), { headers: { 'Content-Type': 'application/json' } });

  if (loginRes.status !== 200) throw new Error(`Login failed: ${loginRes.status}`);

  let cookies = {};
  for (const name of Object.keys(loginRes.cookies || {})) {
    cookies[name] = loginRes.cookies[name][0].value;
  }

  const csrfRes = http.get(`${BASE_URL}/api/csrf-token`, {
    headers: { 'Cookie': Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join('; ') }
  });
  if (csrfRes.status === 200) {
    cookies['csrf_token'] = JSON.parse(csrfRes.body).csrf_token || cookies['csrf_token'];
  }
  
  const schuelerRes = http.get(`${BASE_URL}/api/schueler?q=LTS-`, {
    headers: { 'Cookie': Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join('; ') }
  });
  
  let students = JSON.parse(schuelerRes.body || '[]').filter(s => s.barcode_id.startsWith('LTS-'));
  if (students.length === 0) students = JSON.parse(schuelerRes.body || '[]');

  return { cookies, csrfToken: cookies['csrf_token'] || '', students };
}

function getReqHeaders(data) {
  return {
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': data.csrfToken,
      'Cookie': Object.entries(data.cookies).map(([k, v]) => `${k}=${v}`).join('; ')
    }
  };
}

// ------------------------------------------------------------------
// SCENARIO A: Ausleihe & OPAC (Hoher Durchsatz)
// ------------------------------------------------------------------
export function ausleiheAndOpac(data) {
  if (!data.students.length) return;
  const reqConf = getReqHeaders(data);
  const student = data.students[Math.floor(Math.random() * data.students.length)];

  // 1. OPAC Search
  const opacRes = http.get(`${BASE_URL}/api/public/opac/suche?q=Loadtest`, reqConf);
  check(opacRes, { 'OPAC ok': (r) => r.status === 200 });

  // 2. Global Admin Search
  const searchRes = http.get(`${BASE_URL}/api/search?q=Loadtest`, reqConf);
  check(searchRes, { 'Search ok': (r) => r.status === 200 });

  // 3. Ausleihen
  const randomBarcode = `LTB-${Math.floor(Math.random() * 5000) + 1}`;
  const actionRes = http.post(`${BASE_URL}/api/action`, JSON.stringify({
    query: randomBarcode,
    active_student_id: student.id
  }), reqConf);
  
  check(actionRes, { 'Ausleihe ok': (r) => r.status !== 500 });
  
  if (actionRes.status === 200) {
    sleep(0.5);
    // 4. Rückgabe
    const rueckgabeRes = http.post(`${BASE_URL}/api/action`, JSON.stringify({
      query: randomBarcode
    }), reqConf);
    check(rueckgabeRes, { 'Rückgabe ok': (r) => r.status !== 500 });
  }

  sleep(Math.random() * 2);
}

// ------------------------------------------------------------------
// SCENARIO B: Backoffice (Stammdaten & Bestellungen)
// ------------------------------------------------------------------
export function backofficeTasks(data) {
  if (!data.students.length) return;
  const reqConf = getReqHeaders(data);
  const student = data.students[Math.floor(Math.random() * data.students.length)];

  // 1. Adresse patchen
  const patchRes = http.patch(`${BASE_URL}/api/schueler/${student.id}`, JSON.stringify({
    vorname: student.vorname,
    nachname: student.nachname,
    klasse: student.klasse,
    abgaenger_jahr: student.abgaenger_jahr,
    strasse: `Teststraße ${Math.floor(Math.random() * 100)}`,
    plz: "12345",
    ort: "Teststadt",
    bemerkung: `Zuletzt bearbeitet von VU ${__VU} um ${new Date().toISOString()}`
  }), reqConf);
  check(patchRes, { 'Patch ok': (r) => r.status === 200 || r.status === 409 });

  // 2. Schüler manuell sperren
  const lockRes = http.patch(`${BASE_URL}/api/admin/students/${student.id}/lock`, JSON.stringify({
    is_locked: true,
    reason: "Loadtest Sperre"
  }), reqConf);
  check(lockRes, { 'Lock ok': (r) => r.status === 200 || r.status === 409 });

  sleep(1);

  // 3. Schüler entsperren
  const unlockRes = http.patch(`${BASE_URL}/api/admin/students/${student.id}/lock`, JSON.stringify({
    is_locked: false,
    reason: ""
  }), reqConf);
  check(unlockRes, { 'Unlock ok': (r) => r.status === 200 || r.status === 409 });

  // 4. Bestellung abschicken
  const orderRes = http.post(`${BASE_URL}/api/bestellungen`, JSON.stringify({
    supplier_id: SUPPLIER_ID,
    items: [
      {
        titel_id: TITEL_ID,
        menge: Math.floor(Math.random() * 5) + 1,
        preis: 19.99,
        generate_barcodes: true
      }
    ]
  }), reqConf);
  check(orderRes, { 'Order ok': (r) => r.status === 200 || r.status === 404 });

  sleep(5 + Math.random() * 5); // Backoffice macht langsamer
}

// ------------------------------------------------------------------
// SCENARIO C: Inventur
// ------------------------------------------------------------------
export function inventurTasks(data) {
  const reqConf = getReqHeaders(data);
  const randomBarcode = `LTB-${Math.floor(Math.random() * 5000) + 1}`;
  
  const scanRes = http.post(`${BASE_URL}/api/inventur/scan`, JSON.stringify({
    barcode: randomBarcode,
    aktion: "anwesend"
  }), reqConf);
  
  check(scanRes, { 'Inventur Scan ok': (r) => r.status === 200 || r.status === 400 });

  sleep(2 + Math.random() * 2);
}

// ------------------------------------------------------------------
// SCENARIO D: Buchhalter (Mahnwesen & PDFs)
// ------------------------------------------------------------------
export function buchhalterTasks(data) {
  if (!data.students.length) return;
  const reqConf = getReqHeaders(data);
  const student = data.students[Math.floor(Math.random() * data.students.length)];

  // 1. Kontoauszug PDF drucken
  const dsgvoRes = http.get(`${BASE_URL}/api/print/kontoauszug/${student.id}`, reqConf);
  check(dsgvoRes, { 'Kontoauszug PDF ok': (r) => r.status === 200 || r.status === 404 });

  // 2. Mahnlauf PDF für eine Klasse drucken
  const randomKlasse = `LT-${Math.floor(Math.random() * 6) + 1}`;
  const mahnRes = http.get(`${BASE_URL}/api/print/mahnung/klasse/${randomKlasse}`, reqConf);
  check(mahnRes, { 'Mahnung PDF ok': (r) => r.status === 200 || r.status === 404 });

  // 3. Mahnwesen Bulk Send
  const sendRes = http.post(`${BASE_URL}/api/mail/send-bulk-overdue`, JSON.stringify({}), reqConf);
  check(sendRes, { 'Mahn-Mail ok': (r) => r.status === 200 || r.status === 400 });

  // PDFs sind sehr rechenintensiv, daher lange Pause
  sleep(15 + Math.random() * 15);
}

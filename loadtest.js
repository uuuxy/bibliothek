import http from 'k6/http';
import { check, sleep } from 'k6';

// Hier definieren wir das Szenario
export const options = {
  vus: 11,           // Exakt 11 parallele virtuelle Rechner
  duration: '10s',   // Wir feuern 10 Sekunden lang
};

export default function () {
  // Passe die URL an die Route deines Go-Backends an (z. B. der Bulk-Import)
  const url = 'http://localhost:8080/api/orders/bulk'; 
  
  // Wir simulieren einen Offline-Batch von 5 Scans
  const payload = JSON.stringify({
    exemplar_ids: [1001, 1002, 1003, 1004, 1005] 
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Feuer frei: Request an den Server schicken
  const res = http.post(url, payload, params);

  // Wir prüfen knallhart, ob der Server einknickt
  check(res, {
    'ist Status 200 (Erfolg)': (r) => r.status === 200,
    'Server nicht abgestürzt (kein 500)': (r) => r.status !== 500,
  });

  // Eine minimale Pause (300ms), um den Jitter der echten Rechner zu simulieren
  sleep(0.3);
}

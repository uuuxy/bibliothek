// Schülersuche des Vormerkungs-Reiters — über die Schülerdatei (GET /api/schueler?q=,
// Recht view_students), NICHT über die Theken-Suche GET /api/search (perform_actions).
//
// Bis 05.09.2026 rief der Reiter die Theken-Route und warf deren Bücher-Hälfte weg; und
// weil die Route am Theken-Recht hängt, meldete die Schülersuche in der Buchakte ohne
// perform_actions nur noch „Suche nicht möglich" — obwohl der Reiter mit der Theke nichts
// zu tun hat. Die Schülerdatei sucht mit derselben Bedingung (Name, Barcode, Klasse;
// repository.SchuelerSuchBedingung), lässt Abgänger weg (für die wird nichts mehr
// vorgemerkt) und trägt das Recht, das zu den Daten passt.
//
// Eigenes Modul, damit der Weg ohne Browser prüfbar ist (vormerkungSchuelersuche.test.js)
// und der Reiter unter der 200-Zeilen-Regel bleibt.
import { apiFetch } from './apiFetch.js';

// Die Theken-Suche lieferte höchstens zehn; die Schülerdatei kappt eine Suche bewusst
// nicht (sie ist eine Liste). Der Reiter zeigt eine Auswahl, keine Liste — also hier.
export const MAX_TREFFER = 10;

/**
 * @param {string} q
 * @returns {Promise<Array<{ id: string, title: string, subtitle: string }>>}
 */
export async function sucheSchuelerFuerVormerkung(q) {
	const res = await apiFetch(`/api/schueler?q=${encodeURIComponent(q.trim())}`);
	if (!res.ok) return [];
	const liste = await res.json();
	return (Array.isArray(liste) ? liste : []).slice(0, MAX_TREFFER).map((s) => ({
		id: s.id,
		title: `${s.vorname} ${s.nachname}`,
		subtitle: `${s.klasse} · ${s.barcode_id}`
	}));
}

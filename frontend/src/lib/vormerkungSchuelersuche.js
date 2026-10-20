// Schülersuche des Vormerkungs-Reiters, über die Schülerdatei (GET /api/schueler?q=, Recht
// view_students) und nicht über die Theken-Suche GET /api/search (perform_actions): Der
// Reiter hat mit der Theke nichts zu tun, und ohne perform_actions meldete die Suche sonst
// nur „Suche nicht möglich". Die Schülerdatei sucht mit derselben Bedingung (Name, Barcode,
// Klasse; repository.leserSuchBedingung) und lässt Abgänger weg, für die nichts mehr
// vorgemerkt wird.
//
// Eigenes Modul, damit der Weg ohne Browser prüfbar ist (vormerkungSchuelersuche.test.js).
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

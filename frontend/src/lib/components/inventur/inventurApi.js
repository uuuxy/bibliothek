import { apiFetch, leseApiFehler, leseApiFehlerMitDaten } from '../../apiFetch.js';
import { fehlertext } from '../../utils/fehlertext.js';

// Reine API-Aufrufe der Inventur, getrennt von der Zustandslogik im Hook.
// Jede Funktion liefert ein { ok, data?, status, error? }-Objekt, damit der Hook
// Fälle wie 409 (Scope läuft bereits) unterscheiden kann, ohne HTTP-Details zu kennen.

/** @param {any} res */
async function auswerten(res) {
	if (res.ok) {
		return { ok: true, status: res.status, data: await res.json().catch(() => ({})) };
	}
	// `data` wird auch im Fehlerfall durchgereicht, damit der Hook strukturierte 409-Antworten
	// (z. B. "ausser_scope" mit Titel + Warntext) auswerten kann, ohne HTTP-Details zu kennen.
	// Ohne Meldung des Servers bleibt error leer: Den Satz dafür kennt der Aufrufer.
	const { meldung, daten } = await leseApiFehlerMitDaten(res, '');
	return { ok: false, status: res.status, error: meldung, data: daten ?? {} };
}

/**
 * Schickt die Anfrage und wertet die Antwort aus. Ohne Antwort (kein Netz, Frist abgelaufen)
 * kommt der Fehler als Ergebnis mit status 0 zurück, und der Aufrufer meldet ihn wie eine
 * Ablehnung; geworfen blieb er ungefangen, und die Seite tat sichtbar nichts.
 * @param {string} url @param {RequestInit} [options]
 */
async function sende(url, options) {
	let res;
	try {
		res = await apiFetch(url, options);
	} catch (e) {
		return { ok: false, status: 0, error: fehlertext(e), data: {} };
	}
	return auswerten(res);
}

/**
 * Laufende Sessions laden (für die "bereits laufend"-Anzeige). Lehnt der Server ab, wirft der
 * Abruf mit seinem Grund: Eine leere Liste hieße, es laufe keine Inventur.
 */
export async function ladeOffeneSessions() {
	const res = await apiFetch('/api/inventur/sessions');
	if (!res.ok) throw new Error(await leseApiFehler(res));
	return (await res.json().catch(() => [])) || [];
}

/**
 * Neue Session eröffnen.
 * @param {{type: string, signatur?: string, subject?: string, grade?: number}} payload
 */
export async function starteLauf(payload) {
	return sende('/api/inventur/start', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(payload)
	});
}

/**
 * Ein Exemplar in einer Session verbuchen. Ohne Antwort wirft der Aufruf: Die Inventur zeigt den
 * Fehler dann als Ergebnis des Scans, mit Fehlerton (bucheScan in useInventur).
 * @param {string} sessionId
 * @param {string} barcode
 */
export async function scanne(sessionId, barcode) {
	const res = await apiFetch('/api/inventur/scan', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ session_id: sessionId, barcode_id: barcode })
	});
	return auswerten(res);
}

/**
 * Deutet eine Scan-Antwort in { lastScan, zaehlen, stand } — pur, ohne Zustand, damit die
 * Optik-Regeln (Treffer / außer-Scope-Warnung / unbekannt) getrennt vom Hook testbar sind.
 * `zaehlen` ist nur beim echten Treffer true; ein außer-Scope-Buch wird bewusst NICHT
 * mitgezählt, aber als (gelbe) Warnung mit echtem Titel gezeigt. `stand` sind die Zahlen des
 * Servers, gezählt wie beim Abschluss: Ein zweimal gescanntes Buch zählt einmal, ein verliehenes
 * oder gesperrtes im Regal gar nicht, weil der Abschluss es nicht erwartet.
 * @param {{ok: boolean, status: number, data?: any, error?: string}} r
 * @param {string} barcode
 */
export function deuteScanErgebnis(r, barcode) {
	if (r.ok) {
		return {
			zaehlen: true,
			stand: { erwartet: r.data.erwartet, fehlend: r.data.fehlend },
			lastScan: {
				success: true,
				barcode: r.data.barcode_id,
				title: r.data.titel,
				warnings: r.data.warnungen || []
			}
		};
	}
	if (r.status === 409 && r.data?.status === 'ausser_scope') {
		return {
			zaehlen: false,
			lastScan: {
				success: true,
				barcode: r.data.barcode_id || barcode,
				title: r.data.titel || 'Buch',
				warnings: r.data.warnungen?.length
					? r.data.warnungen
					: ['Buch gehört nicht zum Bereich dieser Inventur.']
			}
		};
	}
	return {
		zaehlen: false,
		lastScan: {
			success: false,
			barcode,
			title: 'Unbekanntes Buch',
			warnings: [r.error || 'Unbekannter Fehler']
		}
	};
}

/** @param {string} sessionId */
export async function schliesseAb(sessionId) {
	return sende('/api/inventur/finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ session_id: sessionId })
	});
}

/**
 * Zuletzt abgeschlossene Inventuren laden — die Auswahl, über die man einen früheren
 * Fehlbestand wieder aufruft.
 *
 * Ohne sie war der Bericht flüchtig: Er entstand nur aus der Antwort von schliesseAb()
 * und lebte im Arbeitsspeicher des Browsers. Ein Neuladen oder ein Wechsel an einen
 * anderen Arbeitsplatz — und die Liste der fehlenden Bücher war weg, obwohl der Server
 * sie dauerhaft gespeichert hat.
 */
export async function ladeAbgeschlosseneInventuren() {
	const res = await apiFetch('/api/inventur/abgeschlossen');
	if (!res.ok) throw new Error(await leseApiFehler(res));
	return (await res.json().catch(() => [])) || [];
}

/**
 * Fehlbestand einer bereits abgeschlossenen Inventur nachladen.
 * @param {string} sessionId
 */
export async function ladeFehlbestand(sessionId) {
	return sende(`/api/inventur/fehlbestand?session_id=${encodeURIComponent(sessionId)}`);
}

/** @param {string} sessionId */
export async function brichAb(sessionId) {
	return sende('/api/inventur/abort', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ session_id: sessionId })
	});
}

/** Ein als Verlust gebuchtes Exemplar wurde beim Nachsuchen gefunden — zurück in Umlauf.
 * @param {string} exemplarId */
export async function meldeVerlustGefunden(exemplarId) {
	return sende(`/api/buecher/exemplare/${encodeURIComponent(exemplarId)}/gefunden`, {
		method: 'POST'
	});
}

/** Verlust-Exemplare unwiderruflich löschen.
 * @param {string[]} exemplarIds */
export async function loescheVerlustEndgueltig(exemplarIds) {
	return sende('/api/buecher/exemplare/verlust-endgueltig-loeschen', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ exemplar_ids: exemplarIds })
	});
}

import { apiFetch } from './apiFetch.js';

// Reine API-Aufrufe der Inventur, getrennt von der Zustandslogik im Hook.
// Jede Funktion liefert ein { ok, data?, status, error? }-Objekt, damit der Hook
// Fälle wie 409 (Scope läuft bereits) unterscheiden kann, ohne HTTP-Details zu kennen.

/** @param {any} res */
async function auswerten(res) {
	if (res.ok) {
		return { ok: true, status: res.status, data: await res.json().catch(() => ({})) };
	}
	const body = await res.json().catch(() => ({}));
	// `data` wird auch im Fehlerfall durchgereicht, damit der Hook strukturierte 409-Antworten
	// (z. B. "ausser_scope" mit Titel + Warntext) auswerten kann, ohne HTTP-Details zu kennen.
	return { ok: false, status: res.status, error: body.error || body.message || 'Unbekannter Fehler', data: body };
}

/** Laufende Sessions laden (für die "bereits laufend"-Anzeige). */
export async function ladeOffeneSessions() {
	const res = await apiFetch('/api/inventur/sessions');
	if (!res.ok) return [];
	return (await res.json().catch(() => [])) || [];
}

/**
 * Neue Session eröffnen.
 * @param {{type: string, signatur?: string, subject?: string, grade?: number}} payload
 */
export async function starteSession(payload) {
	const res = await apiFetch('/api/inventur/start', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(payload)
	});
	return auswerten(res);
}

/**
 * Ein Exemplar in einer Session verbuchen.
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
 * Deutet eine Scan-Antwort in { lastScan, zaehlen } — pur, ohne Zustand, damit die
 * Optik-Regeln (Treffer / außer-Scope-Warnung / unbekannt) getrennt vom Hook testbar sind.
 * `zaehlen` ist nur beim echten Treffer true; ein außer-Scope-Buch wird bewusst NICHT
 * mitgezählt, aber als (gelbe) Warnung mit echtem Titel gezeigt.
 * @param {{ok: boolean, status: number, data?: any, error?: string}} r
 * @param {string} barcode
 */
export function deuteScanErgebnis(r, barcode) {
	if (r.ok) {
		return {
			zaehlen: true,
			lastScan: { success: true, barcode: r.data.barcode_id, title: r.data.titel, warnings: r.data.warnungen || [] }
		};
	}
	if (r.status === 409 && r.data?.status === 'ausser_scope') {
		return {
			zaehlen: false,
			lastScan: {
				success: true,
				barcode: r.data.barcode_id || barcode,
				title: r.data.titel || 'Buch',
				warnings: r.data.warnungen?.length ? r.data.warnungen : ['Buch gehört nicht zum Scope dieser Inventur.']
			}
		};
	}
	return { zaehlen: false, lastScan: { success: false, barcode, title: 'Unbekanntes Buch', warnings: [r.error] } };
}

/** @param {string} sessionId */
export async function schliesseAb(sessionId) {
	const res = await apiFetch('/api/inventur/finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ session_id: sessionId })
	});
	return auswerten(res);
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
	if (!res.ok) return [];
	return (await res.json().catch(() => [])) || [];
}

/**
 * Fehlbestand einer bereits abgeschlossenen Inventur nachladen.
 * @param {string} sessionId
 */
export async function ladeFehlbestand(sessionId) {
	const res = await apiFetch(`/api/inventur/fehlbestand?session_id=${encodeURIComponent(sessionId)}`);
	return auswerten(res);
}

/** @param {string} sessionId */
export async function brichAb(sessionId) {
	const res = await apiFetch('/api/inventur/abort', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ session_id: sessionId })
	});
	return auswerten(res);
}

/** Ein als Verlust gebuchtes Exemplar wurde beim Nachsuchen gefunden — zurück in Umlauf.
 * @param {string} exemplarId */
export async function meldeVerlustGefunden(exemplarId) {
	const res = await apiFetch(`/api/buecher/exemplare/${encodeURIComponent(exemplarId)}/gefunden`, {
		method: 'POST'
	});
	return auswerten(res);
}

/** Verlust-Exemplare unwiderruflich löschen.
 * @param {string[]} exemplarIds */
export async function loescheVerlustEndgueltig(exemplarIds) {
	const res = await apiFetch('/api/buecher/exemplare/verlust-endgueltig-loeschen', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ exemplar_ids: exemplarIds })
	});
	return auswerten(res);
}

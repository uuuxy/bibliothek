import { apiFetch, leseApiFehler } from '../../../../lib/apiFetch.js';

/**
 * Die Vorschau des Dialogs „Titel zusammenführen": was das Zusammenführen täte, vom Server
 * gerechnet (GET /api/buecher/titel/{id}/zusammenfuehren?quelle_id=). Sie läuft dort durch
 * denselben Weg wie das Zusammenführen und wird am Ende zurückgenommen, damit Vorschau und Tat
 * nicht auseinanderlaufen. Nur die jüngste Anfrage schreibt; eine Ablehnung, etwa zweier Titel
 * mit ISBN, steht als Fehler da.
 */
export function erzeugeZusammenfuehrenVorschau() {
	/** @type {any | null} */
	let vorschau = $state(null);
	let fehler = $state('');
	let laedt = $state(false);
	let nr = 0;

	/** @param {string} zielId @param {string} quelleId */
	async function lade(zielId, quelleId) {
		const meine = ++nr;
		vorschau = null;
		fehler = '';
		laedt = true;
		try {
			const res = await apiFetch(
				`/api/buecher/titel/${zielId}/zusammenfuehren?quelle_id=${encodeURIComponent(quelleId)}`
			);
			if (!res.ok) {
				const meldung = await leseApiFehler(res);
				if (meine === nr) fehler = meldung;
				return;
			}
			const daten = await res.json();
			if (meine === nr) vorschau = daten;
		} catch {
			if (meine === nr) fehler = 'Netzwerkfehler — die Vorschau hat den Server nicht erreicht.';
		} finally {
			if (meine === nr) laedt = false;
		}
	}

	return /** @satisfies {object} */ ({
		get vorschau() {
			return vorschau;
		},
		get fehler() {
			return fehler;
		},
		get laedt() {
			return laedt;
		},
		lade,
		leeren() {
			nr++;
			vorschau = null;
			fehler = '';
			laedt = false;
		}
	});
}

/**
 * Die Zahl der Exemplare in Worten, wie die Trefferliste und die Zeile „Danach" sie nennen.
 * @param {number | null | undefined} n
 */
export const exemplare = (n) => `${n ?? 0} ${n === 1 ? 'Exemplar' : 'Exemplare'}`;

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
 * Was zum bleibenden Titel kommt, als Zeilen der Vorschau: die gewanderten Zeilen mit ihrer
 * Zahl, dann die Felder, die er übernimmt.
 * @param {any} v die Antwort der Vorschau
 * @returns {string[]}
 */
export function kommtHinzu(v) {
	/** @param {number} n @param {string} eins @param {string} mehr */
	const zahl = (n, eins, mehr) => `${n} ${n === 1 ? eins : mehr}`;
	const zeilen = [];
	if (v.exemplare > 0)
		zeilen.push(
			`${zahl(v.exemplare, 'Exemplar', 'Exemplare')} — Nummer, Etikett und Ausleihen bleiben`
		);
	if (v.vormerkungen > 0) {
		const doppelt = v.vormerkungen_doppelt
			? `; ${zahl(v.vormerkungen_doppelt, 'doppelte fällt', 'doppelte fallen')} weg`
			: '';
		zeilen.push(zahl(v.vormerkungen, 'Vormerkung', 'Vormerkungen') + doppelt);
	}
	if (v.klassensaetze > 0)
		zeilen.push(zahl(v.klassensaetze, 'Klassensatz-Reservierung', 'Klassensatz-Reservierungen'));
	if (v.bestellpositionen > 0)
		zeilen.push(
			zahl(v.bestellpositionen, 'Position einer Bestellung', 'Positionen von Bestellungen')
		);
	if (v.anliegen > 0) zeilen.push(zahl(v.anliegen, 'Meldung', 'Meldungen') + ' von Lehrkräften');
	for (const f of v.uebernommen ?? [])
		zeilen.push(f.wert ? `${f.bezeichnung}: ${f.wert}` : f.bezeichnung);
	return zeilen;
}

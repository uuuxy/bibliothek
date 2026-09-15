// bescheidFormular.svelte.js
// Der Zustand des Bescheid-Dialogs: Vorschlag laden, Auswahl und Beträge halten, erstellen.
//
// Eigene Datei, damit BescheidDialog.svelte unter der 200-Zeilen-Marke bleibt und die
// Zusammenführung von Forderungen und überfälligen Büchern ohne Browser prüfbar ist.
//
// Seit Stufe 2 des Mahnverfahrens (15.09.2026) kennt der Vorschlag zwei Quellen: offene
// FORDERUNGEN (Verlust oder Schaden schon gemeldet) und überfällige BÜCHER ohne Forderung.
// Der Dialog zeigt beide in einer Liste; beim Erstellen bucht der Server die Bücher als
// Verlust und schreibt den Brief in einer Transaktion. Der Rumpf trennt die Quellen
// wieder, wie der Server sie kennt.

import { apiGet, apiPost } from '../../apiFetch.js';

/**
 * Forderungen und überfällige Bücher in EINER Liste. `key` ist die ID der Quelle
 * (Schadensfall oder Ausleihe), `quelle` sagt, welche. Bücher zuerst: Sie sind der
 * häufige Fall, seit der Brief ohne Verlustmeldung je Buch entsteht.
 * @param {any} vorschlag
 * @returns {any[]}
 */
export function zeilenAus(vorschlag) {
	const buecher = (vorschlag?.ausleihen ?? []).map((/** @type {any} */ a) => ({
		...a,
		key: a.ausleihe_id,
		quelle: 'ausleihe',
		art: 'nicht_zurueckgegeben'
	}));
	const forderungen = (vorschlag?.positionen ?? []).map((/** @type {any} */ p) => ({
		...p,
		key: p.schadensfall_id,
		quelle: 'forderung'
	}));
	return [...buecher, ...forderungen];
}

export class BescheidFormular {
	/** @type {any} */
	vorschlag = $state(null);
	laedt = $state(true);
	sendet = $state(false);
	/** @type {Record<string, boolean>} */
	gewaehlt = $state({});
	/** @type {Record<string, number>} */
	betraege = $state({});
	frist = $state('');
	/** Der Brief ist der des Landes (Wortlaut und Konto der Lernmittelfreiheit). Die
	 *  Rechnung der Schülerbücherei ist noch nicht gebaut (Konzept 4.7, Etappe 3); ihre
	 *  Zeilen stehen gesperrt in der Liste, und der Server weist alles andere ab. */
	mittel = 'land';
	/** @type {string} */
	#schuelerId;

	/** @param {string} schuelerId */
	constructor(schuelerId) {
		this.#schuelerId = schuelerId;
	}

	positionen = $derived(zeilenAus(this.vorschlag));
	ausgewaehlt = $derived(this.positionen.filter((p) => this.gewaehlt[p.key]));
	summe = $derived(this.ausgewaehlt.reduce((s, p) => s + (Number(this.betraege[p.key]) || 0), 0));
	/** @type {string[]} */
	fehlt = $derived(this.vorschlag?.fehlende_angaben ?? []);
	bereit = $derived(
		this.ausgewaehlt.length > 0 && this.frist !== '' && this.fehlt.length === 0 && !this.sendet
	);

	/** Der Vorschlag wird EINMAL geholt; danach gehört das Formular dem Menschen.
	 *  Vorgewählt ist, was auf den Brief des Landes darf: Lernmittel. */
	async laden() {
		this.laedt = true;
		try {
			const daten = await apiGet(`/api/schueler/${this.#schuelerId}/bescheid-vorschlag`);
			this.vorschlag = daten;
			this.frist = daten?.frist_bis ?? '';
			for (const p of this.positionen) {
				this.gewaehlt[p.key] = !!p.ist_lernmittel;
				this.betraege[p.key] = p.betrag;
			}
		} catch {
			this.vorschlag = null;
		} finally {
			this.laedt = false;
		}
	}

	/** Der Rumpf für POST /api/schueler/{id}/bescheide: Forderungen und Bücher getrennt. */
	rumpf() {
		const betrag = (/** @type {any} */ p) => Number(this.betraege[p.key]) || 0;
		return {
			mittel: this.mittel,
			frist_bis: this.frist,
			positionen: this.ausgewaehlt
				.filter((p) => p.quelle === 'forderung')
				.map((p) => ({ schadensfall_id: p.key, betrag: betrag(p) })),
			ausleihen: this.ausgewaehlt
				.filter((p) => p.quelle === 'ausleihe')
				.map((p) => ({ ausleihe_id: p.key, betrag: betrag(p) }))
		};
	}

	/** Erstellt den Brief; null, wenn der Server abgewiesen hat (apiFetch zeigt den Toast).
	 *  @returns {Promise<any>} */
	async erstellen() {
		if (!this.bereit) return null;
		this.sendet = true;
		try {
			return await apiPost(`/api/schueler/${this.#schuelerId}/bescheide`, this.rumpf());
		} catch {
			return null;
		} finally {
			this.sendet = false;
		}
	}
}

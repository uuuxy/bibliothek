import { apiFetch, leseApiFehler } from '../../lib/apiFetch.js';

/**
 * Sieht die Eingabe aus wie eine Nummer? Ein Wort ohne Ziffer und ein Satz fragen den Server
 * nicht, sonst wäre jede getippte Silbe eine Anfrage.
 * @param {string} code
 */
function siehtAusWieNummer(code) {
	return /^\S{3,100}$/.test(code) && /\d/.test(code);
}

/**
 * @typedef {{ code: string, stand: 'laeuft' | 'fertig' | 'fehler', titelId?: string,
 *   barcode?: string, ausgesondert?: boolean }} Antwort
 */

/**
 * Die Frage der Katalogsuche nach einem Exemplar. Eine Exemplarnummer, gescannt oder getippt,
 * steht in keiner Angabe eines Titels; der Server erkennt sie wie die Theke, auch über das
 * Etikett von Littera (GET /api/buecher/exemplare/erkennen). Gefragt wird erst, wenn die Eingabe
 * eine Viertelsekunde steht, und es gilt nur die Antwort auf die jüngste Frage.
 *
 * `stand` ist 'laeuft', solange zur aktuellen Eingabe keine Antwort da ist: Die Suche sagt dann
 * noch nicht „nichts gefunden". 'fehler' heißt, die Nummer ließ sich nicht prüfen.
 */
export function erzeugeExemplarFrage() {
	/** @type {Antwort} */
	let antwort = $state.raw({ code: '', stand: 'fertig' });
	let lauf = 0;
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let zeitgeber;

	function stoppe() {
		lauf++;
		clearTimeout(zeitgeber);
	}

	/** @param {string} eingabe */
	function frage(eingabe) {
		stoppe();
		const code = eingabe.trim();
		if (!siehtAusWieNummer(code)) {
			antwort = { code, stand: 'fertig' };
			return;
		}
		const meinLauf = lauf;
		antwort = { code, stand: 'laeuft' };
		zeitgeber = setTimeout(async () => {
			try {
				const res = await apiFetch(
					`/api/buecher/exemplare/erkennen?code=${encodeURIComponent(code)}`
				);
				if (meinLauf !== lauf) return;
				if (!res.ok) throw new Error(await leseApiFehler(res));
				const { treffer } = await res.json();
				if (meinLauf !== lauf) return;
				antwort = treffer
					? {
							code,
							stand: 'fertig',
							titelId: treffer.titel_id,
							barcode: treffer.barcode,
							ausgesondert: treffer.ausgesondert === true
						}
					: { code, stand: 'fertig' };
			} catch {
				if (meinLauf !== lauf) return;
				antwort = { code, stand: 'fehler' };
			}
		}, 250);
	}

	/**
	 * Die Antwort zu diesem Suchtext. Eine Antwort zu einem anderen Text gilt nicht: Bis die neue
	 * da ist, läuft die Frage noch.
	 * @param {string} eingabe
	 * @returns {Antwort}
	 */
	function fuer(eingabe) {
		const code = eingabe.trim();
		if (!siehtAusWieNummer(code)) return { code, stand: 'fertig' };
		return antwort.code === code ? antwort : { code, stand: 'laeuft' };
	}

	return { frage, fuer, stoppe };
}

/**
 * Die Treffer der Titelsuche mit dem Titel des gefundenen Exemplars vorn; steht er schon unter
 * den Treffern, rückt er nach vorn, statt doppelt zu erscheinen.
 * @param {any} titel der Titel des Exemplars, oder null
 * @param {any[]} treffer
 */
export function mitTitelDesExemplars(titel, treffer) {
	if (!titel) return treffer;
	return [titel, ...treffer.filter((b) => b.id !== titel.id)];
}

/**
 * Der Satz über den Karten, wenn die Nummer ein Exemplar trifft, dessen Titel die Liste zeigt.
 * @param {Antwort} exemplar
 * @param {any} titel
 */
export function satzZumExemplar(exemplar, titel) {
	return exemplar.ausgesondert
		? `Exemplar ${exemplar.barcode} ist ausgesondert und gehört zu „${titel.title}“.`
		: `Exemplar ${exemplar.barcode} gehört zu „${titel.title}“.`;
}

/**
 * Der Hinweis, wenn keine Karte zu sehen ist. Ein Exemplar, dessen Titel die Liste nicht zeigt,
 * heißt nicht „keins gefunden": Der Katalog führt nur Titel mit einem Exemplar im Bestand.
 * @param {string} text der Suchtext
 * @param {Antwort} exemplar die Antwort zum Suchtext
 */
export function hinweisOhneTreffer(text, exemplar) {
	if (exemplar.stand === 'fehler')
		return `Kein Titel passt zu „${text}“. Ob ein Exemplar diese Nummer trägt, ließ sich nicht prüfen.`;
	if (exemplar.titelId && exemplar.ausgesondert)
		return `Exemplar ${exemplar.barcode} ist ausgesondert. Sein Titel hat kein Exemplar mehr im Bestand und steht in der Titelliste unter „Ohne Exemplare“.`;
	if (exemplar.titelId)
		return `Exemplar ${exemplar.barcode} gehört zu einem Titel, der in dieser Liste noch fehlt. Neu laden zeigt ihn.`;
	if (siehtAusWieNummer(text)) return `Weder ein Titel noch ein Exemplar passt zu „${text}“.`;
	return `Kein Titel passt zu „${text}“.`;
}

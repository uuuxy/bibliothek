import { apiFetch, leseApiFehler } from '../../apiFetch.js';
import { fehlertext } from '../../utils/fehlertext.js';
import { normalisiereScan } from '../../utils/scanEinordnen.js';
import { grenzeAus, istGekappt } from '../../utils/listenGrenze.js';

/**
 * Die Serversuche der Schülerdatei. Eigene Datei wie ausweisdruck.svelte.js:
 * Schuelerdatei steht an der Größen-Ratsche, und die Suche ist das Stück, das
 * nichts mit dem Führen der Liste zu tun hat.
 *
 * Gesucht wird auf dem SERVER. Vorher filterte die Ansicht im Browser über die
 * gelieferte Liste — und die ist bei 500 Zeilen gekappt. Bei 875 Schülern waren 375
 * über die Suche schlicht nicht erreichbar, welche genau hing an der alphabetischen
 * Reihenfolge der Klassennamen. Für den Benutzer sah das nach Zufall aus.
 *
 * Nebeneffekt, der den Ausschlag gab: Die Serversuche ist dieselbe wie an der Theke
 * (suchnorm) — "Muller" findet Müller, "Hoffmann Lena" dasselbe wie "Lena Hoffmann".
 * Der Browser-Filter konnte beides nicht.
 */
export function erzeugeSchuelerSuche() {
	/** @type {any[]} */
	let schueler = $state.raw([]);
	/** Die Grenze, die der Server für die geladene Liste nennt; 0, wenn keine gilt. */
	let grenze = $state(0);
	let laedt = $state(false);
	/** Leer heißt leer — ein Ladefehler heißt Ladefehler (wie im Papierkorb). */
	let ladefehler = $state('');
	let sucheLaeuft = $state(false);
	let suchtext = $state('');
	/** Jahrgangsfilter, '' = alle. Serverseitig wie die Suche:
	 *  Ein Filter im Browser säße hinter der Kappung bei 500 Zeilen und zeigte dann einen
	 *  Teil des Jahrgangs, ohne das zu sagen. */
	let jahrgang = $state('');
	/** Sortierung, '' = Reihenfolge der Kartei.
	 *  Serverseitig wie Suche und Filter: Im Browser sortiert säße die Sortierung hinter
	 *  der Kappung bei 500 Zeilen — sie ordnete dann die ersten 500 der Kartei-Reihenfolge
	 *  um, statt die ersten 500 der gewählten. Das sieht richtig aus und ist es nicht. */
	let sortierSpalte = $state('');
	let sortierAbsteigend = $state(false);
	/** Die besetzten Jahrgänge fürs Auswahlfeld — vom Server, damit die Ableitung
	 *  „Klassenname → Jahrgang" nicht ein zweites Mal in JavaScript entsteht. */
	let jahrgaenge = $state.raw(/** @type {number[]} */ ([]));
	/** Der Grund, wenn die Jahrgänge nicht geladen werden konnten. Leer heißt leer, ein
	 *  Ladefehler heißt Ladefehler — dieselbe Regel wie bei der Liste darüber. Ohne
	 *  diesen Zustand stünde im Auswahlfeld nur „Alle Jahrgänge", und das sähe aus wie
	 *  „diese Schule hat keine Jahrgänge". */
	let jahrgaengeFehler = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;

	// Nur die JÜNGSTE Anfrage darf die Liste schreiben: Überholen sich zwei Suchen, entschiede
	// sonst der Server, welche Antwort zuletzt eintrifft.
	let ladeNr = 0;

	async function lade() {
		const nr = ++ladeNr;
		laedt = true;
		try {
			const q = normalisiereScan(suchtext);
			const text = q ? `&q=${encodeURIComponent(q)}` : '';
			const jg = jahrgang ? `&jahrgang=${encodeURIComponent(jahrgang)}` : '';
			const richtung = sortierAbsteigend ? 'ab' : 'auf';
			const so = sortierSpalte
				? `&sortierung=${encodeURIComponent(sortierSpalte)}&richtung=${richtung}`
				: '';
			// art=alle: die Leserdatei. Ohne den Zusatz liefert die Tür nur Schüler, wie es Ehemalige
			// und Vormerken brauchen; dort wäre ein Kollege in der Liste falsch.
			const res = await apiFetch(`/api/schueler?art=alle${text}${jg}${so}`);
			// Nur die jüngste Anfrage schreibt, aber sie schreibt in jedem Fall: Sonst blieben
			// nach einem gescheiterten Lauf die Treffer der vorigen Suche unter dem neuen
			// Suchtext stehen, und an der Theke bucht genau diese Form auf den falschen
			// Schüler.
			// Erst gelesen, dann verglichen: Auch das Lesen des Rumpfs dauert.
			let liste = [];
			let fehler = '';
			let neueGrenze = 0;
			if (res.ok) {
				neueGrenze = grenzeAus(res);
				liste = (await res.json()) || [];
			} else fehler = await leseApiFehler(res);
			if (nr !== ladeNr) return;
			schueler = liste;
			grenze = neueGrenze;
			ladefehler = fehler;
		} catch (err) {
			if (nr === ladeNr) {
				schueler = [];
				grenze = 0;
				ladefehler = 'Die Leserdatei konnte nicht geladen werden (Netzwerkfehler).';
			}
			console.error('Fehler beim Laden der Leserdatei:', err);
		} finally {
			if (nr === ladeNr) {
				laedt = false;
				sucheLaeuft = false;
			}
		}
	}

	// Erste Ladung sofort, wie erzeugeAusweisdruck sein Design lädt.
	lade();

	/** Die Jahrgangsliste einmal holen.
	 *
	 *  Scheitert der Abruf, sagt das Auswahlfeld es und sperrt sich, statt eine leere
	 *  Liste anzubieten: Ein Filter, der keine Jahrgänge kennt, sieht sonst aus wie
	 *  einer, der nichts zu filtern findet; wie im Kollegiums-Portal, wo ein gescheiterter
	 *  erster Abruf das auch sagt, statt „nichts da" zu zeigen. Die Leserdatei selbst
	 *  bleibt benutzbar; gesucht werden kann weiter. */
	async function ladeJahrgaenge() {
		try {
			const res = await apiFetch('/api/jahrgaenge');
			if (!res.ok) throw new Error(await leseApiFehler(res));
			jahrgaenge = (await res.json()) || [];
			jahrgaengeFehler = '';
		} catch (err) {
			jahrgaenge = [];
			jahrgaengeFehler = fehlertext(err);
			console.error('Jahrgänge konnten nicht geladen werden:', err);
		}
	}
	ladeJahrgaenge();

	return /** @satisfies {object} */ ({
		get schueler() {
			return schueler;
		},
		get jahrgang() {
			return jahrgang;
		},
		set jahrgang(wert) {
			jahrgang = wert;
		},
		get jahrgaenge() {
			return jahrgaenge;
		},
		get jahrgaengeFehler() {
			return jahrgaengeFehler;
		},
		get beschaeftigt() {
			return laedt || sucheLaeuft;
		},
		get ladefehler() {
			return ladefehler;
		},
		get suchtext() {
			return suchtext;
		},
		set suchtext(wert) {
			suchtext = wert;
		},
		get sortierung() {
			return { spalte: sortierSpalte, absteigend: sortierAbsteigend };
		},
		/** Ein Klick auf einen Spaltenkopf: fremde Spalte → aufsteigend, dieselbe →
		 *  Richtung umdrehen. Kein dritter Zustand „unsortiert": Eine Liste ist immer
		 *  irgendwie sortiert, und ein Klick, der die Ordnung wegnimmt, verwirrt mehr
		 *  als er hilft. */
		sortiere(spalte) {
			if (sortierSpalte === spalte) {
				sortierAbsteigend = !sortierAbsteigend;
			} else {
				sortierSpalte = spalte;
				sortierAbsteigend = false;
			}
			lade();
		},
		get suchend() {
			return suchtext.trim().length > 0 || jahrgang !== '';
		},
		get gekuerzt() {
			// Wann die Liste gekappt wird, entscheidet der Server (repository.LeserListeGrenze):
			// ohne Suchtext ja, auch mit Klassen- oder Jahrgangsfilter. Er nennt die Grenze mit
			// der Antwort.
			return istGekappt(schueler.length, grenze);
		},
		lade,

		// Tippen wird entprellt, damit nicht jeder Tastendruck eine Abfrage auslöst.
		// 300 ms wie in der Omnibox — dieselbe Eingabegeschwindigkeit, dieselbe Wartezeit.
		angestossen() {
			sucheLaeuft = true;
			clearTimeout(timer);
			timer = setTimeout(lade, 300);
		}
	});
}

// lmfplanPlaner.svelte.js — der Zustand des LMF-Planers und die vier Wege, die ihn
// bewegen: laden, Vorschau rechnen, speichern, veröffentlichen, verwerfen.
//
// Eigene Datei seit dem Rasterdurchgang am 06.09.2026: Die Fixe des Durchgangs
// (Sequenznummer im Ladepfad, Zustands-Rückstellung, `catch` in den Schreibwegen) haben
// LmfPlan.svelte über die 200-Zeilen-Regel gehoben. Die Ratsche lockert man dafür nicht —
// hier liegt jetzt das Verhalten, dort nur noch der Bildschirm. Dieselbe Aufteilung wie
// bei den Zeilen-Umformungen (lmfplanZeilen.js) und dem Mahnwesen-Store.
//
// `erzeugePlaner()` gibt EINEN Planer je Bildschirm zurück, kein Modul-Singleton: Der
// Zustand darf den Bildschirm nicht überleben, sonst zeigte ein erneutes Öffnen für
// einen Moment den Plan von vorhin (Rasterfrage 8, Ansichtswechsel).

import { showToast } from '../inventur/lib/store.svelte.js';
import { bestaetigen } from './stores/bestaetigung.svelte.js';
import * as dienst from './lmfplanDienst.js';

/** @typedef {import('./lmfplanDienst.js').PlanStand} PlanStand */
/** @typedef {import('./lmfplanDienst.js').PlanEntwurf} PlanEntwurf */

export function erzeugePlaner() {
	const zustand = $state({
		art: 'rueckgabe',
		/** @type {PlanStand | null} */
		stand: null,
		/** @type {PlanEntwurf} */
		entwurf: dienst.leererEntwurf(),
		/** @type {{ datum: string, stunde: number }[]} */
		plaetze: [],
		/** @type {import('./lmfplanDienst.js').Ausfall[]} */
		ausfaelle: [],
		/** Der gerechnete Beginn des Büchertauschs (der Plan hängt am Ende) — vom Server.
		 *  @type {{ datum: string, stunde: number } | null} */
		beginn: null,
		/** Die gerade eingeplante Zeile — die Tabelle scrollt hin und hebt sie kurz hervor.
		 *  @type {{ index: number } | null} */
		markiert: null,
		laedt: true,
		speichert: false,
		// Gescheitertes Laden ist ein eigener Zustand, kein leerer Plan (ui/LadeFehler.svelte):
		// sonst ersetzte „Plan speichern" den echten Plan durch die Regel-Reihenfolge.
		ladeFehler: false,
		// An einem ANDEREN Platz wurde der Plan geändert, während hier ungespeicherte
		// Arbeit steht. Dann wird NICHT nachgeladen (das würfe sie weg), sondern
		// hingewiesen — siehe fremdesSignal().
		fremdeAenderung: false
	});

	// `ladeNr` ist dieselbe Sequenznummer wie in der Vorschau (Rasterfrage 6) — der
	// Ladepfad hatte sie bis zum 06.09.2026 NICHT. Beim Umschalten der Art laufen zwei
	// GETs; kam die ältere Antwort zuletzt, stand am Ende die Art des einen Plans über
	// den Zeilen des anderen, und „Plan speichern" schickte die Büchertausch-Reihenfolge
	// an PUT /api/lmf-plan/ausgabe — wo sie den weiter veröffentlichten Ausgabe-Plan
	// überschrieb. Der Zustand heilte nicht: Die Vorschau rechnete die fremde Reihenfolge
	// anstandslos durch, die Tabelle sah stimmig aus.
	let ladeNr = 0;
	/**
	 * Steht hier Arbeit, die der Server nicht kennt? Verglichen wird der Entwurf gegen
	 * den, der aus dem geladenen Stand entsteht — dieselbe Ableitung, die `lade()`
	 * benutzt. Kein eigenes „schmutzig"-Flag an jeder Bearbeitung: Das wäre eine zweite
	 * Wahrheit über denselben Sachverhalt, und jede vergessene Stelle liesse es lügen.
	 */
	function hatUngespeichertes() {
		if (!zustand.stand) return false;
		return JSON.stringify(zustand.entwurf) !== JSON.stringify(dienst.entwurfAus(zustand.stand));
	}

	/**
	 * Der Plan wurde anderswo geändert (SSE, siehe api/lmf_plan_live.go).
	 *
	 * Stilles Nachladen ist hier NICHT immer richtig: `lade()` setzt den Entwurf auf den
	 * Server-Stand zurück, und wer gerade eine Reihenfolge zusammengezogen hat, verlöre
	 * sie ohne ein Wort. Deshalb nur nachladen, wenn nichts Ungespeichertes offen ist;
	 * sonst stehen lassen und sagen, dass es einen neueren Stand gibt.
	 */
	function fremdesSignal() {
		if (hatUngespeichertes()) {
			zustand.fremdeAenderung = true;
			return;
		}
		lade();
	}

	async function lade() {
		const meine = ++ladeNr;
		const meineArt = zustand.art;
		zustand.laedt = true;
		zustand.fremdeAenderung = false;
		// Alles, was am vorigen Plan hing, geht zurück auf Anfang. Sonst stünden die Plätze
		// des anderen Plans in der Tabelle, bis die neue Vorschau kommt (250 ms + Rundlauf)
		// — und ein Klick auf eine Datumszelle nähme genau diesen fremden Platz als festen
		// Platz mit (LmfPlanPlatzZellen). `stand = null` sperrt zugleich die Kopf-Aktionen,
		// die sonst auf dem alten Plan arbeiten.
		zustand.stand = null;
		zustand.entwurf = dienst.leererEntwurf();
		zustand.plaetze = [];
		zustand.ausfaelle = [];
		zustand.beginn = null;
		zustand.markiert = null;
		try {
			const geladen = await dienst.ladeStand(meineArt);
			if (meine !== ladeNr) return; // eine jüngere Anfrage ist unterwegs oder schon da
			zustand.stand = geladen;
			zustand.entwurf = dienst.entwurfAus(geladen);
			zustand.ladeFehler = false;
		} catch (e) {
			if (meine !== ladeNr) return;
			zustand.ladeFehler = true;
			showToast(`${e}`, 'error');
		} finally {
			if (meine === ladeNr) zustand.laedt = false;
		}
	}

	/** Art wechseln — und den Plan der neuen Art laden. @param {string} w */
	function waehleArt(w) {
		zustand.art = w;
		lade();
	}

	// Die Plätze rechnet der Server (kein JavaScript-Zwilling der Verteilung). `laufNr`
	// ist die Sequenznummer wie im orderStore.
	let laufNr = 0;
	/** @param {string} art @param {PlanEntwurf} entwurf */
	async function vorschau(art, entwurf) {
		const meine = ++laufNr;
		try {
			const z = await dienst.rechneVorschau(art, entwurf);
			if (meine !== laufNr) return; // eine jüngere Anfrage ist schon unterwegs oder da
			zustand.plaetze = z.plaetze.map((p) => ({ datum: p.datum, stunde: p.stunde }));
			zustand.ausfaelle = z.ausfaelle;
			zustand.beginn = z.beginn;
		} catch (e) {
			if (meine === laufNr) showToast(`${e}`, 'error');
		}
	}

	/** @param {string} k @param {number} [vor] */
	function klasseHinein(k, vor) {
		const erg = dienst.klasseHinein(zustand.entwurf, k, vor);
		zustand.entwurf = erg.entwurf;
		if (erg.index !== null) zustand.markiert = { index: erg.index };
	}

	// Die drei Schreibwege hatten bis zum Rasterdurchgang am 06.09.2026 kein `catch`
	// (Frage 5): Der Dienst wirft bei Netzfehler und beim 10-Sekunden-Zeitlimit von
	// apiFetch, und die Ablehnung landete in window.unhandledrejection — für den Bediener
	// wurde der Knopf einfach wieder aktiv. KEIN Toast, keine Meldung, und ein
	// clientseitig abgebrochenes PUT kann serverseitig trotzdem gelaufen sein.
	/** @param {unknown} e @param {string} was */
	function schreibfehler(e, was) {
		showToast(`${was} fehlgeschlagen: ${e instanceof Error ? e.message : e}`, 'error');
	}

	async function speichern() {
		zustand.speichert = true;
		try {
			const erg = await dienst.speicherePlan(zustand.art, zustand.entwurf);
			showToast(erg.meldung, erg.ok ? 'success' : 'error');
			if (erg.ok) await lade();
		} catch (e) {
			schreibfehler(e, 'Speichern');
		} finally {
			zustand.speichert = false;
		}
	}

	// Veröffentlichen speichert den Entwurf zuerst — was die Schulleitung im PDF sah und
	// was das Kollegium gleich sieht, soll derselbe Stand sein.
	async function veroeffentlichen() {
		const fristen =
			zustand.art === 'rueckgabe' ? ', und die Termine werden die Fristen der Klassen' : '';
		if (
			!(await bestaetigen({
				titel: 'Plan veröffentlichen?',
				text: `Das Kollegium sieht ihn dann im Portal${fristen}.`,
				aktion: 'Veröffentlichen'
			}))
		)
			return;
		zustand.speichert = true;
		try {
			const gespeichert = await dienst.speicherePlan(zustand.art, zustand.entwurf);
			if (!gespeichert.ok) {
				showToast(gespeichert.meldung, 'error');
				return;
			}
			const erg = await dienst.veroeffentlichePlan(zustand.art);
			showToast(erg.meldung, erg.ok ? 'success' : 'error');
			// Auch nach einem Fehlschlag neu laden: Veröffentlichen stempelt zuerst und
			// koppelt danach die Fristen (zwei Schritte, keine gemeinsame Transaktion).
			// Bricht der zweite ab, IST der Plan veröffentlicht — der Planer darf dann nicht
			// weiter einen Entwurf zeigen.
			await lade();
		} catch (e) {
			schreibfehler(e, 'Veröffentlichen');
			await lade();
		} finally {
			zustand.speichert = false;
		}
	}

	async function verwerfen() {
		if (!zustand.stand?.plan) return;
		if (
			!(await bestaetigen({
				titel: `Plan vom ${dienst.datumKurz(zustand.stand.plan.erster_tag)} verwerfen?`,
				aktion: 'Verwerfen',
				gefaehrlich: true
			}))
		)
			return;
		try {
			const erg = await dienst.verwerfePlan(zustand.art);
			showToast(erg.meldung, erg.ok ? 'success' : 'error');
			// Wie beim Veröffentlichen: Gelöscht wird zuerst, die Fristen kehren danach
			// zurück. Nach einem Fehlschlag ist der Plan womöglich weg — neu laden, sonst
			// zeigt der Planer einen Plan, den es nicht mehr gibt.
			await lade();
		} catch (e) {
			schreibfehler(e, 'Verwerfen');
			await lade();
		}
	}

	// Mit Entwurf: das PDF geht zur Abnahme an die Schulleitung.
	const pdf = () => dienst.ladePdf(false, true).catch((e) => showToast(`${e}`, 'error'));

	return {
		zustand,
		lade,
		fremdesSignal,
		hatUngespeichertes,
		waehleArt,
		vorschau,
		klasseHinein,
		speichern,
		veroeffentlichen,
		verwerfen,
		pdf
	};
}

// lmfplanPlaner.svelte.js — der Zustand des LMF-Planers und die Wege, die ihn bewegen:
// laden, Vorschau rechnen, speichern, veröffentlichen, verwerfen. Der Bildschirm steht in
// LmfPlan.svelte, die Zeilen-Umformungen in lmfplanZeilen.js.
//
// `erzeugePlaner()` gibt einen Planer je Bildschirm zurück, kein Modul-Singleton: Der
// Zustand darf den Bildschirm nicht überleben, sonst zeigte ein erneutes Öffnen für
// einen Moment den Plan von vorhin.

import { melde } from '../../../inventur/lib/store.svelte.js';
import { bestaetigen } from '../../stores/bestaetigung.svelte.js';
import * as dienst from './lmfplanDienst.js';
import { fehlertext } from '../../utils/fehlertext.js';

/** @typedef {import('./lmfplanDienst.js').PlanStand} PlanStand */
/** @typedef {import('./lmfplanDienst.js').PlanEntwurf} PlanEntwurf */

// Der Dienst wirft bei Netzfehler und Zeitlimit (apiFetch). Ohne Meldung würde nur der Knopf
// wieder aktiv; ein im Browser abgebrochenes PUT kann am Server trotzdem gelaufen sein.
/** @param {unknown} e @param {string} was */
function schreibfehler(e, was) {
	melde(`${was} fehlgeschlagen: ${fehlertext(e)}`, 'error');
}

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
		// An einem anderen Platz wurde der Plan geändert, während hier ungespeicherte Arbeit
		// steht (fremdesSignal, oder der Server lehnt das Speichern ab): hinweisen, nicht laden.
		fremdeAenderung: false
	});

	// `ladeNr` ist dieselbe Sequenznummer wie in der Vorschau: Beim Umschalten der Art laufen
	// zwei GETs. Käme die ältere Antwort zuletzt, stünde die Art des einen Plans über den
	// Zeilen des anderen, und „Plan speichern" schickte die Büchertausch-Reihenfolge an PUT
	// /api/lmf-plan/ausgabe, wo sie den veröffentlichten Ausgabe-Plan überschriebe. Von selbst
	// heilt das nicht: Die Vorschau rechnet die fremde Reihenfolge anstandslos durch.
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
	 * Der Plan wurde anderswo geändert (SSE, siehe api/lernmittel/lmf_plan_live.go).
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
			melde(`${e}`, 'error');
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
	// ist die Sequenznummer wie im bestellStore.
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
			if (meine === laufNr) melde(`${e}`, 'error');
		}
	}

	/** @param {string} k @param {number} [vor] */
	function klasseHinein(k, vor) {
		const erg = dienst.klasseHinein(zustand.entwurf, k, vor);
		zustand.entwurf = erg.entwurf;
		if (erg.index !== null) zustand.markiert = { index: erg.index };
	}

	async function speichern() {
		zustand.speichert = true;
		try {
			const erg = await dienst.speicherePlan(zustand.art, zustand.entwurf);
			melde(erg.meldung, erg.ok ? 'success' : 'error');
			if (erg.konflikt) zustand.fremdeAenderung = true;
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
				melde(gespeichert.meldung, 'error');
				if (gespeichert.konflikt) zustand.fremdeAenderung = true;
				return;
			}
			const erg = await dienst.veroeffentlichePlan(zustand.art);
			melde(erg.meldung, erg.ok ? 'success' : 'error');
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
			melde(erg.meldung, erg.ok ? 'success' : 'error');
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
	const pdf = () => dienst.ladePdf(false, true).catch((e) => melde(`${e}`, 'error'));

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

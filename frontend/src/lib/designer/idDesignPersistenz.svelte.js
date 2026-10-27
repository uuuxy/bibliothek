import { apiFetch, leseApiFehler } from '../apiFetch.js';
import { fehlertext } from '../utils/fehlertext.js';
import { bestaetigen } from '../stores/bestaetigung.svelte.js';
import { applyDesign, resetDesign, wendeSchulstammdatenAn } from './idDesignerStore.svelte.js';
import { AUSWEIS_VORLAGEN, wendeVorlageAn } from './ausweisVorlagen.js';

/**
 * Laden, Speichern und Zurücksetzen des zentral abgelegten Ausweis-Designs, getrennt von
 * AusweisDesigner.svelte: Der Bildschirm macht zwei Dinge, und die Ablage ist das, was
 * nichts mit der Leinwand zu tun hat.
 *
 * Der Auto-Save-Effekt bleibt in der Komponente: Ein $effect gehört an den Lebenszyklus,
 * der ihn wieder abräumt.
 */

/** @returns {{ readonly zustand: 'idle'|'saving'|'saved'|'error', readonly geladen: boolean, readonly ladefehler: string, readonly speicherfehler: string, laden: () => Promise<void>, speichern: (body: string) => Promise<void>, erneutSpeichern: () => Promise<void>, beginneSpeichern: () => void, zuruecksetzen: () => Promise<boolean>, vorlageAnwenden: (kennung: string) => Promise<boolean> }} */
export function erzeugeDesignAblage() {
	/** @type {'idle'|'saving'|'saved'|'error'} */
	let zustand = $state('idle');
	// Erst nach dem initialen Laden auto-speichern, sonst überschrieben die
	// Store-Vorgabewerte den geladenen Stand.
	//
	// `geladen` gilt nur bei einer echten Antwort, nicht nach einem gescheiterten Laden: Dann
	// zeigt die Leinwand Vorgabewerte, die Heilung der Schulstammdaten fasst den Store an, und
	// 800 ms später ersetzte ein PUT mit dem Vorgabe-Design das Design der Schule auf allen
	// Arbeitsplätzen, ohne einen Klick. Ein leeres {} beim Erststart ist eine (200er) Antwort
	// und darf auto-speichern.
	let geladen = $state(false);
	// Nicht leer = die Leinwand zeigt NICHT den zentralen Stand. Der Bildschirm sagt es,
	// statt still mit Vorgabewerten weiterzuarbeiten.
	let ladefehler = $state('');
	// Scheitert das Speichern, gilt der Stand nur an diesem Platz. Der Grund steht da, und der Stand
	// wartet auf „Erneut speichern"; sonst ginge er erst mit der nächsten Änderung hinaus. Nur das
	// jüngste Speichern zählt: Ein älterer Stand, der zuletzt scheitert, überschriebe den neueren.
	let speicherfehler = $state('');
	/** @type {string | null} */
	let ungespeichert = null;
	let speicherLauf = 0;

	return /** @satisfies {object} */ ({
		get zustand() {
			return zustand;
		},
		get geladen() {
			return geladen;
		},
		get ladefehler() {
			return ladefehler;
		},
		get speicherfehler() {
			return speicherfehler;
		},

		/** Lädt das zentral gespeicherte Design. Leeres {} (Erststart) → Vorgabewerte. */
		async laden() {
			try {
				const res = await apiFetch('/api/ausweis-layout');
				if (!res.ok) {
					ladefehler = `Das gespeicherte Ausweis-Design konnte nicht geladen werden (${await leseApiFehler(res)}). Es wird nichts gespeichert, solange das so ist.`;
					return;
				}
				applyDesign(await res.json());
				ladefehler = '';
				geladen = true;
			} catch (e) {
				ladefehler =
					'Das gespeicherte Ausweis-Design konnte nicht geladen werden (Netzwerkfehler). Es wird nichts gespeichert, solange das so ist.';
				console.error('Ausweis-Design konnte nicht geladen werden:', e);
				return;
			}
			// NACH applyDesign(): Sonst überschreibt das geladene Design (auch eines, das
			// den Platzhalter noch trägt) die geheilten Werte sofort wieder. Und nur nach
			// einem GELUNGENEN Laden: Die Heilung fasst den Store an, und auf einer
			// Leinwand voller Vorgabewerte wäre das der erste Schritt zum Überschreiben.
			await heileSchulstammdaten();
		},

		beginneSpeichern() {
			zustand = 'saving';
		},

		/** @param {string} body */
		async speichern(body) {
			const meiner = ++speicherLauf;
			let grund = '';
			try {
				const res = await apiFetch('/api/ausweis-layout', {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body
				});
				if (!res.ok) grund = await leseApiFehler(res);
			} catch (e) {
				grund = fehlertext(e);
			}
			if (meiner !== speicherLauf) return;
			speicherfehler = grund;
			ungespeichert = grund ? body : null;
			zustand = grund ? 'error' : 'saved';
		},

		/** Schickt den Stand noch einmal, dessen Speichern zuletzt scheiterte. */
		async erneutSpeichern() {
			if (ungespeichert === null) return;
			zustand = 'saving';
			await this.speichern(ungespeichert);
		},

		/**
		 * Verwirft das Design und stellt die Vorgabewerte her. Der Auto-Save-Effekt der
		 * Komponente schreibt das Ergebnis anschließend zentral — die Rückfrage ist
		 * deshalb Pflicht: Der Schritt trifft ALLE Arbeitsplätze, nicht nur diesen Browser.
		 *
		 * @returns {Promise<boolean>} true, wenn zurückgesetzt wurde (Aufrufer räumt seine
		 *   Elementauswahl ab — die bisherigen IDs gibt es danach nicht mehr).
		 */
		async zuruecksetzen() {
			const ok = await bestaetigen({
				titel: 'Ausweis-Design auf die Standardwerte zurücksetzen?',
				text:
					'Alle eigenen Anpassungen an Vorder- und Rückseite gehen verloren — ' +
					'auch für die anderen Arbeitsplätze, da das Design zentral gespeichert wird.',
				aktion: 'Zurücksetzen',
				gefaehrlich: true
			});
			if (!ok) return false;
			resetDesign();
			// resetDesign() setzt den Kopf zurück auf PLATZHALTER_SCHULNAME. Ohne diesen
			// erneuten Aufruf würfe „Standardwerte wiederherstellen" einen bereits
			// geheilten echten Schulnamen wieder auf den Platzhalter zurück, ohne dass er
			// sich von selbst erneut heilt.
			await heileSchulstammdaten();
			return true;
		},

		/**
		 * Füllt beide Seiten mit einer Design-Vorlage (ausweisVorlagen.js). Gleiche
		 * Spielregeln wie zuruecksetzen(): Rückfrage ist Pflicht (der Auto-Save trägt das
		 * Ergebnis auf ALLE Arbeitsplätze), und die Vorlagen-Platzhalter werden sofort
		 * mit den echten Schul-Stammdaten geheilt.
		 *
		 * @param {string} kennung
		 * @returns {Promise<boolean>} true, wenn angewendet (Aufrufer räumt seine
		 *   Elementauswahl ab — die bisherigen IDs gibt es danach nicht mehr).
		 */
		async vorlageAnwenden(kennung) {
			const name = AUSWEIS_VORLAGEN.find((v) => v.value === kennung)?.label ?? kennung;
			const ok = await bestaetigen({
				titel: `Design-Vorlage „${name}" anwenden?`,
				text:
					'Vorder- und Rückseite werden ersetzt; eigene Anpassungen gehen verloren — ' +
					'auch für die anderen Arbeitsplätze, da das Design zentral gespeichert wird.',
				aktion: 'Anwenden',
				gefaehrlich: true
			});
			if (!ok) return false;
			if (!wendeVorlageAn(kennung)) return false;
			await heileSchulstammdaten();
			return true;
		}
	});
}

// /api/einstellungen verlangt manage_settings — wer den Ausweis-Designer nur zum Drucken
// öffnet (view_students reicht dafür), bekäme sonst ein sichtbares Berechtigungs-Toast
// für eine reine Komfortfunktion. Deshalb roh über apiFetch und bei jedem Fehler
// (auch 403) still nichts tun.
async function heileSchulstammdaten() {
	try {
		const res = await apiFetch('/api/einstellungen');
		if (!res.ok) return;
		const daten = await res.json();
		const adresse = [
			daten.schule_strasse,
			[daten.schule_plz, daten.schule_ort].filter(Boolean).join(' ')
		]
			.filter(Boolean)
			.join(', ');
		wendeSchulstammdatenAn(daten.schule_name ?? '', adresse);
	} catch {
		/* Komfortfunktion — Platzhalter bleibt stehen */
	}
}

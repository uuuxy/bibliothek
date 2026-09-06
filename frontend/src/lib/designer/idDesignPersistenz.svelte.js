import { apiFetch } from '../apiFetch.js';
import { applyDesign, resetDesign, wendeSchulstammdatenAn } from './idDesignerStore.svelte.js';
import { AUSWEIS_VORLAGEN, wendeVorlageAn } from './ausweisVorlagen.js';

/**
 * Laden, Speichern und Zurücksetzen des zentral abgelegten Ausweis-Designs.
 *
 * Aus StudentIdDesigner.svelte herausgelöst (24.08.2026): Die Komponente lag an der
 * 200-Zeilen-Marke und hat zweimal hintereinander die Größen-Ratsche gerissen, als
 * etwas an ihr gebaut wurde. Kommentare zu kürzen, um einen Zähler zu bedienen, ist die
 * falsche Antwort darauf — der Bildschirm macht schlicht zwei Dinge, und die Ablage ist
 * das Ding, das nichts mit der Leinwand zu tun hat.
 *
 * Der Auto-Save-Effekt bleibt bewusst in der Komponente: Ein $effect gehört an den
 * Lebenszyklus, der ihn wieder abräumt.
 */

/** @returns {{ readonly zustand: 'idle'|'saving'|'saved'|'error', readonly geladen: boolean, readonly ladefehler: string, laden: () => Promise<void>, speichern: (body: string) => Promise<void>, beginneSpeichern: () => void, zuruecksetzen: () => Promise<boolean>, vorlageAnwenden: (kennung: string) => Promise<boolean> }} */
export function erzeugeDesignAblage() {
	/** @type {'idle'|'saving'|'saved'|'error'} */
	let zustand = $state('idle');
	// Erst nach dem initialen Laden auto-speichern, sonst überschrieben die
	// Store-Vorgabewerte den geladenen Stand.
	//
	// Genau das geschah bis zum Sweep am 06.09.2026 bei einem FEHLGESCHLAGENEN Laden:
	// `geladen` wurde im finally gesetzt, egal wie der Abruf ausging. Die Leinwand zeigte
	// dann die Vorgabewerte, der Auto-Save-Effekt war scharf — und die anschließende
	// Heilung der Schulstammdaten fasst den Store an. Das allein genügte: 800 ms später
	// ging ein PUT mit dem VORGABE-Design an den Server und ersetzte das Design der
	// Schule auf ALLEN Arbeitsplätzen. Ohne einen Klick, nur weil jemand den Bildschirm
	// öffnete, während der GET scheiterte.
	//
	// Deshalb: `geladen` nur bei einer echten Antwort. Ein leeres {} beim Erststart ist
	// eine (200er) Antwort und darf weiterhin auto-speichern.
	let geladen = $state(false);
	// Nicht leer = die Leinwand zeigt NICHT den zentralen Stand. Der Bildschirm sagt es,
	// statt still mit Vorgabewerten weiterzuarbeiten.
	let ladefehler = $state('');

	// /api/einstellungen verlangt manage_settings — wer den Ausweis-Designer nur zum Drucken
	// öffnet (view_students reicht dafür), bekäme sonst ein sichtbares Berechtigungs-Toast
	// für eine reine Komfortfunktion. Deshalb roh über apiFetch und bei jedem Fehler
	// (auch 403) still nichts tun.
	async function heileSchulstammdaten() {
		try {
			const res = await apiFetch('/api/einstellungen');
			if (!res.ok) return;
			const data = await res.json();
			const adresse = [
				data.schule_strasse,
				[data.schule_plz, data.schule_ort].filter(Boolean).join(' ')
			]
				.filter(Boolean)
				.join(', ');
			wendeSchulstammdatenAn(data.schule_name ?? '', adresse);
		} catch {
			/* Komfortfunktion — Platzhalter bleibt stehen */
		}
	}

	return {
		get zustand() {
			return zustand;
		},
		get geladen() {
			return geladen;
		},
		get ladefehler() {
			return ladefehler;
		},

		/** Lädt das zentral gespeicherte Design. Leeres {} (Erststart) → Vorgabewerte. */
		async laden() {
			try {
				const res = await apiFetch('/api/ausweis-layout');
				if (!res.ok) {
					ladefehler = `Das gespeicherte Ausweis-Design konnte nicht geladen werden (Fehler ${res.status}). Es wird nichts gespeichert, solange das so ist.`;
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
			try {
				const res = await apiFetch('/api/ausweis-layout', {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body
				});
				zustand = res.ok ? 'saved' : 'error';
			} catch {
				zustand = 'error';
			}
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
			const ok = window.confirm(
				'Ausweis-Design auf die Standardwerte zurücksetzen?\n\n' +
					'Alle eigenen Anpassungen an Vorder- und Rückseite gehen verloren — ' +
					'auch für die anderen Arbeitsplätze, da das Design zentral gespeichert wird.'
			);
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
			const ok = window.confirm(
				`Design-Vorlage „${name}" anwenden?\n\n` +
					'Vorder- und Rückseite werden ersetzt; eigene Anpassungen gehen verloren — ' +
					'auch für die anderen Arbeitsplätze, da das Design zentral gespeichert wird.'
			);
			if (!ok) return false;
			if (!wendeVorlageAn(kennung)) return false;
			await heileSchulstammdaten();
			return true;
		}
	};
}

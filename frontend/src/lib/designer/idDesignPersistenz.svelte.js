import { apiFetch } from '../apiFetch.js';
import { applyDesign, resetDesign, wendeSchulstammdatenAn } from './idDesignerStore.svelte.js';

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

/** @returns {{ readonly zustand: 'idle'|'saving'|'saved'|'error', readonly geladen: boolean, laden: () => Promise<void>, speichern: (body: string) => Promise<void>, beginneSpeichern: () => void, zuruecksetzen: () => Promise<boolean> }} */
export function erzeugeDesignAblage() {
	/** @type {'idle'|'saving'|'saved'|'error'} */
	let zustand = $state('idle');
	// Erst nach dem initialen Laden auto-speichern, sonst überschrieben die
	// Store-Vorgabewerte den geladenen Stand.
	let geladen = $state(false);

	// /api/einstellungen verlangt manage_users — wer den Ausweis-Designer nur zum Drucken
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

		/** Lädt das zentral gespeicherte Design. Leeres {} (Erststart) → Vorgabewerte. */
		async laden() {
			try {
				const res = await apiFetch('/api/ausweis-layout');
				if (res.ok) applyDesign(await res.json());
			} catch (e) {
				console.error('Ausweis-Design konnte nicht geladen werden:', e);
			} finally {
				geladen = true;
			}
			// NACH applyDesign(): Sonst überschreibt das geladene Design (auch eines, das
			// den Platzhalter noch trägt) die geheilten Werte sofort wieder.
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
		}
	};
}

import { snackbars } from '../../stores/snackbars.svelte.js';
import { ladeAbgeschlosseneInventuren, loescheAbgeschlosseneInventur } from './inventurApi.js';
import { fehlertext } from '../../utils/fehlertext.js';

/**
 * Die früheren Inventuren unter „Keine Inventur aktiv": zehn je Seite vom Server, „Mehr laden"
 * holt die nächsten, und ein Lauf lässt sich mit seinem Fehlbestandsbericht löschen. Die nächste
 * Seite schließt am letzten gezeigten Lauf an (naechste_seite); endet an einem anderen
 * Arbeitsplatz ein Lauf, rückt keine Zeile doppelt nach.
 * @param {{ frueher: string }} nichtGeladen bekommt den Grund, wenn die erste Seite nicht kam
 * @param {(laufId: string) => void} [nachLoeschen] erfährt, welcher Lauf gelöscht ist
 */
export function useFruehereInventuren(nichtGeladen, nachLoeschen) {
	let laeufe = $state.raw(/** @type {any[]} */ ([]));
	let weitere = $state(0);
	let laedtMehr = $state(false);
	let naechsteSeite = '';

	/** Die erste Seite, neu vom Server: nach dem Öffnen, nach Abschluss und nach Verwerfen. */
	async function lade() {
		try {
			const seite = await ladeAbgeschlosseneInventuren();
			laeufe = seite.laeufe;
			weitere = seite.weitere;
			naechsteSeite = seite.naechste_seite;
			nichtGeladen.frueher = '';
		} catch (e) {
			nichtGeladen.frueher = fehlertext(e);
		}
	}

	async function ladeMehr() {
		if (!naechsteSeite || laedtMehr) return;
		laedtMehr = true;
		try {
			const seite = await ladeAbgeschlosseneInventuren(naechsteSeite);
			laeufe = [...laeufe, ...seite.laeufe];
			weitere = seite.weitere;
			naechsteSeite = seite.naechste_seite;
		} catch (e) {
			snackbars.addToast(fehlertext(e), 'error');
		} finally {
			laedtMehr = false;
		}
	}

	/**
	 * Löscht den Lauf am Server und nimmt ihn aus der Liste. Kennt der Server ihn nicht mehr
	 * (404, an einem anderen Arbeitsplatz gelöscht), verschwindet die Zeile ebenfalls.
	 * @param {{ session_id: string, label: string }} inventur
	 */
	async function loesche(inventur) {
		const r = await loescheAbgeschlosseneInventur(inventur.session_id);
		if (!r.ok && r.status !== 404) {
			snackbars.addToast(r.error || 'Die Inventur konnte nicht gelöscht werden.', 'error');
			return;
		}
		laeufe = laeufe.filter((l) => l.session_id !== inventur.session_id);
		nachLoeschen?.(inventur.session_id);
		if (r.ok) snackbars.addToast(`Inventur „${inventur.label}“ gelöscht.`, 'success');
		else snackbars.addToast(r.error || 'Die Inventur war schon gelöscht.', 'warning');
	}

	return /** @satisfies {object} */ ({
		get laeufe() {
			return laeufe;
		},
		get weitere() {
			return weitere;
		},
		get laedtMehr() {
			return laedtMehr;
		},
		lade,
		ladeMehr,
		loesche
	});
}

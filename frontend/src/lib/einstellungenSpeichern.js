// Das Speichern EINER Einstellungs-Kategorie.
//
// Der Rumpf trägt ausschließlich die Felder dieser Kategorie; alles andere lässt der
// Server unangetastet (repository/system_settings_patch.go). Genau darauf beruht die
// Zusage der Oberfläche: Wer in „Bestellwesen" speichert, ändert nichts an den
// Löschfristen — vorher tat er das, weil ein einziger Knopf am Seitenende immer das
// ganze Formular schickte.
//
// Zahlenfelder gehen NICHT ungeprüft raus: Ein leer geräumtes Feld ist keine 0 und
// auch kein „lass es wie es war", sondern ein unvollständiges Formular. Es wird
// gemeldet, und zwar mit der Beschriftung des Feldes.
import { apiPut } from './apiFetch.js';
import { toastStore } from './stores/toastStore.svelte.js';
import { sammleZahlen } from './settingsWerte.js';

/**
 * @param {object} eingabe
 * @param {Record<string, string|boolean|null>} [eingabe.felder] Text- und Schalterfelder.
 * @param {{ schluessel: string, label: string, wert: unknown }[]} [eingabe.zahlen] Zahlenfelder.
 * @param {() => void | Promise<void>} [eingabe.onSaved] Neu laden nach dem Speichern.
 * @returns {Promise<boolean>} true, wenn gespeichert wurde.
 */
export async function speichereKategorie({ felder = {}, zahlen = [], onSaved } = {}) {
	const { werte, fehlend } = sammleZahlen(zahlen);
	if (fehlend.length > 0) {
		toastStore.addToast(
			`Bitte eine Zahl eintragen bei: ${fehlend.join(', ')}. Nichts wurde gespeichert.`,
			'warning'
		);
		return false;
	}

	try {
		await apiPut('/api/einstellungen', { ...felder, ...werte });
		toastStore.addToast('Gespeichert.', 'success');
		if (onSaved) await onSaved();
		return true;
	} catch {
		// Die Fehlermeldung kommt bereits aus apiPut.
		return false;
	}
}

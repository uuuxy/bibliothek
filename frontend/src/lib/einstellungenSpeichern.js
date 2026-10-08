// Das Speichern EINER Einstellungs-Kategorie.
//
// Der Rumpf trägt ausschließlich die Felder dieser Kategorie; alles andere lässt der
// Server unangetastet (repository/system_settings_patch.go). Genau darauf beruht die
// Zusage der Oberfläche: Wer in „Bestellwesen" speichert, ändert nichts an den
// Löschfristen — vorher tat er das, weil ein einziger Knopf am Seitenende immer das
// ganze Formular schickte.
//
// Zahlenfelder gehen NICHT ungeprüft raus: Ein leer geräumtes Feld ist keine 0 und
// auch kein „lass es wie es war", sondern ein unvollständiges Formular. Dasselbe gilt
// für einen Wert unterhalb der Feldgrenze — das Backend ersetzt ihn sonst still durch
// die Vorgabe. Gemeldet wird mit der Beschriftung des Feldes.
//
// Von den Feldern der Kategorie geht nur mit, was vom Stand beim Öffnen abweicht: Zwei
// Plätze, die dieselbe Kategorie offen haben, überschrieben sich sonst gegenseitig Felder,
// die keiner von beiden angefasst hat.
import { apiPut } from './apiFetch.js';
import { toastStore } from './stores/toastStore.svelte.js';
import { sammleZahlen } from './settingsWerte.js';
import { nurGeaendertes } from './utils/geaendert.js';

/**
 * @param {object} eingabe
 * @param {Record<string, string|number|boolean|null>} eingabe.geladen Die Werte der Felder beim Öffnen, unter den Namen der Anfrage.
 * @param {Record<string, string|boolean|null>} [eingabe.felder] Text- und Schalterfelder.
 * @param {{ schluessel: string, label: string, wert: unknown, min?: number }[]} [eingabe.zahlen] Zahlenfelder mit ihrer Untergrenze.
 * @param {() => void | Promise<void>} [eingabe.onSaved] Neu laden nach dem Speichern.
 * @returns {Promise<boolean>} true, wenn gespeichert wurde.
 */
export async function speichereKategorie({ geladen, felder = {}, zahlen = [], onSaved }) {
	const { werte, fehlend } = sammleZahlen(zahlen);
	if (fehlend.length > 0) {
		toastStore.addToast(
			`Bitte eine gültige Zahl eintragen bei: ${fehlend.join(', ')}. Nichts wurde gespeichert.`,
			'warning'
		);
		return false;
	}

	const rumpf = nurGeaendertes(geladen, { ...felder, ...werte });
	try {
		// Ohne Änderung gibt es nichts zu schicken; der Server wiese den leeren Rumpf ab.
		if (Object.keys(rumpf).length > 0) await apiPut('/api/einstellungen', rumpf);
		toastStore.addToast('Gespeichert.', 'success');
		if (onSaved) await onSaved();
		return true;
	} catch {
		// Die Fehlermeldung kommt bereits aus apiPut.
		return false;
	}
}

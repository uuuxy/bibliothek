import { snackbars } from '../../stores/snackbars.svelte.js';
import {
	ladeAbgeschlosseneInventuren,
	ladeFehlbestand,
	meldeVerlustGefunden,
	loescheVerlustEndgueltig
} from './inventurApi.js';
import { formatEuro } from '../../utils/format.js';
import { fehlertext } from '../../utils/fehlertext.js';

/**
 * Der Fehlbestandsbericht: Liste, frühere Läufe, und die zwei Handlungen darauf
 * (Gefunden / endgültig löschen). Aus useInventur herausgelöst — die
 * Session-Scan-Mechanik und der Bericht über eine ABGESCHLOSSENE Inventur sind zwei
 * verschiedene Zuständigkeiten, die nur zufällig denselben Bildschirm teilen.
 * @param {{ frueher: string }} nichtGeladen bekommt den Grund, wenn die früheren nicht kamen
 */
export function useFehlbestand(nichtGeladen) {
	/** Fehlbestand der zuletzt abgeschlossenen (oder nachträglich aufgerufenen)
	 *  Inventur — bleibt stehen, bis er ausdruecklich geschlossen wird. */
	let fehlbestand = $state.raw(/** @type {any[]} */ ([]));
	let fehlbestandBezeichnung = $state('');

	// Frühere Inventuren: Der Bericht entstand bisher NUR aus der Antwort von
	// schliesseAb() und lebte damit im Arbeitsspeicher dieses einen Browsers. Ein
	// Neuladen — oder der Kollege am zweiten Arbeitsplatz, der ins Regal geht — sah ihn
	// nie. Der Server hat die Liste dauerhaft (inventur_verluste); hier ist der Weg
	// zurück zu ihr.
	let abgeschlosseneInventuren = $state.raw(/** @type {any[]} */ ([]));
	let ladeFruehereLaeuft = $state(false);

	/** Setzt den Bericht neu — Aufrufer ist der Session-Abschluss.
	 * @param {any[]} liste @param {string} label */
	function setzeFehlbestand(liste, label) {
		fehlbestand = liste;
		fehlbestandBezeichnung = label;
	}

	function fehlbestandSchliessen() {
		fehlbestand = [];
		fehlbestandBezeichnung = '';
	}

	/** Ein Buch wurde beim Regal-Absuchen doch gefunden — zurück in Umlauf, im Bericht
	 *  als geklärt markiert statt aus der Liste zu verschwinden (der Bericht dokumentiert
	 *  auch, DASS es gefunden wurde, nicht nur das Ergebnis).
	 * @param {string} exemplarId */
	async function fehlbestandGefunden(exemplarId) {
		const r = await meldeVerlustGefunden(exemplarId);
		if (!r.ok) {
			snackbars.addToast(r.error || 'Fund konnte nicht verbucht werden.', 'error');
			return;
		}
		// Nur ein Wahrheitswert nötig — die Karte zeigt "Gefunden" statt Datum an, ein
		// exaktes ISO-Datum wäre hier ungenutzt. Date.now() statt `new Date(...)`, damit
		// keine mutable Date-Instanz in diesem $state.raw-Baum landet.
		fehlbestand = fehlbestand.map((e) =>
			e.exemplar_id === exemplarId ? { ...e, gefunden_am: Date.now() } : e
		);
		// Der Fund beendet auch die Forderung, die das Buch abgerechnet hatte (dieselbe Regel
		// wie an der Theke). Liegt der Bescheid dazu schon bei
		// der Aufsicht, bleibt sie offen — dann sagt der Server, wer zu informieren ist.
		const storniert = Number(r.data?.stornierte_forderungen) || 0;
		snackbars.addToast(
			storniert > 0
				? `Als gefunden verbucht — Exemplar ist wieder verfügbar. Die Forderung über ${formatEuro(r.data.stornierter_betrag)} wurde storniert.`
				: 'Als gefunden verbucht — Exemplar ist wieder verfügbar.',
			'success'
		);
		if (r.data?.hinweis) snackbars.addToast(r.data.hinweis, 'warning');
	}

	/** @param {string[]} exemplarIds */
	async function fehlbestandEndgueltigLoeschen(exemplarIds) {
		const r = await loescheVerlustEndgueltig(exemplarIds);
		if (!r.ok) {
			snackbars.addToast(r.error || 'Löschen fehlgeschlagen.', 'error');
			return;
		}
		// Entfernt wird, was der Server WIRKLICH gelöscht hat — nicht, was angefragt
		// wurde. Vorher filterte diese Zeile über `exemplarIds`: Wer fünf auswählte, von
		// denen zwei inzwischen als „gefunden" markiert oder gar nicht als Verlust
		// gebucht waren, sah alle fünf verschwinden. Zwei davon nur auf seinem
		// Bildschirm; nach dem nächsten Laden waren sie wieder da.
		// Plain Array statt Set: die Liste bleibt zweistellig, ein .includes() ist hier
		// klarer als eine weitere Set-Instanz im reaktiven Zustand zu rechtfertigen.
		const geloeschteIds = r.data.geloeschte_ids ?? [];
		fehlbestand = fehlbestand.filter((e) => !geloeschteIds.includes(e.exemplar_id));

		// Und die Meldung sagt, was geschah — nicht, was verlangt wurde. „0 Exemplare
		// endgültig gelöscht" als grüne Erfolgsmeldung war die Bauform, an der dieses
		// Projekt schon einmal eine falsche Sortier-Bilanz hatte.
		const uebersprungen = exemplarIds.length - geloeschteIds.length;
		if (geloeschteIds.length === 0) {
			snackbars.addToast(
				'Nichts gelöscht — keines der ausgewählten Exemplare ist (noch) als Verlust gebucht.',
				'warning'
			);
		} else if (uebersprungen > 0) {
			snackbars.addToast(
				`${geloeschteIds.length} endgültig gelöscht, ${uebersprungen} übersprungen (nicht mehr als Verlust gebucht).`,
				'warning'
			);
		} else {
			snackbars.addToast(`${geloeschteIds.length} Exemplare endgültig gelöscht.`, 'success');
		}
	}

	async function ladeFruehereInventuren() {
		try {
			abgeschlosseneInventuren = await ladeAbgeschlosseneInventuren();
			nichtGeladen.frueher = '';
		} catch (e) {
			nichtGeladen.frueher = fehlertext(e);
		}
	}

	/** @param {{session_id: string, label: string, abgeschlossen_am: string}} inventur */
	async function zeigeFrueherenFehlbestand(inventur) {
		ladeFruehereLaeuft = true;
		try {
			const r = await ladeFehlbestand(inventur.session_id);
			if (!r.ok) {
				snackbars.addToast(r.error || 'Fehlbestand konnte nicht geladen werden.', 'error');
				return;
			}
			fehlbestand = r.data ?? [];
			fehlbestandBezeichnung = inventur.label;
			if (fehlbestand.length === 0) {
				snackbars.addToast('Diese Inventur war vollständig — kein Fehlbestand.', 'success');
			}
		} finally {
			ladeFruehereLaeuft = false;
		}
	}

	return /** @satisfies {object} */ ({
		get fehlbestand() {
			return fehlbestand;
		},
		get fehlbestandBezeichnung() {
			return fehlbestandBezeichnung;
		},
		setzeFehlbestand,
		fehlbestandSchliessen,
		fehlbestandGefunden,
		fehlbestandEndgueltigLoeschen,
		get abgeschlosseneInventuren() {
			return abgeschlosseneInventuren;
		},
		get ladeFruehereLaeuft() {
			return ladeFruehereLaeuft;
		},
		ladeFruehereInventuren,
		zeigeFrueherenFehlbestand
	});
}

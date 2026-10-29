import { snackbars } from '../../stores/snackbars.svelte.js';
import { ladeFehlbestand, meldeVerlustGefunden, loescheVerlustEndgueltig } from './inventurApi.js';
import { formatEuro } from '../../utils/format.js';

/**
 * Der Fehlbestandsbericht: Liste und die zwei Handlungen darauf (Gefunden / endgültig
 * löschen). Aus useInventur herausgelöst — die Session-Scan-Mechanik und der Bericht über eine
 * ABGESCHLOSSENE Inventur sind zwei verschiedene Zuständigkeiten, die nur zufällig denselben
 * Bildschirm teilen. Die Liste der früheren Läufe führt useFruehereInventuren.
 */
export function useFehlbestand() {
	/** Fehlbestand der zuletzt abgeschlossenen (oder nachträglich aufgerufenen)
	 *  Inventur — bleibt stehen, bis er ausdruecklich geschlossen wird. */
	let fehlbestand = $state.raw(/** @type {any[]} */ ([]));
	let fehlbestandBezeichnung = $state('');
	// Der Lauf, zu dem der Bericht gehört: Wird er gelöscht, schließt sich der Bericht.
	let fehlbestandLauf = '';
	let ladeFruehereLaeuft = $state(false);

	/** Setzt den Bericht neu — Aufrufer ist der Session-Abschluss.
	 * @param {any[]} liste @param {string} label @param {string} laufId */
	function setzeFehlbestand(liste, label, laufId) {
		fehlbestand = liste;
		fehlbestandBezeichnung = label;
		fehlbestandLauf = laufId;
	}

	function fehlbestandSchliessen() {
		fehlbestand = [];
		fehlbestandBezeichnung = '';
		fehlbestandLauf = '';
	}

	/** @param {string} laufId der gelöschte Lauf */
	function schliesseBerichtVon(laufId) {
		if (laufId === fehlbestandLauf) fehlbestandSchliessen();
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

	/** @param {{session_id: string, label: string, abgeschlossen_am: string}} inventur */
	async function zeigeFrueherenFehlbestand(inventur) {
		ladeFruehereLaeuft = true;
		try {
			const r = await ladeFehlbestand(inventur.session_id);
			if (!r.ok) {
				snackbars.addToast(r.error || 'Fehlbestand konnte nicht geladen werden.', 'error');
				return;
			}
			setzeFehlbestand(r.data ?? [], inventur.label, inventur.session_id);
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
		schliesseBerichtVon,
		get ladeFruehereLaeuft() {
			return ladeFruehereLaeuft;
		},
		zeigeFrueherenFehlbestand
	});
}

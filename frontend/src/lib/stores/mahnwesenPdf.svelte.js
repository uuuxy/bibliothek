import { apiFetch, leseApiFehler } from '../apiFetch.js';
import { fehlertext } from '../utils/fehlertext.js';
import { snackbars } from './snackbars.svelte.js';
import { zeigePdf, PDF_BLOCKIERT } from '../utils/pdfImFenster.js';

/** Der Druck der Mahnbriefe aus der Auswahl. */
export function useMahnwesenPdf() {
	let drucktPdf = $state(false);

	/**
	 * Druckt die Mahnbriefe der angehakten Schüler; der Server zählt dabei die Mahnung.
	 * @param {Set<string>} gewaehlteKennungen
	 * @param {Function} getFilteredSchueler
	 * @param {Function} refreshData
	 */
	async function druckeGewaehlteMahnungen(gewaehlteKennungen, getFilteredSchueler, refreshData) {
		if (gewaehlteKennungen.size === 0) return;

		drucktPdf = true;
		try {
			const aktuelleListe = getFilteredSchueler();
			const ausleihIds = [];
			for (const schuelerId of gewaehlteKennungen) {
				const s = aktuelleListe.find(/** @type {any} */ (x) => x.schueler_id === schuelerId);
				if (s?.medien) {
					for (const m of s.medien) {
						if (m.ausleihe_id) ausleihIds.push(m.ausleihe_id);
					}
				}
			}

			if (ausleihIds.length === 0) {
				snackbars.addToast(
					'Keine überfälligen Medien für die ausgewählten Schüler gefunden.',
					'info'
				);
				return;
			}

			// Der Druck zählt die Mahnung. Blockiert der Browser den Tab, geht keine Anfrage hinaus.
			const res = await zeigePdf(() =>
				apiFetch('/api/admin/mahnungen/bulk-print', {
					method: 'POST',
					body: JSON.stringify({ ausleih_ids: ausleihIds })
				})
			);
			if (!res) {
				snackbars.addToast(PDF_BLOCKIERT, 'warning');
				return;
			}
			if (!res.ok)
				throw new Error(await leseApiFehler(res, 'Die Mahnbriefe ließen sich nicht erzeugen.'));

			gewaehlteKennungen.clear();
			await refreshData();
		} catch (e) {
			snackbars.addToast(fehlertext(e), 'error');
		} finally {
			drucktPdf = false;
		}
	}

	return /** @satisfies {object} */ ({
		get drucktPdf() {
			return drucktPdf;
		},
		druckeGewaehlteMahnungen
	});
}

import { apiFetch } from '../apiFetch.js';
import { toastStore } from './toastStore.svelte.js';
import { SvelteDate } from 'svelte/reactivity';

/** Der Druck der Mahnbriefe aus der Auswahl. */
export function useMahnwesenPdf() {
	let pdfLoading = $state(false);

	/**
	 * Druckt die Mahnbriefe der angehakten Schüler; der Server zählt dabei die Mahnung.
	 * @param {Set<string>} selectedIds
	 * @param {Function} getFilteredSchueler
	 * @param {Function} refreshData
	 */
	async function printSelectedMahnungen(selectedIds, getFilteredSchueler, refreshData) {
		if (selectedIds.size === 0) return;

		pdfLoading = true;
		try {
			const currentList = getFilteredSchueler();
			const ausleihIds = [];
			for (const schuelerId of selectedIds) {
				const s = currentList.find(/** @type {any} */ (x) => x.schueler_id === schuelerId);
				if (s?.medien) {
					for (const m of s.medien) {
						if (m.ausleihe_id) ausleihIds.push(m.ausleihe_id);
					}
				}
			}

			if (ausleihIds.length === 0) {
				toastStore.addToast(
					'Keine überfälligen Medien für die ausgewählten Schüler gefunden.',
					'info'
				);
				return;
			}

			const res = await apiFetch('/api/admin/mahnungen/bulk-print', {
				method: 'POST',
				body: JSON.stringify({ ausleih_ids: ausleihIds })
			});

			if (!res.ok)
				throw new Error(
					'Bulk-PDF-Erzeugung fehlgeschlagen: ' + ((await res.text()) || res.statusText)
				);

			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `mahnbriefe_${new SvelteDate().toISOString().slice(0, 10)}.pdf`;
			a.click();
			URL.revokeObjectURL(url);

			selectedIds.clear();
			await refreshData();
		} catch (e) {
			toastStore.addToast('Fehler: ' + String(e), 'error');
		} finally {
			pdfLoading = false;
		}
	}

	return {
		get pdfLoading() {
			return pdfLoading;
		},
		printSelectedMahnungen
	};
}

import { apiFetch } from '../apiFetch.js';
import { SvelteDate } from 'svelte/reactivity';

/**
 * Handles PDF generation logic for Mahnwesen.
 */
export function useMahnwesenPdf() {
	let pdfLoading = $state(false);
	let elternPdfLoading = $state(false);
	let klassePdfLoading = $state(false);
	let globalErrorToast = $state(/** @type {string|null} */ (null));

	/**
	 * Prints selected Mahnungen by collecting ausleih_ids.
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
				alert('Keine überfälligen Medien für die ausgewählten Schüler gefunden.');
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
			a.download = `Mahnliste_Bulk_${new SvelteDate().toISOString().slice(0, 10)}.pdf`;
			a.click();
			URL.revokeObjectURL(url);

			selectedIds.clear();
			await refreshData();
		} catch (e) {
			alert('Fehler: ' + String(e));
		} finally {
			pdfLoading = false;
		}
	}

	/**
	 * Downloads the global Mahnliste PDF.
	 */
	async function downloadPDF() {
		pdfLoading = true;
		try {
			const res = await apiFetch('/api/mahnwesen/pdf');
			if (!res.ok) throw new Error('PDF-Erzeugung fehlgeschlagen');
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `mahnliste_${new SvelteDate().toISOString().slice(0, 10)}.pdf`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			alert('Fehler: ' + String(e));
		} finally {
			pdfLoading = false;
		}
	}

	/**
	 * Downloads the Eltern Mahnbriefe PDF.
	 */
	async function downloadElternPDF() {
		elternPdfLoading = true;
		globalErrorToast = null;
		try {
			const res = await apiFetch('/api/reports/overdue-pdf');
			if (!res.ok) throw new Error((await res.text()) || 'PDF-Erzeugung fehlgeschlagen');
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `mahnbriefe_${new SvelteDate().toISOString().slice(0, 10)}.pdf`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			globalErrorToast = 'Fehler: ' + String(e);
			setTimeout(() => (globalErrorToast = null), 4000);
		} finally {
			elternPdfLoading = false;
		}
	}

	/**
	 * Downloads the PDF for a specific class.
	 * @param {string} selectedKlasse
	 */
	async function downloadKlassePDF(selectedKlasse) {
		if (!selectedKlasse) return;
		klassePdfLoading = true;
		globalErrorToast = null;
		try {
			const res = await apiFetch(`/api/print/mahnung/klasse/${selectedKlasse}`);
			if (!res.ok) {
				const errText = await res.text();
				throw new Error(errText || 'Keine überfälligen Ausleihen gefunden');
			}
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `Mahnliste_Klasse_${selectedKlasse}.pdf`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			globalErrorToast = String(e);
			setTimeout(() => (globalErrorToast = null), 4000);
		} finally {
			klassePdfLoading = false;
		}
	}

	return {
		get pdfLoading() {
			return pdfLoading;
		},
		get elternPdfLoading() {
			return elternPdfLoading;
		},
		get klassePdfLoading() {
			return klassePdfLoading;
		},
		get globalErrorToast() {
			return globalErrorToast;
		},
		set globalErrorToast(v) {
			globalErrorToast = v;
		},
		printSelectedMahnungen,
		downloadPDF,
		downloadElternPDF,
		downloadKlassePDF
	};
}

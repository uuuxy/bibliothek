<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { bestaetigen, loeschenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
	import { appState, showToast } from '$lib/store.svelte.js';
	import { loescheTitel, coverNeuHolen } from '../../admin_api.js';
	import {
		speichereBuch,
		stehtInSicht,
		DubletteFehler,
		frageVorhandenenOeffnen
	} from '../../buch_speichern.js';
	import { hatRecht } from '../../../../lib/menu.js';
	import { authStore } from '../../../../lib/stores/authStore.svelte.js';

	let { books = $bindable(), isEditMode = $bindable(), formular = $bindable() } = $props();

	export function darfLoeschen() {
		return hatRecht(authStore.currentUser, 'delete_books');
	}

	/** Löschen aus der Maske: ein Titel, Rückfrage, der Server verweigert bei Ausleihen. */
	export async function titelLoeschen() {
		if (!formular.id) return;
		if (!(await loeschenBestaetigen(`„${formular.title}“ mit allen Exemplaren löschen?`))) return;
		try {
			await loescheTitel(formular.id);
			books = books.filter((/** @type {any} */ b) => b.id !== formular.id);
			isEditMode = false;
			showToast('Titel gelöscht.', 'success');
		} catch (fehler) {
			showToast(fehler instanceof Error ? fehler.message : String(fehler), 'error');
		}
	}

	export async function saveChanges() {
		if (!formular.title || !formular.isbn) {
			showToast('Titel und ISBN sind Pflichtfelder', 'error');
			return;
		}

		if (formular.id) {
			const originalBook = books.find((/** @type {any} */ b) => b.id === formular.id);
			if (originalBook && Number(formular.stock) < Number(originalBook.stock)) {
				const proceed = await bestaetigen({
					titel: 'Gesamtbestand verringern?',
					text: 'Die entsprechende Anzahl an Exemplaren wird im Hintergrund als verloren markiert.',
					aktion: 'Verringern',
					gefaehrlich: true
				});
				if (!proceed) return;
			}
		}

		try {
			const neu = !formular.id;
			const updated = await speichereBuch(formular);
			// Die Antwort auf das Ändern trägt den Bestand nicht, die Maske schon.
			const bestand = Number(neu ? updated.stock : formular.stock) || 0;
			if (books.some((/** @type {any} */ b) => b.id === updated.id)) {
				books = books.map((/** @type {any} */ b) => (b.id === updated.id ? updated : b));
			} else if (stehtInSicht(bestand, appState.bestandsAnsicht)) {
				books = [{ ...updated, stock: bestand }, ...books];
			}

			// Sync with appState so Omnibox/Catalog update immediately
			if (appState.selectedBook && appState.selectedBook.id === updated.id) {
				appState.selectedBook = updated;
			}

			if (isEditMode) {
				isEditMode = false;
			}
			showToast(
				neu && bestand === 0
					? 'Titel ohne Exemplar gespeichert. Er steht unter „Ohne Exemplare“.'
					: 'Buch erfolgreich gespeichert!',
				'success'
			);
		} catch (e) {
			// Neue Maske: Statt nur abzulehnen, führt sie zum Titel, der die ISBN schon trägt.
			if (e instanceof DubletteFehler && !formular.id) {
				await frageVorhandenenOeffnen(e.message, e.vorhanden);
				return;
			}
			showToast(e instanceof Error ? e.message : String(e), 'error');
		}
	}

	/** @param {File} file */
	async function compressImageToWebp(file) {
		return new Promise((resolve, reject) => {
			const img = new Image();
			img.onload = () => {
				URL.revokeObjectURL(img.src);
				const canvas = document.createElement('canvas');
				let width = img.width;
				let height = img.height;
				const MAX_WIDTH = 600;
				const MAX_HEIGHT = 900;

				if (width > MAX_WIDTH || height > MAX_HEIGHT) {
					const ratio = Math.min(MAX_WIDTH / width, MAX_HEIGHT / height);
					width = Math.round(width * ratio);
					height = Math.round(height * ratio);
				}

				canvas.width = width;
				canvas.height = height;
				const ctx = canvas.getContext('2d');
				if (ctx) ctx.drawImage(img, 0, 0, width, height);

				canvas.toBlob(
					(blob) => {
						if (!blob) reject(new Error('Compression failed'));
						else
							resolve(
								new File([blob], file.name.replace(/\.[^/.]+$/, '.webp'), { type: 'image/webp' })
							);
					},
					'image/webp',
					0.82
				);
			};
			img.onerror = () => reject(new Error('Invalid image'));
			img.src = URL.createObjectURL(file);
		});
	}

	/** „Cover neu holen": nur das Bild, Titel und Autor bleiben (Server-Regel). */
	export async function handleCoverNeuHolen() {
		if (!formular.id) return;
		try {
			const coverUrl = await coverNeuHolen(formular.id);
			formular.coverUrl = coverUrl;
			books = books.map((/** @type {any} */ b) => (b.id === formular.id ? { ...b, coverUrl } : b));
			showToast('Cover neu geholt', 'success');
		} catch (err) {
			showToast(err instanceof Error ? err.message : String(err), 'error');
		}
	}

	/** @param {Event} e */
	export async function handleCoverUpload(e) {
		const target = /** @type {HTMLInputElement} */ (e.target);
		let file = target.files ? target.files[0] : null;
		if (!file || !formular.id) return;

		try {
			if (file.type.startsWith('image/')) {
				file = await compressImageToWebp(file);
			}
		} catch (err) {
			console.error('WebP compression failed, using original file:', err);
		}

		const fd = new FormData();
		fd.append('cover', /** @type {File} */ (file));
		try {
			const res = await apiFetch(`/api/books/${formular.id}/cover-upload`, {
				method: 'POST',
				credentials: 'include',
				headers: {},
				body: fd
			});
			if (!res.ok) {
				let message = 'Upload fehlgeschlagen';
				try {
					const errorJson = await res.json();
					if (errorJson?.message) {
						message = errorJson.message;
					} else if (errorJson?.error) {
						message = errorJson.error;
					}
				} catch {
					// fallback to default message
				}
				throw new Error(message);
			}
			const json = await res.json();
			formular.coverUrl = json.data.coverUrl;
			books = books.map((/** @type {any} */ b) =>
				b.id === formular.id ? { ...b, coverUrl: json.data.coverUrl } : b
			);
			showToast('Cover erfolgreich hochgeladen', 'success');
		} catch (err) {
			showToast(err instanceof Error ? err.message : String(err), 'error');
		}
	}
</script>

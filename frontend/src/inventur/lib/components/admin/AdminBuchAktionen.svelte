<script>
	import { apiFetch, leseApiFehler } from '../../../../lib/apiFetch.js';
	import { bestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
	import { appState, melde } from '$lib/store.svelte.js';
	import { coverNeuHolen } from '../../admin_api.js';
	import {
		loescheTitelMitRueckfrage,
		meldungZumErgebnis
	} from '../../../../lib/components/buchakte/titelLoeschen.js';
	import {
		stehtInSicht,
		verringernRueckfrage,
		BestandVeraltetFehler,
		DubletteFehler
	} from '../../buch_speichern.js';
	import { frageVorhandenenOeffnen, speichereMitFrage } from '../../buch_vorhanden.js';
	import { uebernimmGespeichert } from '../../buch_felder.js';
	import { hatRecht } from '../../../../lib/menue.js';
	import { authStore } from '../../../../lib/stores/authStore.svelte.js';
	import { fehlertext } from '../../../../lib/utils/fehlertext.js';
	import { verkleinereBild } from '../../../../lib/utils/bildVerkleinern.js';

	let { titelListe = $bindable(), istBearbeitung = $bindable(), formular = $bindable() } = $props();

	export function darfLoeschen() {
		return hatRecht(authStore.currentUser, 'delete_books');
	}

	/** Löschen aus der Maske: derselbe Dialog wie in der Buchakte (titelLoeschen.js). */
	export async function titelLoeschen() {
		if (!formular.id) return;
		try {
			const ergebnis = await loescheTitelMitRueckfrage([formular.id], formular.title);
			if (!ergebnis) return;
			titelListe = titelListe.filter((/** @type {any} */ b) => !ergebnis.geloescht.includes(b.id));
			istBearbeitung = false;
			melde(meldungZumErgebnis(ergebnis), 'success');
		} catch (fehler) {
			melde(fehlertext(fehler), 'error');
		}
	}

	// Ein zweiter Klick, solange das Speichern läuft, schickt nichts: Seine Anfrage träfe den
	// Stand der ersten und käme als Ablehnung zurück.
	let speichert = false;

	/** Speichert die Maske. Pflicht ist der Titel, das prüft der Knopf (BuchFormular). */
	export async function speichereAenderungen() {
		if (speichert) return;
		speichert = true;
		try {
			const frage = verringernRueckfrage(formular);
			if (frage && !(await bestaetigen(frage))) return;

			const neu = !formular.id;
			const updated = await speichereMitFrage(formular);
			// Die Frage nach dem gleichnamigen Titel führte zu ihm oder blieb ohne Antwort.
			if (!updated) return;
			// Die Antwort ist der gespeicherte Titel samt Bestand; die Zeile der Liste wird sie.
			const bestand = Number(updated.stock) || 0;
			if (titelListe.some((/** @type {any} */ b) => b.id === updated.id)) {
				titelListe = titelListe.map((/** @type {any} */ b) => (b.id === updated.id ? updated : b));
			} else if (stehtInSicht(bestand, appState.bestandsAnsicht)) {
				titelListe = [updated, ...titelListe];
			}

			// Sync with appState so Omnibox/Catalog update immediately
			if (appState.selectedBook && appState.selectedBook.id === updated.id) {
				appState.selectedBook = updated;
			}

			if (istBearbeitung) {
				istBearbeitung = false;
			}
			melde(
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
			// Ein anderer Platz hat den Bestand geändert: Das Feld zeigt den neuen Stand, die
			// übrigen Eingaben der Maske bleiben stehen.
			if (e instanceof BestandVeraltetFehler) {
				formular.stock = formular.stockGesehen = e.bestand;
			}
			melde(fehlertext(e), 'error');
		} finally {
			speichert = false;
		}
	}

	// Ein Cover wird höchstens 600 × 900 Pixel groß gespeichert.
	/** @param {File} file */
	async function compressImageToWebp(file) {
		const blob = await verkleinereBild(file, { maxBreite: 600, maxHoehe: 900, guete: 0.82 });
		return new File([blob], file.name.replace(/\.[^/.]+$/, '.webp'), { type: 'image/webp' });
	}

	/** „Cover neu holen": nur das Bild, Titel und Autor bleiben (Server-Regel). */
	export async function holeCoverNeu() {
		if (!formular.id) return;
		try {
			const coverUrl = await coverNeuHolen(formular.id);
			uebernimmGespeichert(formular, 'coverUrl', coverUrl);
			titelListe = titelListe.map((/** @type {any} */ b) =>
				b.id === formular.id ? { ...b, coverUrl } : b
			);
			melde('Cover neu geholt', 'success');
		} catch (err) {
			melde(fehlertext(err), 'error');
		}
	}

	/** @param {Event} e */
	export async function ladeCoverHoch(e) {
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
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Upload fehlgeschlagen'));
			const json = await res.json();
			uebernimmGespeichert(formular, 'coverUrl', json.coverUrl);
			titelListe = titelListe.map((/** @type {any} */ b) =>
				b.id === formular.id ? { ...b, coverUrl: json.coverUrl } : b
			);
			melde('Cover erfolgreich hochgeladen', 'success');
		} catch (err) {
			melde(fehlertext(err), 'error');
		}
	}
</script>

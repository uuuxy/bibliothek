<!--
  admin/+page.svelte
  Hauptseite des Administratorenbereichs: liest/schreibt Bücherdaten über die API, steuert Unterkomponenten.
-->
<script>
	import { onMount } from 'svelte';
	import { bestaetigen, loeschenBestaetigen } from '../../../lib/stores/bestaetigung.svelte.js';
	import { appState, showToast } from '$lib/store.svelte.js';
	import BookTable from '$lib/components/admin/BookTable.svelte';
	import BuchFormular from '$lib/components/admin/BuchFormular.svelte';
	import AdminBuchAktionen from '$lib/components/admin/AdminBuchAktionen.svelte';
	import ClassAssignPicker from '$lib/components/admin/ClassAssignPicker.svelte';
	import { leeresBuchFormular } from '$lib/components/admin/buch_form_optionen.js';
	import {
		holeBuecherListe,
		holeBuchDetail,
		loescheBuecher,
		holeExterneCover,
		retryExterneCover
	} from '$lib/admin_api.js';

	/** @type {any[]} */
	let buecher = $state.raw([]);
	let wirdGeladen = $state(false);
	let istBearbeitenModus = $state(false);
	let wirdGescannt = $state(false);
	let buchAktionen = $state();
	/** @type {string[]|null} Bücher-IDs, die gerade einer Klasse zugewiesen werden (Picker offen). */
	let klassenZuweisenIds = $state(null);

	let formular = $state(leeresBuchFormular());

	/** @type {any} */
	let suchVerzoegerung = null;
	$effect(() => {
		const suchAnfrage = appState.searchQuery;
		void appState.bestandsAnsicht; // die Sicht lädt die Liste genauso neu wie die Suche
		if (suchVerzoegerung) clearTimeout(suchVerzoegerung);
		suchVerzoegerung = setTimeout(() => {
			if (appState.adminAuthenticated && typeof suchAnfrage === 'string') {
				aktualisiereBuecher();
			}
		}, 300);
	});

	onMount(() => {
		aktualisiereBuecher();
	});

	$effect(() => {
		if (appState.bookToEdit && !wirdGeladen) {
			const found = buecher.find((b) => b.id === appState.bookToEdit.id);
			if (found) {
				oeffneDetails(found);
			} else {
				oeffneDetails(appState.bookToEdit);
			}
			appState.bookToEdit = null;
		}
	});

	async function aktualisiereBuecher() {
		wirdGeladen = true;
		try {
			const geladene = await holeBuecherListe();
			buecher = geladene;
			appState.adminAuthenticated = true;
		} catch {
			appState.adminAuthenticated = false;
		} finally {
			wirdGeladen = false;
		}
	}

	function neuesBuchErstellen() {
		formular = leeresBuchFormular();
		istBearbeitenModus = true;
	}

	/** Der Knopf „Scanner": dieselbe Maske wie „Neues Buch", die Kamera ist schon an. */
	function neuesBuchScannen() {
		neuesBuchErstellen();
		wirdGescannt = true;
	}

	/** @param {any} buch */
	async function oeffneDetails(buch) {
		// Immer das VOLLE Buch vom Einzel-Read laden: Die Katalogliste ist bewusst
		// schlank (beschreibung/erweiterteEigenschaften leer), und saveChanges schickt
		// das ganze Formular per PUT zurück — aus dem Listen-Objekt gespreadet würde
		// Speichern genau diese Felder still leeren (Upsert-Blanking-Bugklasse).
		// Nebeneffekt: Bearbeiten arbeitet auf frischen Daten statt einer evtl.
		// veralteten Listenzeile. Bei Ladefehler wird NICHT mit dem schlanken Objekt
		// geöffnet — das wäre derselbe stille Datenverlust durch die Hintertür.
		let voll = buch;
		if (buch?.id) {
			try {
				voll = await holeBuchDetail(buch.id);
			} catch {
				showToast('Buch konnte nicht vollständig geladen werden — Bearbeiten abgebrochen', 'error');
				return;
			}
		}
		// stockGesehen: Mit der Zahl vom Öffnen erkennt das Speichern, ob das Feld „Bestand"
		// geändert wurde und ob sie am Server noch gilt (buch_speichern.js).
		formular = { ...voll, stockGesehen: voll.stock };
		if (!formular.medientyp) {
			formular.medientyp = 'Buch';
		}
		if (formular.lastCounted && formular.lastCounted.includes('T')) {
			formular.lastCounted = formular.lastCounted.split('T')[0];
		}
		istBearbeitenModus = true;
	}

	/** @param {any} ids */
	async function aktionBuecherLoeschen(ids) {
		if (!ids.length) return;
		if (!(await loeschenBestaetigen(`${ids.length} Bücher mit allen Exemplaren löschen?`))) return;
		try {
			await loescheBuecher(ids);
			buecher = buecher.filter((b) => !ids.includes(b.id));
		} catch (fehler) {
			showToast(/** @type {any} */ (fehler).message, 'error');
		}
	}

	async function aktionExterneCoverRetry() {
		try {
			const externe = await holeExterneCover();
			if (!externe.length) {
				showToast('Keine externen Cover mehr vorhanden.', 'info');
				return;
			}
			if (
				!(await bestaetigen({
					titel: `${externe.length} externe Cover erneut lokalisieren?`,
					aktion: 'Lokalisieren'
				}))
			)
				return;

			const ids = externe.map((/** @type {any} */ b) => b.id);
			const ergebnis = await retryExterneCover(ids);
			await aktualisiereBuecher();
			showToast(
				`Cover-Retry fertig. Aktualisiert: ${ergebnis.updated}, Übersprungen: ${ergebnis.skipped}, Fehler: ${ergebnis.failed}`,
				'info'
			);
		} catch (fehler) {
			showToast(/** @type {any} */ (fehler).message, 'error');
		}
	}
</script>

<div class="relative min-h-[calc(100vh-8rem)]">
	{#if istBearbeitenModus}
		<BuchFormular
			bind:formular
			bind:wirdGescannt
			onClose={() => (istBearbeitenModus = false)}
			onSave={() => buchAktionen.saveChanges()}
			onCoverUpload={(/** @type {any} */ ereignis) => buchAktionen.handleCoverUpload(ereignis)}
			onCoverNeuHolen={() => buchAktionen.handleCoverNeuHolen()}
			onAssignClass={() => (klassenZuweisenIds = formular.id ? [formular.id] : [])}
			onDelete={buchAktionen?.darfLoeschen() ? () => buchAktionen.titelLoeschen() : undefined}
		/>
	{:else}
		<BookTable
			books={buecher}
			loading={wirdGeladen}
			onOpenDetail={oeffneDetails}
			onCreateNew={neuesBuchErstellen}
			onScan={neuesBuchScannen}
			onDelete={aktionBuecherLoeschen}
			onAssignClass={(ids) => (klassenZuweisenIds = ids)}
			onRetryCovers={aktionExterneCoverRetry}
		/>
	{/if}

	<AdminBuchAktionen
		bind:this={buchAktionen}
		bind:books={buecher}
		bind:isEditMode={istBearbeitenModus}
		bind:formular
	/>

	{#if klassenZuweisenIds && klassenZuweisenIds.length > 0}
		<ClassAssignPicker
			bookIds={klassenZuweisenIds}
			onClose={() => (klassenZuweisenIds = null)}
			onAssigned={() => (klassenZuweisenIds = null)}
		/>
	{/if}
</div>

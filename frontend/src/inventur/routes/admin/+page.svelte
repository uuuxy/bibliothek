<!--
  admin/+page.svelte
  Hauptseite für den Administratorenbereich der Inventur-App.
  Liest und schreibt Bücherdaten mithilfe der API und steuert Unterkomponenten.
-->
<script>
	import { onMount } from 'svelte';
	import { appState, showToast } from '$lib/store.svelte.js';
	import BookTable from '$lib/components/admin/BookTable.svelte';
	import BuchFormular from '$lib/components/admin/BuchFormular.svelte';
	import StrichcodeScanner from '$lib/components/StrichcodeScanner.svelte';
	import AdminBuchAktionen from '$lib/components/admin/AdminBuchAktionen.svelte';
	import ClassAssignPicker from '$lib/components/admin/ClassAssignPicker.svelte';
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

	let formular = $state({
		id: null,
		isbn: '',
		title: '',
		author: '',
		subject: 'Mathe',
		gradeLevel: 5,
		track: 'Gymnasium',
		stock: 0,
		coverUrl: '',
		lastCounted: '',
		medientyp: 'Buch'
	});

	/** @type {any} */
	let suchVerzoegerung = null;
	$effect(() => {
		const suchAnfrage = appState.searchQuery;
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
		formular = {
			id: null,
			isbn: '',
			title: '',
			author: '',
			subject: 'Mathe',
			gradeLevel: 5,
			track: 'Gymnasium',
			stock: 0,
			coverUrl: '',
			lastCounted: '',
			medientyp: 'Buch'
		};
		istBearbeitenModus = true;
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
		formular = { ...voll };
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
		if (!ids.length || !confirm(`${ids.length} Bücher wirklich löschen?`)) return;
		try {
			await loescheBuecher(ids);
			buecher = buecher.filter((b) => !ids.includes(b.id));
		} catch (fehler) {
			alert(/** @type {any} */ (fehler).message);
		}
	}

	async function aktionExterneCoverRetry() {
		try {
			const externe = await holeExterneCover();
			if (!externe.length) {
				alert('Keine externen Cover mehr vorhanden.');
				return;
			}
			if (!confirm(`${externe.length} externe Cover jetzt erneut lokalisieren?`)) {
				return;
			}

			const ids = externe.map((/** @type {any} */ b) => b.id);
			const ergebnis = await retryExterneCover(ids);
			await aktualisiereBuecher();
			alert(
				`Cover-Retry fertig. Aktualisiert: ${ergebnis.updated}, Übersprungen: ${ergebnis.skipped}, Fehler: ${ergebnis.failed}`
			);
		} catch (fehler) {
			alert(/** @type {any} */ (fehler).message);
		}
	}

	function nachScanAktion() {
		aktualisiereBuecher();
	}
</script>

<div class="relative min-h-[calc(100vh-8rem)]">
	{#if wirdGescannt}
		<div
			class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
		>
			<StrichcodeScanner onClose={() => (wirdGescannt = false)} onCreated={nachScanAktion} />
		</div>
	{:else if istBearbeitenModus}
		<BuchFormular
			bind:formular
			onClose={() => (istBearbeitenModus = false)}
			onSave={() => buchAktionen.saveChanges()}
			onCoverUpload={(/** @type {any} */ ereignis) => buchAktionen.handleCoverUpload(ereignis)}
			onAssignClass={() => (klassenZuweisenIds = formular.id ? [formular.id] : [])}
		/>
	{:else}
		<BookTable
			books={buecher}
			loading={wirdGeladen}
			onOpenDetail={oeffneDetails}
			onCreateNew={neuesBuchErstellen}
			onScan={() => (wirdGescannt = true)}
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

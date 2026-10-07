<!-- Titel-Verwaltung: lädt die Titelliste, öffnet die Maske und steuert die Unterkomponenten. -->
<script>
	import { onMount, untrack } from 'svelte';
	import { bestaetigen, loeschenBestaetigen } from '../../../lib/stores/bestaetigung.svelte.js';
	import Ladekreis from '../../../lib/components/ui/Ladekreis.svelte';
	import { appState, showToast } from '$lib/store.svelte.js';
	import BookTable from '$lib/components/admin/BookTable.svelte';
	import BuchFormular from '$lib/components/admin/BuchFormular.svelte';
	import AdminBuchAktionen from '$lib/components/admin/AdminBuchAktionen.svelte';
	import ClassAssignPicker from '$lib/components/admin/ClassAssignPicker.svelte';
	import { leeresBuchFormular } from '$lib/components/admin/buch_form_optionen.js';
	import { loescheBuecher, holeExterneCover, retryExterneCover } from '$lib/admin_api.js';
	import { titelFuerMaske } from '$lib/buch_speichern.js';
	import { erstelleTitelListe } from '$lib/titelListe.svelte.js';

	// Laden und Ändern der Liste: titelListe.svelte.js.
	const liste = erstelleTitelListe();
	const aktualisiereBuecher = liste.lade;
	let istBearbeitenModus = $state(false);
	let wirdGescannt = $state(false);
	let buchAktionen = $state();
	/** @type {string[]|null} Bücher-IDs, die gerade einer Klasse zugewiesen werden (Picker offen). */
	let klassenZuweisenIds = $state(null);

	let formular = $state(leeresBuchFormular());

	// Die Liste lädt beim Aufbau und nach jedem Wechsel von Suche oder Sicht. Der erste Lauf
	// des Effekts ist kein Wechsel; mit ihm lüde die ganze Liste zweimal.
	let ersterLauf = true;
	$effect(() => {
		const suchAnfrage = appState.searchQuery;
		void appState.bestandsAnsicht; // die Sicht lädt die Liste genauso neu wie die Suche
		if (ersterLauf) {
			ersterLauf = false;
			return;
		}
		const warten = setTimeout(() => {
			if (appState.adminAuthenticated && typeof suchAnfrage === 'string') aktualisiereBuecher();
		}, 300);
		return () => clearTimeout(warten);
	});

	onMount(aktualisiereBuecher);

	// Ein Titel, mit dem jemand zum Bearbeiten kommt, wird sofort geholt: Die Maske braucht
	// nur ihn, nicht die Liste. Solange er unterwegs ist, steht die Tabelle nicht da.
	let titelKommt = $state(false);
	$effect(() => {
		const ziel = appState.bookToEdit;
		if (!ziel) return;
		appState.bookToEdit = null;
		untrack(async () => {
			titelKommt = true;
			await oeffneDetails(ziel);
			titelKommt = false;
		});
	});

	// Nur die jüngste Öffnung gilt: Die Antwort zu einem früher angeklickten Titel legte sich
	// sonst über die Maske des späteren, samt dem dort Getippten.
	let oeffnung = 0;

	function neuesBuchErstellen() {
		oeffnung++;
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
		const meine = ++oeffnung;
		let geholt;
		try {
			geholt = await titelFuerMaske(buch);
		} catch {
			if (meine !== oeffnung) return;
			showToast('Buch konnte nicht vollständig geladen werden — Bearbeiten abgebrochen', 'error');
			return;
		}
		if (meine !== oeffnung) return;
		formular = geholt;
		istBearbeitenModus = true;
	}

	/** @param {any} ids */
	async function aktionBuecherLoeschen(ids) {
		if (!ids.length) return;
		if (!(await loeschenBestaetigen(`${ids.length} Bücher mit allen Exemplaren löschen?`))) return;
		try {
			await loescheBuecher(ids);
			liste.buecher = liste.buecher.filter((b) => !ids.includes(b.id));
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
				`Cover lokalisiert. Aktualisiert: ${ergebnis.updated}, Übersprungen: ${ergebnis.skipped}, Fehler: ${ergebnis.failed}`,
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
	{:else if titelKommt}
		<!-- Unter 200 ms zeigt Material 3 keine Ladeanzeige; sie blendet erst danach ein. -->
		<div class="flex justify-center py-32 transition-opacity delay-200 starting:opacity-0">
			<Ladekreis size="lg" />
		</div>
	{:else}
		<BookTable
			books={liste.buecher}
			loading={liste.wirdGeladen}
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
		bind:books={liste.buecher}
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

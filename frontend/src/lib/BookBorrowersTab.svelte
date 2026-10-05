<script>
	import Button from './components/ui/Button.svelte';
	import { toastStore } from './stores/toastStore.svelte.js';
	import Select from './components/ui/Select.svelte';
	import BorrowersListe from './components/BorrowersListe.svelte';
	import { baueAusleiherDruckHtml } from './utils/ausleiherDruck.js';
	import { druckeDokument, FENSTER_BLOCKIERT } from './utils/listenDruck.js';
	import { fmtDateDE as fmtDate } from './utils/dates.js';
	import { Printer, Users } from '@lucide/svelte';
	import Suchfeld from './components/ui/Suchfeld.svelte';

	/** @type {{ borrowers: any[], book: any, onBack: () => void }} */
	let { borrowers, book, onBack } = $props();

	let filterKlasse = $state('Alle');
	let filterName = $state('');

	let availableKlassen = $derived([
		'Alle',
		...Array.from(new Set(borrowers.map((b) => b.klasse || 'Unbekannt'))).sort()
	]);

	let filteredBorrowers = $derived(
		borrowers.filter((b) => {
			const matchKlasse = filterKlasse === 'Alle' || (b.klasse || 'Unbekannt') === filterKlasse;
			const matchName =
				filterName === '' ||
				`${b.schueler_name} ${b.schueler_nachname}`
					.toLowerCase()
					.includes(filterName.toLowerCase());
			return matchKlasse && matchName;
		})
	);

	function printAusleiher() {
		const html = baueAusleiherDruckHtml(filteredBorrowers, book, filterKlasse);
		if (!druckeDokument(html)) toastStore.addToast(FENSTER_BLOCKIERT, 'warning');
	}
</script>

{#if borrowers.length === 0}
	<div class="py-16 flex flex-col items-center text-on-surface-variant gap-3">
		<Users class="w-10 h-10" aria-hidden="true" />
		<p class="font-semibold text-sm">Aktuell niemand hat dieses Buch ausgeliehen.</p>
	</div>
{:else}
	<!-- Filters -->
	<div class="flex gap-3 mb-4">
		<Select
			bind:value={filterKlasse}
			options={availableKlassen.map((/** @type {string} */ k) => ({ value: k, label: k }))}
			class="w-44"
			aria-label="Nach Klasse filtern"
		/>
		<Suchfeld
			bind:wert={filterName}
			platzhalter="Name eingeben …"
			etikett="Nach Name filtern"
			klasse="flex-1 max-w-xs"
		/>
		{#if filteredBorrowers.length !== borrowers.length}
			<span class="text-xs text-on-surface-variant self-center"
				>{filteredBorrowers.length} von {borrowers.length}</span
			>
		{/if}
		<div class="flex-1"></div>
		<Button variant="secondary" onclick={printAusleiher}>
			<Printer class="w-4 h-4" aria-hidden="true" />
			Liste drucken
		</Button>
	</div>

	<BorrowersListe zeilen={filteredBorrowers} {onBack} {fmtDate} />
{/if}

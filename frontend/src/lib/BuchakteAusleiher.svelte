<script>
	import Button from './components/ui/Button.svelte';
	import { snackbars } from './stores/snackbars.svelte.js';
	import Select from './components/ui/Select.svelte';
	import AusleiherListe from './components/AusleiherListe.svelte';
	import { baueAusleiherDruckHtml } from './utils/ausleiherDruck.js';
	import { druckeDokument, FENSTER_BLOCKIERT } from './utils/listenDruck.js';
	import { fmtDateDE as fmtDate } from './utils/datum.js';
	import { Printer, Users } from '@lucide/svelte';
	import Suchfeld from './components/ui/Suchfeld.svelte';

	/** @type {{ ausleiher: any[], titel: any, onBack: () => void }} */
	let { ausleiher, titel, onBack } = $props();

	let filterKlasse = $state('Alle');
	let filterName = $state('');

	let availableKlassen = $derived([
		'Alle',
		...Array.from(new Set(ausleiher.map((b) => b.klasse || 'Unbekannt'))).sort()
	]);

	let gefilterteAusleiher = $derived(
		ausleiher.filter((b) => {
			const matchKlasse = filterKlasse === 'Alle' || (b.klasse || 'Unbekannt') === filterKlasse;
			const matchName =
				filterName === '' ||
				`${b.schueler_name} ${b.schueler_nachname}`
					.toLowerCase()
					.includes(filterName.toLowerCase());
			return matchKlasse && matchName;
		})
	);

	function druckeAusleiher() {
		const html = baueAusleiherDruckHtml(gefilterteAusleiher, titel, filterKlasse);
		if (!druckeDokument(html)) snackbars.addToast(FENSTER_BLOCKIERT, 'warning');
	}
</script>

{#if ausleiher.length === 0}
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
		{#if gefilterteAusleiher.length !== ausleiher.length}
			<span class="text-xs text-on-surface-variant self-center"
				>{gefilterteAusleiher.length} von {ausleiher.length}</span
			>
		{/if}
		<div class="flex-1"></div>
		<Button variant="secondary" onclick={druckeAusleiher}>
			<Printer class="w-4 h-4" aria-hidden="true" />
			Liste drucken
		</Button>
	</div>

	<AusleiherListe zeilen={gefilterteAusleiher} {onBack} {fmtDate} />
{/if}

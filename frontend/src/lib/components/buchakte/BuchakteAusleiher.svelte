<script>
	import Button from '../ui/Button.svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Select from '../ui/Select.svelte';
	import AusleiherListe from './AusleiherListe.svelte';
	import { baueAusleiherDruckHtml } from '../../utils/ausleiherDruck.js';
	import { druckeDokument, FENSTER_BLOCKIERT } from '../../utils/listenDruck.js';
	import { formatDatum } from '../../utils/format.js';
	import Icon from '../ui/Icon.svelte';
	import Suchfeld from '../ui/Suchfeld.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import { passtZurSuche } from '../../utils/suchnorm.js';

	/** @type {{ ausleiher: any[], titel: any, onBack: () => void }} */
	let { ausleiher, titel, onBack } = $props();

	let filterKlasse = $state('Alle');
	let filterName = $state('');

	let availableKlassen = $derived([
		'Alle',
		...Array.from(new Set(ausleiher.map((b) => b.klasse || 'Unbekannt'))).sort((a, b) =>
			a.localeCompare(b, 'de', { numeric: true })
		)
	]);

	let gefilterteAusleiher = $derived(
		ausleiher.filter((b) => {
			const matchKlasse = filterKlasse === 'Alle' || (b.klasse || 'Unbekannt') === filterKlasse;
			// Dieselbe Regel wie an der Theke: „ozturk" findet Öztürk, „kahil mia" Mia Al-Kahil.
			return matchKlasse && passtZurSuche(filterName, b.schueler_name, b.schueler_nachname);
		})
	);

	function druckeAusleiher() {
		const html = baueAusleiherDruckHtml(gefilterteAusleiher, titel, filterKlasse);
		if (!druckeDokument(html)) snackbars.addToast(FENSTER_BLOCKIERT, 'warning');
	}
</script>

{#if ausleiher.length === 0}
	<SuchZustand
		symbol="group"
		titel="Keine Ausleihen"
		hinweis="Kein Exemplar dieses Titels ist ausgeliehen"
	/>
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
			<Icon name="print" />
			Liste drucken
		</Button>
	</div>

	<AusleiherListe zeilen={gefilterteAusleiher} {onBack} fmtDate={formatDatum} />
{/if}

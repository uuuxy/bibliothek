<!-- @component MahnwesenSuchleiste — Freitextsuche + Klassenfilter unter den Registern.
     Beides greift zusammen mit dem aktiven Register und sortiert die Liste nach
     Dringlichkeit. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Select from '../ui/Select.svelte';
	import Suchfeld from '../ui/Suchfeld.svelte';
</script>

<div class="flex items-center gap-3 mt-4 print:hidden">
	<Suchfeld
		bind:wert={mahnwesenStore.searchQuery}
		platzhalter="Schüler oder Klasse suchen …"
		etikett="Schüler oder Klasse suchen"
		klasse="flex-1 max-w-sm"
	/>
	<Select
		bind:value={mahnwesenStore.selectedKlasse}
		options={[
			{ value: '', label: 'Alle Klassen' },
			...mahnwesenStore.klassen.map((/** @type {any} */ k) => ({
				value: k.klasse,
				label: k.klasse
			}))
		]}
		class="w-40"
		aria-label="Nach Klasse filtern"
	/>
</div>

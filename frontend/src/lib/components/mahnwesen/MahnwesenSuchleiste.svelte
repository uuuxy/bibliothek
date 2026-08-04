<!-- @component MahnwesenSuchleiste — Freitextsuche + Klassenfilter unter den Registern.
     Beides greift zusammen mit dem aktiven Register und sortiert die Liste nach
     Dringlichkeit. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Select from '../ui/Select.svelte';
</script>

<div class="flex items-center gap-3 mt-4 print:hidden">
	<div class="relative flex-1 max-w-sm">
		<svg
			class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400 pointer-events-none"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			stroke-width="2"
			aria-hidden="true"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				d="M21 21l-4.35-4.35M17 11a6 6 0 11-12 0 6 6 0 0112 0z"
			/>
		</svg>
		<input
			type="search"
			bind:value={mahnwesenStore.searchQuery}
			placeholder="Schüler oder Klasse suchen …"
			aria-label="Schüler oder Klasse suchen"
			class="w-full pl-9 pr-3 py-2 rounded-xl border border-slate-200 bg-white text-sm text-slate-800 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all"
		/>
	</div>
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

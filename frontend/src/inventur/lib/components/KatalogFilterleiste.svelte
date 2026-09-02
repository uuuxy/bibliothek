<!-- @component KatalogFilterleiste — Filter, Sortierung, Trefferzahl und Ansichtswahl
     der Buch-Suche im Medienkatalog.

     Der Reiter heißt „Suche & Filter", hatte in der Buch-Suche bis zum 02.09.2026 aber
     keinen Filter (Zweig/Jahrgang nur im Reiter „Jahrgänge"), keine Trefferzahl (die
     stand erst nach 50 Treffern im „Mehr laden"-Knopf) und keine Sortierung. Alle
     Optionslisten entstehen aus den geladenen Büchern; der Filter ist ein Objekt aus
     leererFilter(), das hier direkt verändert wird (tief reaktiv).

     Neues Markup auf den M3-Rollen — die Farb-Ratsche lässt keine neuen
     Paletten-Fundstellen zu. -->
<script>
	import Select from '../../../lib/components/ui/Select.svelte';
	import Button from '../../../lib/components/ui/Button.svelte';
	import { LayoutGrid, List, X } from '@lucide/svelte';
	import { SORTIERUNGEN, BESTAND_FILTER, leererFilter } from '../startseiten_api.js';

	/**
	 * @type {{
	 *   filter: ReturnType<typeof leererFilter>,
	 *   sortierung: string,
	 *   ansicht: 'karten'|'liste',
	 *   fachOptionen: { value: string, label: string }[],
	 *   jahrgangOptionen: { value: string, label: string }[],
	 *   zweigOptionen: { value: string, label: string }[],
	 *   medientypOptionen: { value: string, label: string }[],
	 *   treffer: number,
	 *   gesamt: number
	 * }}
	 */
	let {
		filter,
		sortierung = $bindable(''),
		ansicht = $bindable('karten'),
		fachOptionen = [],
		jahrgangOptionen = [],
		zweigOptionen = [],
		medientypOptionen = [],
		treffer = 0,
		gesamt = 0
	} = $props();

	const aktiv = $derived(Object.values(filter).some(Boolean));

	function zuruecksetzen() {
		Object.assign(filter, leererFilter());
	}

	/** @type {{ value: 'karten'|'liste', label: string, Symbol: any }[]} */
	const ANSICHTEN = [
		{ value: 'karten', label: 'Karten', Symbol: LayoutGrid },
		{ value: 'liste', label: 'Liste', Symbol: List }
	];
</script>

<div class="space-y-3" id="filter-suche">
	<div class="flex flex-wrap items-center gap-2">
		<Select
			bind:value={filter.fach}
			options={fachOptionen}
			class="w-44"
			aria-label="Fach filtern"
		/>
		<Select
			bind:value={filter.jahrgang}
			options={jahrgangOptionen}
			class="w-40"
			aria-label="Jahrgang filtern"
		/>
		<Select
			bind:value={filter.zweig}
			options={zweigOptionen}
			class="w-44"
			aria-label="Schulzweig filtern"
		/>
		<!-- Nur anbieten, wenn es mehr als eine Medienart gibt — ein Dropdown mit einer
		     einzigen Wahl ist Möblierung. -->
		{#if medientypOptionen.length > 2}
			<Select
				bind:value={filter.medientyp}
				options={medientypOptionen}
				class="w-36"
				aria-label="Medienart filtern"
			/>
		{/if}
		<Select
			bind:value={filter.bestand}
			options={BESTAND_FILTER}
			class="w-40"
			aria-label="Bestand filtern"
		/>
		<Select bind:value={sortierung} options={SORTIERUNGEN} class="w-48" aria-label="Sortierung" />
		{#if aktiv}
			<Button variant="ghost" size="sm" onclick={zuruecksetzen}>
				<X class="h-4 w-4" aria-hidden="true" />
				Filter zurücksetzen
			</Button>
		{/if}
	</div>

	<div class="flex items-center justify-between gap-3">
		<!-- aria-live: Wer filtert, hört die neue Trefferzahl, ohne sie suchen zu müssen. -->
		<span class="text-sm text-on-surface-variant" aria-live="polite">
			{#if treffer === gesamt}
				{gesamt} Titel
			{:else}
				{treffer} von {gesamt} Titeln
			{/if}
		</span>

		<!-- Segmentierter Schalter (M3) wie in der Statistik, auf den Rollen statt slate. -->
		<div
			class="flex items-center rounded-full border border-outline-variant bg-surface-container p-1"
			role="group"
			aria-label="Ansicht"
		>
			{#each ANSICHTEN as a (a.value)}
				<button
					type="button"
					onclick={() => (ansicht = a.value)}
					aria-pressed={ansicht === a.value}
					class="flex cursor-pointer items-center gap-1.5 rounded-full px-3 py-1 text-sm font-medium whitespace-nowrap transition-colors {ansicht ===
					a.value
						? 'bg-surface-container-lowest text-on-surface shadow-xs'
						: 'text-on-surface-variant hover:text-on-surface'}"
				>
					<a.Symbol class="h-4 w-4" aria-hidden="true" />
					{a.label}
				</button>
			{/each}
		</div>
	</div>
</div>

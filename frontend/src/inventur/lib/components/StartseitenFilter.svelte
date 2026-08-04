<!--
	StartseitenFilter.svelte
	Filtert die Startseite nach Buch-Suche, Jahrgängen oder Schulklassen.
	Refactored: Clean SaaS light-mode design with Google-style tabs.
-->
<script>
	import Select from '../../../lib/components/ui/Select.svelte';
	import KlassenSuchfeld from './KlassenSuchfeld.svelte';

	/**
	 * @type {{
	 *   viewMode: string,
	 *   searchQuery: string,
	 *   selectedZweig: string,
	 *   selectedJahrgang: string,
	 *   klasseSearchQuery: string,
	 *   isKlasseDropdownOpen: boolean,
	 *   filteredKlassenList: string[],
	 *   onSelectKlasse: (klasse: string) => void
	 * }}
	 */
	let {
		viewMode = $bindable('suche'),
		searchQuery = $bindable(''),
		selectedZweig = $bindable(''),
		selectedJahrgang = $bindable(''),
		klasseSearchQuery = $bindable(''),
		isKlasseDropdownOpen = $bindable(false),
		filteredKlassenList = [],
		onSelectKlasse
	} = $props();

	const zweigOptionen = [
		{ value: '', label: 'Alle Zweige' },
		...['Gymnasium', 'Realschule', 'Hauptschule'].map((z) => ({ value: z, label: z }))
	];
	const jahrgangOptionen = [
		{ value: '', label: 'Alle Jahrgänge' },
		...['5', '6', '7', '8', '9', '10', '11', '12', '13'].map((j) => ({
			value: j,
			label: `Klasse ${j}`
		}))
	];
</script>

<header class="pt-6 pb-6 px-4 sm:px-6 lg:px-8">
	<div class="max-w-5xl mx-auto flex flex-col items-center space-y-6">
		<!-- Google-style underline Tabs -->
		<div
			class="border-b border-slate-200 w-full max-w-md"
			role="tablist"
			aria-label="Ansichtsmodus"
		>
			<nav class="flex gap-6 justify-center">
				<button
					class="relative pb-2.5 text-sm font-semibold transition-colors cursor-pointer {viewMode ===
					'suche'
						? 'text-blue-600'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => (viewMode = 'suche')}
					role="tab"
					id="tab-suche"
					aria-selected={viewMode === 'suche'}
					aria-controls="filter-suche content-suche"
				>
					Buch-Suche
					{#if viewMode === 'suche'}
						<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
					{/if}
				</button>
				<button
					class="relative pb-2.5 text-sm font-semibold transition-colors cursor-pointer {viewMode ===
					'jahrgaenge'
						? 'text-blue-600'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => (viewMode = 'jahrgaenge')}
					role="tab"
					id="tab-jahrgaenge"
					aria-selected={viewMode === 'jahrgaenge'}
					aria-controls="filter-jahrgaenge content-jahrgaenge"
				>
					Jahrgänge
					{#if viewMode === 'jahrgaenge'}
						<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
					{/if}
				</button>
				<button
					class="relative pb-2.5 text-sm font-semibold transition-colors cursor-pointer {viewMode ===
					'schulklassen'
						? 'text-blue-600'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => (viewMode = 'schulklassen')}
					role="tab"
					id="tab-schulklassen"
					aria-selected={viewMode === 'schulklassen'}
					aria-controls="filter-schulklassen content-schulklassen"
				>
					Schulklassen
					{#if viewMode === 'schulklassen'}
						<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
					{/if}
				</button>
			</nav>
		</div>

		<!-- Dynamic Filter Area -->
		<div class="w-full max-w-3xl transition-all duration-300 ease-in-out">
			{#if viewMode === 'suche'}
				<!-- Material-3-Suchleiste: weiche Pille mit Flächen-Fokus. Bewusst NICHT auf der
				     36-px-Control-Skala und bewusst rounded-full — dieses Feld ist kein Datenfeld
				     im Formular, sondern das globale Werkzeug der Seite und soll sich davon
				     abheben. Der Container trägt Rahmen, Fläche und Fokus; das Feld selbst nichts. -->
				<div
					class="group flex items-center w-full h-12 px-5 bg-slate-100 rounded-full border border-transparent transition-all duration-200 focus-within:bg-white focus-within:shadow-md focus-within:border-blue-600 focus-within:ring-1 focus-within:ring-blue-600"
					id="filter-suche"
				>
					<svg
						class="h-5 w-5 shrink-0 text-slate-500 group-focus-within:text-blue-600 transition-colors duration-200"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
						/>
					</svg>
					<input
						id="katalog-suchfeld"
						type="text"
						bind:value={searchQuery}
						aria-label="Suchen nach Titel, Fach, Klasse oder Autor"
						placeholder="Suchen nach Titel, Fach, Klasse (z.B. 'Mathe 5' oder 'Gymnasium')..."
						class="h-full flex-1 bg-transparent border-none outline-none focus:ring-0 px-3 text-slate-900 placeholder:text-slate-500 text-base"
					/>
				</div>
			{:else if viewMode === 'jahrgaenge'}
				<div class="flex flex-col sm:flex-row gap-3 justify-center" id="filter-jahrgaenge">
					<Select
						bind:value={selectedZweig}
						options={zweigOptionen}
						class="w-full sm:w-56"
						aria-label="Schulzweig filtern"
					/>
					<Select
						bind:value={selectedJahrgang}
						options={jahrgangOptionen}
						class="w-full sm:w-56"
						aria-label="Jahrgang filtern"
					/>
				</div>
			{:else if viewMode === 'schulklassen'}
				<KlassenSuchfeld
					bind:klasseSearchQuery
					bind:isKlasseDropdownOpen
					{filteredKlassenList}
					{onSelectKlasse}
				/>
			{/if}
		</div>
	</div>
</header>

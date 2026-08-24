<!--
	StartseitenFilter.svelte
	Filtert die Startseite nach Buch-Suche oder Jahrgängen.
	Refactored: Clean SaaS light-mode design with Google-style tabs.
-->
<script>
	import Select from '../../../lib/components/ui/Select.svelte';
	import Suchpille from '../../../lib/components/ui/Suchpille.svelte';

	/**
	 * Die Optionslisten kommen als Props aus den geladenen Büchern
	 * (startseiten_api.js). Bis zum 24.08.2026 standen hier zwei handgepflegte
	 * Kopien: drei Zweige (das Bearbeiten-Formular kennt sechs — die Förderstufe war
	 * so nie wählbar) und Jahrgänge fix 5–13, auch wenn kein Buch sie trug.
	 * @type {{
	 *   viewMode: string,
	 *   searchQuery: string,
	 *   selectedZweig: string,
	 *   selectedJahrgang: string,
	 *   zweigOptionen: { value: string, label: string }[],
	 *   jahrgangOptionen: { value: string, label: string }[]
	 * }}
	 */
	let {
		viewMode = $bindable('suche'),
		searchQuery = $bindable(''),
		selectedZweig = $bindable(''),
		selectedJahrgang = $bindable(''),
		zweigOptionen = [],
		jahrgangOptionen = []
	} = $props();
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
				<!-- Hier stand ein dritter Reiter für die Klassensätze. Er zeigte dieselbe
				     Liste wie Verwaltung → Schulklassen, aus derselben Quelle, nur ohne
				     Aktionen — und hiess bis zum 08.08.2026 auch noch genauso. Umbenennen
				     hätte nur den Namen entschärft, nicht die Dopplung: Der Reiter ist
				     aufgelöst, die Klassensuche steht jetzt auf der Schulklassen-Seite. -->
			</nav>
		</div>

		<!-- Dynamic Filter Area -->
		<!-- Ohne `ease-in-out`: Das war die letzte Material-2-Kurve im Haus (Tailwind
		     bildet ease-in-out auf cubic-bezier(0.4, 0, 0.2, 1) ab). Die Kurve kommt
		     jetzt aus theme-mass.css. duration-300 bleibt — hier wechselt ein ganzer
		     Bereich, und dafür nennt M3 die mittlere Stufe, nicht die kurze. -->
		<div class="w-full max-w-3xl transition-all duration-300">
			{#if viewMode === 'suche'}
				<Suchpille
					id="katalog-suchfeld"
					bind:wert={searchQuery}
					platzhalter="Titel, Fach oder Klasse eingeben …"
					etikett="Suchen nach Titel, Fach, Klasse oder Autor"
				/>
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
			{/if}
		</div>
	</div>
</header>

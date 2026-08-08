<!--
	StartseitenFilter.svelte
	Filtert die Startseite nach Buch-Suche oder Jahrgängen.
	Refactored: Clean SaaS light-mode design with Google-style tabs.
-->
<script>
	import Select from '../../../lib/components/ui/Select.svelte';
	import { Search } from '@lucide/svelte';

	/**
	 * @type {{
	 *   viewMode: string,
	 *   searchQuery: string,
	 *   selectedZweig: string,
	 *   selectedJahrgang: string
	 * }}
	 */
	let {
		viewMode = $bindable('suche'),
		searchQuery = $bindable(''),
		selectedZweig = $bindable(''),
		selectedJahrgang = $bindable('')
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
				<!-- Material-3-Suchleiste: weiche Pille mit Flächen-Fokus. Bewusst NICHT auf der
				     36-px-Control-Skala und bewusst rounded-full — dieses Feld ist kein Datenfeld
				     im Formular, sondern das globale Werkzeug der Seite und soll sich davon
				     abheben. Der Container trägt Rahmen, Fläche und Fokus; das Feld selbst nichts. -->
				<div
					class="group flex items-center w-full h-12 px-5 bg-slate-100 rounded-full border border-transparent transition-all duration-200 focus-within:bg-white focus-within:shadow-md focus-within:border-blue-600 focus-within:ring-1 focus-within:ring-blue-600"
					id="filter-suche"
				>
					<Search
						class="h-5 w-5 shrink-0 text-slate-500 group-focus-within:text-blue-600 transition-colors duration-200"
						aria-hidden="true"
					/>
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
			{/if}
		</div>
	</div>
</header>

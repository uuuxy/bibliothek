<!--
	StartseitenFilter.svelte
	Filtert die Startseite nach Buch-Suche oder Jahrgängen.
-->
<script>
	import Select from '../../../lib/components/ui/Select.svelte';
	import Suchpille from '../../../lib/components/ui/Suchpille.svelte';
	import Reiter from '../../../lib/components/ui/Reiter.svelte';

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
		<!-- Sekundäre Reiter (M3 secondary tabs): zwei Sichten auf denselben Katalog,
		     unter der Primärleiste des Medienkatalogs. Der dritte Reiter „Klassensätze"
		     ist seit 08.08.2026 aufgelöst — dieselbe Liste stand schon unter
		     Bibliothek → Klassensätze. -->
		<div class="w-full max-w-md">
			<Reiter
				variante="sekundaer"
				klasse="justify-center"
				etikett="Ansichtsmodus"
				aktiv={viewMode}
				onwahl={(id) => (viewMode = id)}
				reiter={[
					{ id: 'suche', label: 'Buch-Suche', steuert: 'filter-suche content-suche' },
					{ id: 'jahrgaenge', label: 'Jahrgänge', steuert: 'filter-jahrgaenge content-jahrgaenge' }
				]}
			/>
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

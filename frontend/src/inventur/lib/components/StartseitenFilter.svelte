<!-- @component StartseitenFilter — der Kopf der Buch-Suche: eine Suchpille, sonst nichts.
     Bis 02.09.2026 standen hier zwei Reiter („Buch-Suche" / „Jahrgänge") mit eigenen
     Zweig-/Jahrgangs-Selects; der Jahrgangs-Reiter ist weg (siehe routes/+page.svelte). -->
<script>
	import Suchpille from '../../../lib/components/ui/Suchpille.svelte';

	/** @type {{ searchQuery: string }} */
	let { searchQuery = $bindable('') } = $props();
</script>

<header class="pt-6 pb-6 px-4 sm:px-6 lg:px-8">
	<div class="mx-auto w-full max-w-3xl">
		<Suchpille
			id="katalog-suchfeld"
			bind:wert={searchQuery}
			platzhalter="Titel, Fach oder Klasse eingeben …"
			etikett="Suchen nach Titel, Fach, Klasse oder Autor"
		/>
	</div>
</header>

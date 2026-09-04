<!-- @component StartseitenFilter — der Kopf der Buch-Suche: eine Suchpille, sonst nichts.
     Bis 02.09.2026 standen hier zwei Reiter („Buch-Suche" / „Jahrgänge") mit eigenen
     Zweig-/Jahrgangs-Selects; der Jahrgangs-Reiter ist weg (siehe routes/+page.svelte). -->
<script>
	import Suchpille from '../../../lib/components/ui/Suchpille.svelte';

	/** @type {{ searchQuery: string }} */
	let { searchQuery = $bindable('') } = $props();
</script>

<!-- Volle Breite und DIESELBE linke Kante wie das Kartenraster darunter (04.09.2026):
     Die Pille war auf 48 rem zentriert (x=494) und stand danach mit eigenem Innenabstand
     32 px weiter rechts als das Raster — gemessen im Browser, nicht geschätzt. -->
<!-- `pt-4`, nicht `pt-6`: Ueber der Pille steht das Reiterband des Katalogs, und der
     Abstand Reiter→Pille ist im Haus 24 px (Huelle) + 16 px. Mit pt-6 begann die Pille
     hier bei 81 px statt bei 73 — im Browser gemessen am 04.09.2026. -->
<header class="pt-4 pb-6">
	<div class="w-full">
		<Suchpille
			id="katalog-suchfeld"
			bind:wert={searchQuery}
			platzhalter="Titel, Fach oder Klasse eingeben …"
			etikett="Suchen nach Titel, Fach, Klasse oder Autor"
		/>
	</div>
</header>

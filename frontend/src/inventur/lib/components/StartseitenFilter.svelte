<!-- @component StartseitenFilter — der Kopf der Buch-Suche: eine Suchpille, sonst nichts. -->
<script>
	import Suchpille from '../../../lib/components/ui/Suchpille.svelte';

	/** @type {{ suche: string }} */
	let { suche = $bindable('') } = $props();
</script>

<!-- Volle Breite und dieselbe linke Kante wie das Kartenraster darunter, ohne eigenen
     Innenabstand: Zentriert oder eingerückt stünde die Pille gegen das Raster versetzt. -->
<!-- `pt-4`, nicht `pt-6`: Über der Pille steht das Reiterband des Katalogs, und der
     Abstand Reiter→Pille ist im Haus 24 px (Hülle) + 16 px. -->
<header class="pt-4 pb-6">
	<div class="w-full">
		<Suchpille
			kamera
			autofokus
			id="katalog-suchfeld"
			bind:wert={suche}
			platzhalter="Titel, Fach, Klasse oder Exemplarnummer …"
			etikett="Suchen nach Titel, Fach, Klasse oder Autor"
		/>
	</div>
</header>

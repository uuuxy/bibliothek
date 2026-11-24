<script>
	import BestellSuche from './BestellSuche.svelte';
	import Warenkorb from './Warenkorb.svelte';
	import Bestellbedarf from './Bestellbedarf.svelte';
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
</script>

<!-- Eine Liste: oben, was bestellt wird, darunter der Bestellbedarf, wie im Warenkorb des Google
     Store „Für später gespeichert" unter den Artikeln steht. @container: Die Zeilen brechen nach
     der Breite dieses Bereichs um, nicht nach der des Fensters. Kein Kartenrahmen; die Linie zur
     Zusammenfassung setzt Bestellwesen. -->
<div class="@container">
	<h2 class="pb-4 border-b border-outline-variant text-lg text-on-surface tracking-tight">
		Deine Bestellung
	</h2>
	<div class="pt-5 space-y-5">
		<BestellSuche />
		<Warenkorb />
		<Bestellbedarf
			vorschlaege={bestellStore.recommendations}
			onAddToCart={(/** @type {any} */ book, /** @type {number} */ menge) =>
				bestellStore.addToCart(book, menge)}
		/>
	</div>
</div>

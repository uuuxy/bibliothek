<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import MahnwesenTabs from './MahnwesenTabs.svelte';
	import MahnwesenSuchleiste from './MahnwesenSuchleiste.svelte';

	/** @type {{ onMahnlauf: () => void, onBescheid: (schuelerId: string) => void, darfBescheid: boolean }} */
	let { onMahnlauf, onBescheid, darfBescheid } = $props();
</script>

<!-- Register nach Dringlichkeit, darunter Suche, Filter und Knöpfe.

     Die Register hängen an geladenen Daten: ohne sie stünde dort „Alle 0", und das wäre
     neben der Fehlermeldung der Tabelle eine Behauptung statt einer Zahl. Suche und
     Werkzeugzeile stehen unabhängig davon — sie tragen den Rückweg „Neu laden". -->
{#if mahnwesenStore.data && !mahnwesenStore.loading}
	<MahnwesenTabs />
{/if}
<MahnwesenSuchleiste {onMahnlauf} {onBescheid} {darfBescheid} />

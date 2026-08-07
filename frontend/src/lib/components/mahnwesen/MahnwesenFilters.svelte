<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import MahnwesenTabs from './MahnwesenTabs.svelte';
	import MahnwesenSuchleiste from './MahnwesenSuchleiste.svelte';
</script>

<!-- Register nach Dringlichkeit + Filterleiste. Beide hängen an derselben Bedingung:
     ohne geladene Daten gibt es nichts zu filtern. -->
{#if mahnwesenStore.data && !mahnwesenStore.loading}
	<MahnwesenTabs />
	<MahnwesenSuchleiste />
{/if}

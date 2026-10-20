<!-- @component Hauptbereich — das <main> der angemeldeten Anwendung.

     Genau eines: Mit verschachtelten Hauptbereichen landete ein Screenreader, der „zum
     Hauptinhalt" springt, irgendwo. Öffentliche Seiten (Katalog, Monitor, Login,
     Bestellbestätigung) tragen ihr <main> selbst.

     Die Überschrift ist unsichtbar: Sichtbare Seitenköpfe gibt es nicht, die Seitenleiste
     sagt, wo man ist. Ein Screenreader liest die Seitenleiste aber nicht mit und braucht
     einen Seitentitel. Der Name kommt aus dem Menü (menue.js: bildschirmTitel), damit es
     keine zweite Liste von Bildschirmnamen gibt.

     id + tabindex: Ziel des Skip-Links (SkipLink.svelte); tabindex="-1" macht das
     Element programmatisch fokussierbar, ohne es in die Tab-Reihenfolge zu nehmen.
     Gate: e2e/barrierefreiheit-axe.spec.js (Gerüst). -->
<script>
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { bildschirmTitel } from '../../menue.js';

	/** @type {{ children: import('svelte').Snippet }} */
	let { children } = $props();
</script>

<main id="hauptinhalt" tabindex="-1" class="flex min-h-0 w-full flex-1 flex-col outline-none">
	<h1 class="sr-only">{bildschirmTitel(uiStore.activeTab)}</h1>
	{@render children()}
</main>

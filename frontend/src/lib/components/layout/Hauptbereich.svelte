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
     Gate: e2e/barrierefreiheit-axe.spec.js (Gerüst).

     Der Hauptbereich rollt und reicht seitlich in den Rand des Anwendungsrahmens (dieselben
     Maße, -mx und px heben sich auf): Ein rollender Bereich schneidet ab, und ein Symbolknopf
     am Rand des Inhalts verlöre sonst den Teil seiner Trefferfläche, der über ihn hinausragt
     (e2e/icon-trefferflaechen.spec.js). -->
<script>
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { bildschirmTitel } from '../../menue.js';

	/** @type {{ children: import('svelte').Snippet }} */
	let { children } = $props();
</script>

<main
	id="hauptinhalt"
	tabindex="-1"
	class="print-flow -mx-4 flex min-h-0 flex-1 flex-col overflow-y-auto px-4 outline-none md:-mx-8 md:px-8"
>
	<h1 class="sr-only">{bildschirmTitel(uiStore.activeTab)}</h1>
	{@render children()}
</main>

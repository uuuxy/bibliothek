<!-- @component Hauptbereich — DAS <main> der angemeldeten Anwendung.

     Genau eines: Bis zum 09.09.2026 lag ein <main> um die ganze App (samt Seitenleiste,
     Login und Overlays) und ein zweites im Router — zwei verschachtelte Hauptbereiche,
     im Medienkatalog sogar drei. Ein Screenreader, der „zum Hauptinhalt" springt,
     landete irgendwo. Öffentliche Seiten (Katalog, Monitor, Login, Bestellbestätigung)
     tragen ihr <main> selbst.

     Die Überschrift ist unsichtbar: Sichtbare Seitenköpfe sind seit a3e4184/68c4810
     bewusst abgeschafft — die Seitenleiste sagt, wo man ist. Ein Screenreader liest
     die Seitenleiste aber nicht mit; ihm fehlte jeder Seitentitel. Der Name kommt aus
     dem Menü (menu.js: tabTitel), damit es keine zweite Liste von Bildschirmnamen gibt.

     id + tabindex: Ziel des Skip-Links (SkipLink.svelte); tabindex="-1" macht das
     Element programmatisch fokussierbar, ohne es in die Tab-Reihenfolge zu nehmen.
     Gate: e2e/barrierefreiheit-axe.spec.js (Gerüst). -->
<script>
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { tabTitel } from '../../menu.js';

	/** @type {{ children: import('svelte').Snippet }} */
	let { children } = $props();
</script>

<main id="hauptinhalt" tabindex="-1" class="flex min-h-0 w-full flex-1 flex-col outline-none">
	<h1 class="sr-only">{tabTitel(uiStore.activeTab)}</h1>
	{@render children()}
</main>

<!-- @component CoverEinblendung — beim Zeigen und bei Tastaturfokus oben auf dem Cover: Fach und
     Titel in den Klassen des Platzhalters von BuchCover (PLATZHALTER), darunter weitere Zeilen und
     zuletzt die Nummer. Katalog- und Klassenkachel nehmen sie von hier. Sie deckt: Ohne Bild
     verlängert sie den Platzhalter nur, statt den Titel ein zweites Mal versetzt darüberzulegen.
     Ohne Zeigerereignisse, damit ein Klick auf sie die Kachel darunter trifft. -->
<script>
	import { PLATZHALTER } from '../../../lib/components/ui/BuchCover.svelte';

	/** @type {{ titel: string, fach?: string, zeilen?: string[], nummer?: string, nummerArt?: string }} */
	let { titel, fach = '', zeilen = [], nummer = '', nummerArt = 'ISBN' } = $props();
</script>

<!-- Sichtbar beim Zeigen auf die Kachel (group), bei Tastaturfokus auf der Kachel selbst
     (Klassenkachel) oder auf dem Knopf im Cover (group/cover, Katalog-Kachel). Fach und Titel
     sind für Hilfstechnik verborgen, sie stehen dort schon als Name. -->
<div
	class="pointer-events-none absolute inset-x-0 top-0 max-h-full overflow-hidden rounded-t-lg {PLATZHALTER.flaeche} text-left opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 group-has-[:focus-visible]/cover:opacity-100"
	data-einblendung
>
	<div aria-hidden="true">
		<p class={PLATZHALTER.fach}>{fach}</p>
		<p class={PLATZHALTER.titel}>{titel}</p>
	</div>
	{#each zeilen.filter(Boolean) as zeile, i (i)}
		<p class="mt-1.5 text-xs text-on-surface-variant">{zeile}</p>
	{/each}
	<!-- Die Nummer liest man gegen das Buch in der Hand ab, deshalb in Festbreite. Kopiert wird
	     sie in der Akte: Man braucht sie selten, die Kachel gibt ihr keinen eigenen Platz. -->
	{#if nummer}
		<p class="mt-1.5 text-xs text-on-surface-variant" data-nummer>
			{nummerArt} <span class="font-mono">{nummer}</span>
		</p>
	{/if}
</div>

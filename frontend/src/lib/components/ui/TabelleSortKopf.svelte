<!-- @component TabelleSortKopf — eine sortierbare Kopfzelle für ui/Tabelle.

     Der erste sortierbare Spaltenkopf der Anwendung (17.09.2026, Protokoll des
     Medienzentrums Punkt 5). Als eigenes Bauteil, weil die Frage bei jeder weiteren
     Liste wiederkommt und 24 Tabellen schon einmal 24 Rezepte hatten.

     Die Zelle selbst trägt nur Layout (Breite, Ausrichtung) — das Aussehen kommt aus
     dem <style> von Tabelle.svelte, die Ratsche frontend-hygiene-tabellen prüft das.
     Bedient wird über einen echten <button> IN der Zelle: Ein klickbares <th> ohne
     Knopf erreicht die Tastatur nicht.

     `aria-sort` sitzt an der Zelle, nicht am Knopf — so verlangt es ARIA, und nur so
     liest ein Screenreader „aufsteigend sortiert" beim Betreten der Spalte.

     OHNE `onsortiere` ist das Bauteil ein gewöhnlicher Kopf. Dieselbe Liste steht auch
     dort, wo Sortieren nichts bewirkt (Kiosk, Abgänger, Schülersuche der Vormerkung) —
     ein Pfeil, der nichts tut, wäre dort schlimmer als keiner. -->
<script>
	import { ArrowDown, ArrowUp, ChevronsUpDown } from '@lucide/svelte';

	/**
	 * @type {{
	 *   spalte: string,
	 *   text: string,
	 *   sortierung?: { spalte: string, absteigend: boolean },
	 *   onsortiere?: (spalte: string) => void,
	 *   class?: string
	 * }}
	 */
	let {
		spalte,
		text,
		sortierung = { spalte: '', absteigend: false },
		onsortiere,
		class: klasse = ''
	} = $props();

	const aktiv = $derived(sortierung.spalte === spalte);
	const ariaSort = $derived(!aktiv ? 'none' : sortierung.absteigend ? 'descending' : 'ascending');
	// Was der nächste Klick TUT, steht im Titel — nicht, was gerade gilt. „Nach Name
	// sortieren" an einer schon sortierten Spalte wäre eine Lüge über den Knopf.
	const titel = $derived(
		!aktiv
			? `Nach ${text} sortieren`
			: sortierung.absteigend
				? `Nach ${text} aufsteigend sortieren`
				: `Nach ${text} absteigend sortieren`
	);
</script>

{#if !onsortiere}
	<th class={klasse}>{text}</th>
{:else}
	<th class={klasse} aria-sort={ariaSort}>
		<button
			type="button"
			onclick={() => onsortiere(spalte)}
			title={titel}
			aria-label={titel}
			class="inline-flex cursor-pointer items-center gap-1 rounded-sm hover:text-on-surface focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none"
		>
			{text}
			{#if !aktiv}
				<!-- Der Doppelpfeil sagt „hier kann man sortieren", ohne eine Richtung zu
				     behaupten. Ohne Symbol sähe die Spalte aus wie eine gewöhnliche. -->
				<ChevronsUpDown class="h-3.5 w-3.5 opacity-50" aria-hidden="true" />
			{:else if sortierung.absteigend}
				<ArrowDown class="h-3.5 w-3.5 text-primary" aria-hidden="true" />
			{:else}
				<ArrowUp class="h-3.5 w-3.5 text-primary" aria-hidden="true" />
			{/if}
		</button>
	</th>
{/if}

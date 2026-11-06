<!-- @component SelectListe — die Menüfläche des M3-Auswahlfelds und des Felds mit
     Vorschlägen (Select, VorschlagFeld).

     Eigene Datei aus zwei Gründen: Sie ist ein eigenes Bauteil (M3 nennt das
     „menu" und gibt ihm eigene Maße — 4-px-Ecken, 8-px-Innenabstand,
     48-px-Zeilen), und sie hält Select.svelte flach.

     POSITION FIXED, nicht absolute: Die Liste öffnet sich regelmäßig in einem
     Container mit overflow-y-auto (Formularspalten, Tabellen, Dialoge).
     Absolut positioniert würde sie dort abgeschnitten — dieselbe Falle, die in
     CoverVorschau schon dokumentiert ist. -->
<script>
	import { Check } from '@lucide/svelte';

	/**
	 * @type {{
	 *   options: Array<{ value: any, label: string, disabled?: boolean, zusatz?: string }>,
	 *   value: any,
	 *   aktiv: number,
	 *   box: { left: number, top: number, breite: number },
	 *   kennung: string,
	 *   zeilenKennung: (i: number) => string,
	 *   onwaehlen: (i: number) => void,
	 *   onaktiv: (i: number) => void,
	 *   onelement: (el: HTMLDivElement | undefined) => void
	 * }}
	 */
	let { options, value, aktiv, box, kennung, zeilenKennung, onwaehlen, onaktiv, onelement } =
		$props();

	/** @type {HTMLDivElement | undefined} */
	let el = $state();

	// Select braucht den Knoten, um Klicks außerhalb von Klicks in der Liste zu
	// unterscheiden.
	$effect(() => onelement(el));

	// Der aktive Eintrag muss sichtbar bleiben, sonst wandert die Auswahl blind. Das
	// Scrollen gehört hierher: Diese Datei hält die Liste, Select hält nur den Index.
	$effect(() => {
		if (aktiv < 0) return;
		el?.querySelector(`[data-i="${aktiv}"]`)?.scrollIntoView({ block: 'nearest' });
	});
</script>

<!-- Ein Klick in die Liste lässt den Fokus, wo er ist: Ein Textfeld mit Vorschlägen schlösse
     beim Verlassen seine Liste, bevor der Klick ankommt. -->
<div
	bind:this={el}
	id={`${kennung}-liste`}
	role="listbox"
	tabindex="-1"
	style="position:fixed; left:{box.left}px; top:{box.top}px; width:{box.breite}px; z-index:60;"
	class="max-h-80 overflow-y-auto rounded-sm bg-surface-container py-2 shadow-xl"
	onmousedown={(e) => e.preventDefault()}
>
	{#each options as o, i (o.value)}
		<div
			data-i={i}
			id={zeilenKennung(i)}
			role="option"
			aria-selected={o.value === value}
			aria-disabled={o.disabled || undefined}
			tabindex="-1"
			onclick={() => onwaehlen(i)}
			onkeydown={() => {}}
			onpointerenter={() => onaktiv(i)}
			class="m3-state flex h-12 cursor-pointer items-center gap-3 px-3 text-sm
				{o.disabled ? 'cursor-not-allowed opacity-40' : ''}
				{o.value === value
				? 'bg-secondary-container text-on-secondary-container'
				: i === aktiv
					? 'bg-on-surface/8 text-on-surface'
					: 'text-on-surface'}"
		>
			<span class="min-w-0 flex-1 truncate">{o.label}</span>
			<!-- M3 nennt das „List item trailing text“; Schrift und Farbe wie in material-web
			     (list-item-trailing-supporting-text: label-small, on-surface-variant). -->
			{#if o.zusatz}
				<span class="shrink-0 text-label-small text-on-surface-variant">{o.zusatz}</span>
			{/if}
			{#if o.value === value}
				<Check class="h-5 w-5 shrink-0" aria-hidden="true" />
			{/if}
		</div>
	{/each}
	{#if !options.length}
		<div class="flex h-12 items-center px-3 text-sm text-on-surface-variant">
			Keine Auswahl verfügbar
		</div>
	{/if}
</div>

<!-- @component SelectListe — die Menüfläche des M3-Auswahlfelds.

     Eigene Datei aus zwei Gründen: Sie ist ein eigenes Bauteil (M3 nennt das
     „menu" und gibt ihm eigene Maße — 4-px-Ecken, 8-px-Innenabstand,
     48-px-Zeilen), und sie hält Select.svelte flach.

     POSITION FIXED, nicht absolute: Die Liste öffnet sich regelmäßig in einem
     Container mit overflow-y-auto (Formularspalten, Tabellen, Dialoge).
     Absolut positioniert würde sie dort abgeschnitten — dieselbe Falle, die in
     CoverPeek schon dokumentiert ist. -->
<script>
	import { Check } from '@lucide/svelte';

	/**
	 * @type {{
	 *   options: Array<{ value: any, label: string, disabled?: boolean }>,
	 *   value: any,
	 *   aktiv: number,
	 *   box: { left: number, top: number, breite: number },
	 *   id?: string,
	 *   onwaehlen: (i: number) => void,
	 *   onaktiv: (i: number) => void,
	 *   onelement: (el: HTMLDivElement | undefined) => void
	 * }}
	 */
	let { options, value, aktiv, box, id, onwaehlen, onaktiv, onelement } = $props();

	/** @type {HTMLDivElement | undefined} */
	let el = $state();

	// Select braucht den Knoten, um Klicks außerhalb von Klicks in der Liste zu
	// unterscheiden.
	$effect(() => onelement(el));
</script>

<div
	bind:this={el}
	id={id ? `${id}-liste` : undefined}
	role="listbox"
	tabindex="-1"
	style="position:fixed; left:{box.left}px; top:{box.top}px; width:{box.breite}px; z-index:60;"
	class="max-h-80 overflow-y-auto rounded-sm bg-surface-container py-2 shadow-xl"
>
	{#each options as o, i (o.value)}
		<div
			data-i={i}
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

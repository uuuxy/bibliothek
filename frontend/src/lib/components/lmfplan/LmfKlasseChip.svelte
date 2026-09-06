<!-- @component LmfKlasseChip — ein Chip im Planer nach Material 3: Radius 8 px,
     Höhe 32 px, im Set (M3: „Don't display a single chip by itself. Chips should
     appear in a set"). Mit `onentfernen` ist er ein Input-Chip mit dem Pflicht-× zum
     Entfernen (Klasse aus einer geteilten Stunde, freier Tag); mit `onklick` ein
     Assist-Chip, und der beginnt nach M3 mit einem Verb („Write assist chips like
     buttons: start with a verb") — „12T3 einplanen". Eine EINZELNE Klasse in einer
     Zeile ist deshalb kein Chip, sondern Text (LmfPlanZeile). `hinweis` hängt eine
     leise Einordnung an („ohne Schüler"), ohne einen zweiten Chip zu bauen. -->
<script>
	import { Plus, X } from '@lucide/svelte';

	/** @type {{ name: string, hinweis?: string, onentfernen?: () => void, onklick?: () => void }} */
	let { name, hinweis = '', onentfernen = undefined, onklick = undefined } = $props();
</script>

{#if onklick}
	<button
		type="button"
		onclick={onklick}
		class="inline-flex h-8 cursor-pointer items-center gap-1 rounded-md border border-outline px-3 text-sm font-medium text-on-surface transition-colors hover:bg-surface-container"
		title="{name} einplanen{hinweis ? ` (${hinweis})` : ''}"
	>
		<Plus class="h-4 w-4" aria-hidden="true" />
		{name} einplanen
		{#if hinweis}<span class="font-normal text-on-surface-variant">· {hinweis}</span>{/if}
	</button>
{:else}
	<span
		class="inline-flex h-8 items-center gap-1 rounded-md bg-secondary-container pl-3 text-sm font-medium text-on-secondary-container {onentfernen
			? 'pr-0'
			: 'pr-3'}"
	>
		{name}
		{#if hinweis}<span class="font-normal opacity-80">· {hinweis}</span>{/if}
		{#if onentfernen}
			<!-- 32 × 32 px: die ganze Chip-Höhe als Zielfläche (Gate icon-trefferflaechen,
			     M3 Icon-Button „extra small"); vorher 24 px und damit zu klein. -->
			<button
				type="button"
				onclick={onentfernen}
				class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-full hover:bg-on-secondary-container/10"
				title="{name} aus dem Plan nehmen"
				aria-label="{name} aus dem Plan nehmen"
			>
				<X class="h-4 w-4" aria-hidden="true" />
			</button>
		{/if}
	</span>
{/if}

<!-- @component LmfKlasseChip — eine Klasse im Planer als Material-3-Chip: Radius 8 px,
     Höhe 32 px (Input-Chip), getönte Fläche. Mit `onentfernen` trägt er das × des
     Input-Chips (Klasse aus der Zeile nehmen), mit `onklick` ist er ein Assist-Chip
     (Klasse aus „Nicht im Plan" in den Plan holen). -->
<script>
	import { Plus, X } from '@lucide/svelte';

	/** @type {{ name: string, onentfernen?: () => void, onklick?: () => void }} */
	let { name, onentfernen = undefined, onklick = undefined } = $props();
</script>

{#if onklick}
	<button
		type="button"
		onclick={onklick}
		class="inline-flex h-8 cursor-pointer items-center gap-1 rounded-md border border-outline px-3 text-sm font-medium text-on-surface transition-colors hover:bg-surface-container"
		title="{name} in den Plan"
	>
		<Plus class="h-4 w-4" aria-hidden="true" />
		{name}
	</button>
{:else}
	<span
		class="inline-flex h-8 items-center gap-1 rounded-md bg-secondary-container pl-3 text-sm font-medium text-on-secondary-container {onentfernen
			? 'pr-1'
			: 'pr-3'}"
	>
		{name}
		{#if onentfernen}
			<button
				type="button"
				onclick={onentfernen}
				class="flex h-6 w-6 cursor-pointer items-center justify-center rounded-full hover:bg-on-secondary-container/10"
				title="{name} aus dem Plan nehmen"
				aria-label="{name} aus dem Plan nehmen"
			>
				<X class="h-4 w-4" aria-hidden="true" />
			</button>
		{/if}
	</span>
{/if}

<!-- @component VorlagenGalerie — Auswahl der Design-Vorlage als Bildergalerie.

     Bis zum 01.09.2026 war das ein Dropdown mit sieben Namen. „Marine (Seitenband)"
     sagt aber nichts darüber, wie die Karte aussieht — und jede Wahl fragt nach, weil
     sie das zentrale Design aller Arbeitsplätze ersetzt. Wer erst nach dem Anwenden
     sieht, was er gewählt hat, klickt sich siebenmal durch die Rückfrage. Die Galerie
     zeigt jede Vorlage als Miniatur VOR der Wahl.

     Wie das Dropdown davor ist sie eine AKTION, kein Zustand: Es gibt keine „aktive"
     Vorlage — nach dem Anwenden ist sie nur noch Ausgangspunkt fürs freie Editieren. -->
<script>
	import { ChevronDown } from '@lucide/svelte';
	import { AUSWEIS_VORLAGEN } from './ausweisVorlagen.js';
	import VorlageMiniatur from './VorlageMiniatur.svelte';

	/** @type {{ onVorlage: (kennung: string) => void }} */
	let { onVorlage } = $props();

	let offen = $state(false);
	/** @type {HTMLDivElement | undefined} */
	let wurzel = $state();

	/** @param {string} kennung */
	function waehle(kennung) {
		offen = false;
		onVorlage(kennung);
	}
</script>

<svelte:window
	onpointerdown={(e) => {
		if (offen && wurzel && !wurzel.contains(/** @type {Node} */ (e.target))) offen = false;
	}}
	onkeydown={(e) => {
		if (e.key === 'Escape') offen = false;
	}}
/>

<div class="relative" bind:this={wurzel}>
	<!-- Auslöser in der Gestalt von Select.svelte, damit er neben Barcode-Typ und
	     Karten-Hintergrund in derselben Reihe nicht aus der Rolle fällt. -->
	<button
		type="button"
		aria-haspopup="dialog"
		aria-expanded={offen}
		aria-label="Design-Vorlage"
		onclick={() => (offen = !offen)}
		class="flex h-9 w-full cursor-pointer items-center gap-2 rounded-xl border bg-surface-container-lowest px-3 text-left text-sm text-on-surface transition-colors focus-visible:outline-none {offen
			? 'border-primary ring-1 ring-primary'
			: 'border-outline-variant'}"
	>
		<span class="min-w-0 flex-1 truncate text-on-surface-variant">Vorlage wählen …</span>
		<ChevronDown
			class="h-4 w-4 shrink-0 text-on-surface-variant transition-transform {offen
				? 'rotate-180'
				: ''}"
			aria-hidden="true"
		/>
	</button>

	{#if offen}
		<div
			role="dialog"
			aria-label="Design-Vorlagen"
			class="absolute top-full left-0 z-50 mt-1 grid w-max max-w-[calc(100vw-4rem)] grid-cols-2 gap-1 rounded-sm bg-surface-container p-2 shadow-xl sm:grid-cols-3 md:grid-cols-4"
		>
			{#each AUSWEIS_VORLAGEN as v (v.value)}
				<button
					type="button"
					onclick={() => waehle(v.value)}
					aria-label={v.label}
					class="m3-state flex cursor-pointer flex-col items-center gap-1.5 rounded-sm p-2"
				>
					<span class="rounded-xs shadow-sm ring-1 ring-outline-variant">
						<VorlageMiniatur kennung={v.value} />
					</span>
					<span class="text-label-small text-on-surface">{v.label}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

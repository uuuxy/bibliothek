<!-- @component AuswahlAktionsleiste — schwebender Balken, sobald Schüler markiert sind.

     Der Balken liegt über dem Inhalt statt in der Werkzeugleiste: Wer eine Klasse
     markiert, scrollt dabei durch die Liste. Ein Knopf oben wäre nach dem dritten Haken
     aus dem Bild. Der Balken steht im dunklen Schema (styles/rollen.css). Warnung, Startfeld
     und Druckknopf teilt er mit dem Reiter „Ausweise je Klasse" (AusweisDruckSteuerung). -->
<script>
	import { X } from '@lucide/svelte';
	import AusweisDruckSteuerung from './AusweisDruckSteuerung.svelte';
	import { meldungenWeichenAus } from '../../actions/meldungenWeichenAus.js';

	/**
	 * @typedef {Object} Props
	 * @property {number} anzahl
	 * @property {number} ohneDatum   markierte Schüler ohne ableitbares Ablaufjahr
	 * @property {boolean} [etikettModus] Klebebogen statt Ausweiskarten (idStore.printMode)
	 * @property {number} [maxPosition]   Felder auf dem gewählten Bogen
	 * @property {number} [startPosition] erstes zu bedruckendes Feld (bindable)
	 * @property {() => void | Promise<void>} onDrucken
	 * @property {() => void} onLeeren
	 */
	/** @type {Props} */
	let {
		anzahl,
		ohneDatum,
		etikettModus = false,
		maxPosition = 21,
		startPosition = $bindable(1),
		onDrucken,
		onLeeren
	} = $props();
</script>

{#if anzahl > 0}
	<div
		class="no-print fixed inset-x-0 bottom-6 z-40 flex justify-center px-4"
		role="region"
		aria-label="Aktionen für die markierten Schüler"
	>
		<div
			class="schema-dunkel flex max-w-3xl items-center gap-4 rounded-2xl bg-surface-container px-4 py-3 text-on-surface shadow-2xl"
			use:meldungenWeichenAus
		>
			<span class="text-sm whitespace-nowrap">
				<span class="font-semibold">{anzahl}</span>
				{anzahl === 1 ? 'Schüler' : 'Schüler'} markiert
			</span>

			<AusweisDruckSteuerung
				{ohneDatum}
				{etikettModus}
				{maxPosition}
				bind:startPosition
				{onDrucken}
				rechts
			>
				<button
					type="button"
					onclick={onLeeren}
					aria-label="Markierung aufheben"
					data-tip="Markierung aufheben"
					class="icon-btn text-on-surface-variant"
				>
					<X class="h-4 w-4" />
				</button>
			</AusweisDruckSteuerung>
		</div>
	</div>
{/if}

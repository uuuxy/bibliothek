<!-- @component AusweisDruckSteuerung — was vor dem Druck eines Ausweis-Stapels zu entscheiden ist,
     an einer Stelle für die Leserdatei (AuswahlAktionsleiste) und den Reiter „Ausweise je Klasse"
     (AusweiseJeKlasse): die Warnung ohne Ablaufjahr, das erste Feld eines angebrochenen Bogens und
     der Druckknopf, der während des Drucks gesperrt bleibt. Ohne eigenen Rahmen: Wo die Teile
     stehen, legt der Aufrufer fest. -->
<script>
	import { IdCard } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';

	/**
	 * @typedef {Object} Props
	 * @property {number} ohneDatum   Schüler ohne ableitbares Ablaufjahr
	 * @property {boolean} [etikettModus] Klebebogen statt Ausweiskarten (idStore.printMode)
	 * @property {number} [maxPosition]   Felder auf dem gewählten Bogen
	 * @property {number} [startPosition] erstes zu bedruckendes Feld (bindable)
	 * @property {boolean} [gesperrt]     nichts zu drucken
	 * @property {boolean} [rechts]       Knopf rechtsbündig, wie in der Aktionsleiste
	 * @property {() => void | Promise<void>} onDrucken
	 * @property {import('svelte').Snippet} [children]  steht neben dem Knopf
	 */
	/** @type {Props} */
	let {
		ohneDatum,
		etikettModus = false,
		maxPosition = 21,
		startPosition = $bindable(1),
		gesperrt = false,
		rechts = false,
		onDrucken,
		children
	} = $props();

	// Ein zweiter Klick vor dem Ende des Drucks schickte den Etikettenbogen noch einmal und öffnete
	// ihn in einem zweiten Tab; im Kartenmodus liegt vor dem Druckdialog der Abruf des Designs.
	let druckt = $state(false);
	async function drucke() {
		if (druckt) return;
		druckt = true;
		try {
			await onDrucken();
		} finally {
			druckt = false;
		}
	}
</script>

<!-- Der Hinweis gilt der Ausweiskarte: Sie trägt sonst "Gültig bis: 31.07.–", und wer das erst am
     fertigen Stapel merkt, hat die Rohlinge verbraucht. Auf dem Etikett steht kein Ablaufdatum —
     dort wäre die Warnung nur Lärm. -->
{#if ohneDatum > 0 && !etikettModus}
	<span
		class="rounded-lg bg-warning-container px-2.5 py-1 text-xs leading-snug text-on-warning-container"
	>
		{ohneDatum} davon ohne Ablaufjahr — Klasse lässt keine Ableitung zu. Einzeln im Profil eintragen.
	</span>
{/if}

{#if etikettModus}
	<!-- Angebrochener Bogen: Auf welchem Feld soll der Druck anfangen? Dieselbe Angabe wie bei den
	     Buch-Etiketten, hier direkt am Druckknopf, weil es die einzige Entscheidung des Vorgangs ist. -->
	<label class="flex shrink-0 items-center gap-2 text-xs whitespace-nowrap text-on-surface-variant">
		Ab Feld
		<Feld type="number" min="1" max={maxPosition} bind:value={startPosition} feld="w-16" />
		<span>von {maxPosition}</span>
	</label>
{/if}

<div class="{rechts ? 'ml-auto ' : ''}flex items-center gap-2">
	<Button variant="primary" onclick={drucke} disabled={druckt || gesperrt}>
		{#if druckt}
			<Ladekreis size="sm" farbe="aktuell" />
		{:else}
			<IdCard class="h-4 w-4" />
		{/if}
		{etikettModus ? 'Etiketten drucken' : 'Ausweise drucken'}
	</Button>
	{@render children?.()}
</div>

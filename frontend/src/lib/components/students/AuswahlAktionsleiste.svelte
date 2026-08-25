<!-- @component AuswahlAktionsleiste — schwebender Balken, sobald Schüler markiert sind.

     Der Balken liegt über dem Inhalt statt in der Werkzeugleiste: Wer eine Klasse
     markiert, scrollt dabei durch die Liste. Ein Knopf oben wäre nach dem dritten Haken
     aus dem Bild — und man sucht ihn, statt zu drucken. -->
<script>
	import { IdCard, X } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	/**
	 * @typedef {Object} Props
	 * @property {number} anzahl
	 * @property {number} ohneDatum   markierte Schüler ohne ableitbares Ablaufjahr
	 * @property {boolean} [etikettModus] Klebebogen statt Ausweiskarten (idStore.printMode)
	 * @property {number} [maxPosition]   Felder auf dem gewählten Bogen
	 * @property {number} [startPosition] erstes zu bedruckendes Feld (bindable)
	 * @property {() => void} onDrucken
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
			class="flex max-w-3xl items-center gap-4 rounded-2xl border border-slate-700/10 bg-slate-900 px-4 py-3 text-white shadow-2xl"
		>
			<span class="text-sm whitespace-nowrap">
				<span class="font-semibold">{anzahl}</span>
				{anzahl === 1 ? 'Schüler' : 'Schüler'} markiert
			</span>

			<!-- Der Hinweis gilt der AUSWEISKARTE: Sie trägt sonst "Gültig bis: 31.07.–",
			     und wer das erst am fertigen Stapel merkt, hat die Rohlinge verbraucht.
			     Auf dem Etikett steht kein Ablaufdatum — dort wäre die Warnung nur Lärm. -->
			{#if ohneDatum > 0 && !etikettModus}
				<span class="rounded-lg bg-amber-500/15 px-2.5 py-1 text-xs leading-snug text-amber-200">
					{ohneDatum} davon ohne Ablaufjahr — Klasse lässt keine Ableitung zu. Einzeln im Profil eintragen.
				</span>
			{/if}

			{#if etikettModus}
				<!-- Angebrochener Bogen: Auf welchem Feld soll der Druck anfangen? Dieselbe
				     Angabe wie bei den Buch-Etiketten, hier direkt am Druckknopf, weil es
				     die einzige Entscheidung dieses Vorgangs ist. -->
				<label class="flex shrink-0 items-center gap-2 text-xs whitespace-nowrap text-white/75">
					Ab Feld
					<Feld type="number" min="1" max={maxPosition} bind:value={startPosition} feld="w-16" />
					<span class="text-white/55">von {maxPosition}</span>
				</label>
			{/if}

			<div class="ml-auto flex items-center gap-2">
				<Button variant="primary" onclick={onDrucken}>
					<IdCard class="h-4 w-4" />
					{etikettModus ? 'Etiketten drucken' : 'Ausweise drucken'}
				</Button>
				<button
					type="button"
					onclick={onLeeren}
					aria-label="Markierung aufheben"
					data-tip="Markierung aufheben"
					class="rounded-lg p-2 text-slate-300 transition-colors hover:bg-white/10 hover:text-white"
				>
					<X class="h-4 w-4" />
				</button>
			</div>
		</div>
	</div>
{/if}

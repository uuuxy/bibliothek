<!-- @component Eine Position im Warenkorb — Cover, Titel, Menge, Preis, Topf-Wechsel.

     Eigene Datei, weil der Warenkorb nach Topf gruppiert: OrderCart trägt die Gruppen und
     den Absenden-Knopf, diese Datei die Zeile. Beides zusammen läge über der
     200-Zeilen-Marke. -->
<script>
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import Feld from '../ui/Feld.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import { MITTEL, anderesMittel } from './mittel.js';
	import { X, Tag } from '@lucide/svelte';

	/** @type {{ item: import('../../stores/orderStore.svelte.js').CartItem }} */
	let { item } = $props();

	/** Wohin die Position wechseln würde — der Knopf nennt das ZIEL, nicht den Zustand. */
	const ziel = $derived(MITTEL[anderesMittel(item.mittel)]);
</script>

<div class="space-y-2.5 rounded-xl border border-outline-variant bg-surface-container-lowest p-3">
	<div class="flex items-start gap-2.5">
		<BuchCover coverUrl={item.cover_url} isbn={item.isbn} titel={item.titel} klasse="shrink-0" />
		<div class="min-w-0 flex-1">
			<h4 class="truncate text-sm leading-snug font-semibold text-on-surface">
				{item.titel}
			</h4>
			<p class="truncate font-mono text-xs text-on-surface-variant">{item.isbn || '—'}</p>
			{#if item.generate_barcodes}
				<div class="mt-1">
					<StatusChip icon={Tag} text={item.menge === 1 ? '1 Barcode' : `${item.menge} Barcodes`} />
				</div>
			{/if}
		</div>
		<button
			onclick={() => orderStore.removeFromCart(item)}
			aria-label="Entfernen"
			class="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-full text-on-surface-variant transition-colors hover:text-error"
		>
			<X class="w-3.5 h-3.5" aria-hidden="true" />
		</button>
	</div>

	<div class="flex items-center justify-between gap-2 pl-10">
		<div
			class="flex items-center overflow-hidden rounded-xl border border-outline-variant bg-surface-container-lowest"
		>
			<button
				aria-label="Menge verringern"
				title={item.menge <= 1 ? 'Mindestmenge erreicht' : ''}
				disabled={item.menge <= 1}
				onclick={() => (item.menge = Math.max(1, item.menge - 1))}
				class="cursor-pointer px-2.5 py-1 font-bold text-on-surface-variant disabled:opacity-50 disabled:cursor-not-allowed"
				>−</button
			><span class="min-w-6 px-2 text-center text-sm font-bold text-on-surface tabular-nums"
				>{item.menge}</span
			><button
				aria-label="Menge erhöhen"
				onclick={() => (item.menge += 1)}
				class="cursor-pointer px-2.5 py-1 font-bold text-on-surface-variant">+</button
			>
		</div>
		{#if orderStore.preiseErfassen}
			<div class="flex items-center gap-1.5">
				<!-- Der Vorschlag bleibt als solcher erkennbar, solange er unverändert ist: Er ist
				     der DNB-Ladenpreis bei Erscheinen, nicht der Preis, den die Schule zahlt. Wer
				     ihn überschreibt, verliert das Abzeichen. -->
				{#if item.preis_vorschlag > 0 && Number(item.preis) === item.preis_vorschlag}
					<StatusChip
						ton="warten"
						text="DNB"
						tip="Ladenpreis aus dem DNB-Datensatz — bitte gegen den Schulpreis prüfen"
					/>
				{/if}
				<Feld
					type="number"
					step="0.01"
					bind:value={item.preis}
					aria-label="Preis"
					feld="w-20 text-right font-semibold"
				/>
				<span class="text-sm font-semibold text-on-surface-variant">€</span>
			</div>
		{/if}
	</div>

	<!-- Der Topf ist eine Entscheidung je Bestellung; der Titel schlägt ihn nur vor. Ein
	     falsch gekennzeichnetes Schulbuch wandert hier in die Lernmittel-Bestellung, ohne
	     dass jemand erst den Titel bearbeiten muss. Der Knopf nennt das Ziel. -->
	<div class="pl-10">
		<button
			onclick={() => orderStore.verschiebe(item)}
			class="text-label-small font-medium text-primary hover:underline cursor-pointer"
			aria-label="{item.titel} in die Bestellung „{ziel.label}“ verschieben"
		>
			→ {ziel.label} ({ziel.traeger})
		</button>
	</div>
</div>

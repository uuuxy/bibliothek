<!-- @component Eine Position im Warenkorb — Cover, Titel, Menge, Preis, Topf-Wechsel.

     Eigene Datei, seit der Warenkorb nach Topf gruppiert (10.09.2026): OrderCart trägt
     die Gruppen und den Absenden-Knopf, diese Datei die Zeile. Beides zusammen lag über
     der 200-Zeilen-Marke. -->
<script>
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import Feld from '../ui/Feld.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import { MITTEL, anderesMittel } from './mittel.js';
	import { X, Tag } from '@lucide/svelte';

	/** @type {{ item: import('../../stores/orderStore.svelte.js').CartItem }} */
	let { item } = $props();

	/** Wohin die Position wechseln würde — der Knopf nennt das ZIEL, nicht den Zustand. */
	const ziel = $derived(MITTEL[anderesMittel(item.mittel)]);
</script>

<div class="rounded-xl border border-slate-200 bg-white p-3 space-y-2.5">
	<div class="flex items-start gap-2.5">
		<BuchCover coverUrl={item.cover_url} isbn={item.isbn} titel={item.titel} klasse="shrink-0" />
		<div class="min-w-0 flex-1">
			<h4 class="font-semibold text-slate-900 text-sm truncate leading-snug">
				{item.titel}
			</h4>
			<p class="text-xs text-slate-400 truncate font-mono">{item.isbn || '—'}</p>
			{#if item.generate_barcodes}
				<div
					class="text-label-small font-bold text-blue-600 mt-1 flex items-center gap-1 bg-blue-50 w-fit px-1.5 py-0.5 rounded-md"
				>
					<Tag class="w-3 h-3" aria-hidden="true" />
					{item.menge}
					{item.menge === 1 ? 'Barcode' : 'Barcodes'}
				</div>
			{/if}
		</div>
		<button
			onclick={() => orderStore.removeFromCart(item)}
			aria-label="Entfernen"
			class="shrink-0 w-6 h-6 rounded-full text-slate-400 hover:text-rose-500 hover:bg-rose-50 flex items-center justify-center cursor-pointer transition-colors"
		>
			<X class="w-3.5 h-3.5" aria-hidden="true" />
		</button>
	</div>

	<div class="flex items-center justify-between gap-2 pl-10">
		<div class="flex items-center border border-slate-200 bg-white rounded-xl overflow-hidden">
			<button
				aria-label="Menge verringern"
				onclick={() => (item.menge = Math.max(1, item.menge - 1))}
				class="px-2.5 py-1 hover:bg-slate-50 font-bold text-slate-500 cursor-pointer">−</button
			><span class="px-2 font-bold text-slate-800 text-sm min-w-6 text-center tabular-nums"
				>{item.menge}</span
			><button
				aria-label="Menge erhöhen"
				onclick={() => (item.menge += 1)}
				class="px-2.5 py-1 hover:bg-slate-50 font-bold text-slate-500 cursor-pointer">+</button
			>
		</div>
		{#if orderStore.preiseErfassen}
			<div class="flex items-center gap-1.5">
				<!-- Der Vorschlag bleibt als solcher erkennbar, solange er unveraendert ist.
				     Er ist der DNB-Ladenpreis bei Erscheinen — NICHT der Schulpreis, den die
				     Schule tatsaechlich zahlt. Wer ihn ueberschreibt, verliert das Abzeichen
				     und damit die Erinnerung daran, dass hier geraten wurde. -->
				{#if item.preis_vorschlag > 0 && Number(item.preis) === item.preis_vorschlag}
					<span
						class="text-label-small font-bold uppercase text-amber-700 bg-amber-50 border border-amber-100 px-1.5 py-0.5 rounded"
						title="Ladenpreis aus dem DNB-Datensatz — bitte gegen den Schulpreis pruefen"
					>
						DNB
					</span>
				{/if}
				<Feld
					type="number"
					step="0.01"
					bind:value={item.preis}
					aria-label="Preis"
					feld="w-20 text-right font-semibold"
				/>
				<span class="text-sm font-semibold text-slate-400">€</span>
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

<!-- @component BuchKarteFuss — Regaladresse, Prüfdatum und Bestand am Fuß der Buchkarte.

     Aus BuchKarte.svelte herausgelöst (02.09.2026), als drei Dinge zu ändern waren und
     die Karte im eingefrorenen Größen-Bestand stand:
       - Die SIGNATUR fehlte — laut Handbuch „die Regaladresse", der Payload trug sie
         längst. Wer ein Buch sucht, will wissen, wo es steht.
       - „Zuletzt geprüft: Unbekannt" stand auf JEDER Karte: ein Inventurdatum, das
         fast nie bekannt ist (lokal 0 von 761 Exemplaren je gezählt). Jetzt nur, wenn
         es eines gibt.
       - „Verfügbar 0 / 0" sah aus wie „alles verliehen": derselbe rote Punkt für einen
         Titel OHNE Exemplare wie für einen komplett ausgeliehenen. Jetzt „Keine
         Exemplare" in Grau — und der Bestandsfilter findet genau diese Titel. -->
<script>
	import { Clock, MapPin } from '@lucide/svelte';
	import { getStockDotColor, formatDate } from '../bookHelpers.js';

	/** @type {{ book: { signatur?: string, lastCounted?: string, verfuegbar?: number, gesamt?: number } }} */
	let { book } = $props();
</script>

<div class="space-y-3">
	{#if book.signatur}
		<div class="flex items-center gap-1.5 text-sm text-on-surface-variant">
			<MapPin class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
			<span class="font-mono tracking-wide">{book.signatur}</span>
		</div>
	{/if}

	{#if book.lastCounted}
		<div
			class="inline-flex items-center gap-1.5 w-full px-2.5 py-1.5 rounded-lg bg-slate-50 border border-slate-100 text-label-small text-slate-500 font-medium"
		>
			<Clock class="w-3.5 h-3.5 text-slate-400" aria-hidden="true" />
			<span>Zuletzt geprüft: {formatDate(book.lastCounted)}</span>
		</div>
	{/if}

	<div class="pt-3 border-t border-slate-100 flex justify-between items-center">
		<span class="text-xs font-semibold text-slate-400"
			>{book.gesamt === 0 ? 'Bestand' : 'Verfügbar'}</span
		>
		{#if book.gesamt === 0}
			<span class="text-sm font-medium text-on-surface-variant">Keine Exemplare</span>
		{:else}
			<div class="flex items-center gap-2">
				<span class="w-2 h-2 rounded-full {getStockDotColor(book.verfuegbar || 0)}"></span>
				<span class="text-lg font-extrabold text-slate-800">{book.verfuegbar || 0}</span>
				{#if book.gesamt !== undefined}
					<span class="text-xs text-slate-500 font-medium">/ {book.gesamt}</span>
				{/if}
			</div>
		{/if}
	</div>
</div>

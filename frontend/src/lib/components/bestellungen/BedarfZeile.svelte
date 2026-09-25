<!-- @component BedarfZeile — eine Zeile des Bestellbedarfs, aus OrderRecommendations.svelte
     herausgezogen (Dateigrößen-Ratsche), als die Liste am 25.09.2026 am Buch zu zählen begann
     (docs/OFFEN.md 4.18, Stufe 3). Gehört der Titel zu einem Buch mit mehreren Auflagen, ist
     die Zeile das Buch: Titel und ISBN der neuesten Auflage — sie wird bestellt —, die Zahlen
     sind die Summe, und die dritte Zeile nennt die Auflagen einzeln. Material 3, Lists:
     „Limit supporting text to one to three lines". Die Farben der übrigen Zeile stehen, wie
     sie waren; ihre Umstellung auf Rollen ist 5.21, Bildschirm für Bildschirm. -->
<script>
	import { Plus } from '@lucide/svelte';
	import CoverPeek from '../ui/CoverPeek.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import { auflagenAufschluesselung } from '../../utils/auflagenText.js';

	/** @type {{ r: any, onAddToCart: (r: any) => void }} */
	let { r, onAddToCart } = $props();
</script>

<div
	class="group flex items-center gap-3 rounded-xl border border-transparent px-3 py-2 hover:bg-slate-50 hover:border-slate-200 transition-colors"
>
	<!-- Cover IN der Zeile, zugleich Auslöser der Großansicht (so sieht CoverPeek es
	     über `children` vor). Frühere Gegengründe gemessen widerlegt: `loading="lazy"`
	     erspart die 247 Requests, 5.724 von 8.706 Titeln ohne Exemplar haben ein Cover. -->
	<CoverPeek isbn={r.isbn || ''} coverUrl={r.cover_url || ''} titel={r.titel}>
		<BuchCover coverUrl={r.cover_url || ''} isbn={r.isbn || ''} titel={r.titel} />
	</CoverPeek>

	<div class="min-w-0 flex-1">
		<h4 class="font-semibold text-slate-900 text-sm truncate leading-snug">{r.titel}</h4>
		<p class="text-xs text-slate-500 truncate">
			{#if r.isbn}<span class="font-mono text-slate-400">{r.isbn}</span>{/if}
			{#if r.verlag}<span class="mx-1.5 text-slate-400">·</span>{r.verlag}{/if}
			{#if r.signatur}<span class="mx-1.5 text-slate-400">·</span>{r.signatur}{/if}
		</p>
		{#if r.auflagen?.length > 1}
			<p class="truncate text-xs text-on-surface-variant">
				{auflagenAufschluesselung(r.auflagen)}
			</p>
		{/if}
	</div>

	<!-- Durchgehend dieselbe Bestandsspalte statt eines Pills im Null-Fall. Das
	     „Fehlt komplett"-Pill stand auf 252 von 334 Zeilen — ein Signal, das auf drei
	     Vierteln der Liste steht, markiert den Normalzustand statt der Ausnahme.

	     Dieselbe Ueberlegung gilt fuer die FARBE, und dort stand sie noch aus: Rot
	     fuer gesamt_bestand === 0 traf 243 von 327 Zeilen. In M3 traegt die
	     Error-Rolle Zustaende, die korrigiert werden muessen — in einer Liste, die
	     ausschliesslich Bedarf enthaelt, ist Bedarf kein Fehler. Die Zugehoerigkeit
	     zur Liste IST das Signal; innerhalb der Liste rangiert jetzt die BETONUNG
	     (on-surface gegen on-surface-variant) statt einer zweiten Alarmfarbe. -->
	<div
		class="text-right shrink-0 leading-tight text-sm font-bold tabular-nums {r.gesamt_bestand === 0
			? 'text-slate-900'
			: 'text-slate-500'}"
		title="verfügbar / im Bestand"
	>
		{r.verfuegbarer_bestand}<span class="text-slate-400 font-medium">/</span>{r.gesamt_bestand}
	</div>

	<button
		onclick={() => onAddToCart(r)}
		aria-label="{r.titel} zur Bestellung hinzufügen"
		data-tip="Zur Bestellung hinzufügen"
		class="shrink-0 w-9 h-9 rounded-full border border-slate-200 text-slate-400 flex items-center justify-center hover:border-blue-500 hover:text-white hover:bg-blue-600 active:scale-90 transition-all cursor-pointer"
	>
		<Plus class="w-4 h-4" aria-hidden="true" />
	</button>
</div>

<!-- @component ZusammenfuehrenKandidat — eine Schülerzeile im Zusammenführen-Dialog:
     Name, Klasse, Barcode, Geburtsdatum und die Marken, die beim Zusammenführen zählen
     (Abgänger, gesperrt, offene Bücher). Als Karte mit Etikett für die beiden gewählten
     Datensätze, kompakt als Zeile in der Trefferliste. -->
<script>
	import { X } from '@lucide/svelte';

	/** @type {{ kandidat: any, etikett?: string, kompakt?: boolean, onAbwaehlen?: () => void }} */
	let { kandidat, etikett = '', kompakt = false, onAbwaehlen = undefined } = $props();

	/** @param {string | null | undefined} iso */
	function datum(iso) {
		if (!iso) return '';
		const [j, m, t] = String(iso).slice(0, 10).split('-');
		return `${t}.${m}.${j}`;
	}

	const offene = $derived(
		typeof kandidat?.offene_buecher === 'number'
			? kandidat.offene_buecher
			: (kandidat?.entliehene_buecher?.length ?? 0)
	);
</script>

<div
	class={kompakt
		? 'flex flex-wrap items-center gap-x-3 gap-y-1 text-sm'
		: 'rounded-xl border border-outline-variant bg-surface-container-lowest px-4 py-3 flex items-start justify-between gap-3'}
>
	<div class="min-w-0 space-y-0.5">
		{#if etikett}
			<p class="text-label-small font-semibold text-on-surface-variant uppercase tracking-wide">
				{etikett}
			</p>
		{/if}
		<p class="text-sm font-semibold text-on-surface">
			{kandidat.vorname}
			{kandidat.nachname}
			<span class="font-mono font-normal text-on-surface-variant">{kandidat.klasse}</span>
		</p>
		<p class="text-xs text-on-surface-variant flex flex-wrap gap-x-3 gap-y-0.5">
			<span class="font-mono">{kandidat.barcode_id}</span>
			{#if kandidat.geburtsdatum}<span>geb. {datum(kandidat.geburtsdatum)}</span>{/if}
			{#if kandidat.ist_abgaenger}<span class="font-semibold">Abgänger</span>{/if}
			{#if kandidat.ist_gesperrt}<span class="font-semibold text-error">gesperrt</span>{/if}
			{#if offene > 0}<span>{offene} offene {offene === 1 ? 'Ausleihe' : 'Ausleihen'}</span>{/if}
		</p>
	</div>
	{#if onAbwaehlen}
		<button
			type="button"
			onclick={onAbwaehlen}
			aria-label="Auswahl aufheben"
			class="p-1 text-on-surface-variant hover:text-on-surface rounded-lg hover:bg-surface-container transition-colors shrink-0"
		>
			<X class="w-4 h-4" aria-hidden="true" />
		</button>
	{/if}
</div>

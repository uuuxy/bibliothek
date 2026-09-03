<!-- @component FachKarte — ein Fach im Portal-Reiter „Schulbücher", baugleich mit der
     Klassensatz-Karte (KlassenKarte): Zeile mit Zähler-Chip und Namen, Klick klappt die
     Bücher als Cover-Raster auf.

     Peter am 03.09.2026: „Englisch möchte nur die Englisch-Bücher … im Grunde ähnlich wie
     bei Klassensätzen." Genau dieselbe Frage wie dort — „was hat MEINE Klasse / MEIN Fach,
     und stimmt der Bestand?" —, also dieselbe Bauform statt einer dritten. Die Zeile
     beantwortet „wie viel?" schon eingeklappt; das Raster beantwortet „was genau?" für das
     eine Fach, das man gerade ansieht. Der Excel-Knopf sitzt am Fach, nicht an der Seite:
     Der Fachsprecher exportiert sein Fach, nicht die ganze Schule. -->
<script>
	import { ChevronDown, Download } from '@lucide/svelte';
	import KlassenBuchKachel from '../../../inventur/lib/components/admin/KlassenBuchKachel.svelte';

	/**
	 * @type {{
	 *   fach: { fach: string, titel: number, gesamt: number, verliehen: number },
	 *   buecher: any[],
	 *   offen?: boolean,
	 *   exportUrl: string,
	 *   onToggle: () => void
	 * }}
	 */
	let { fach, buecher, offen = false, exportUrl, onToggle } = $props();

	const name = $derived(fach.fach || 'Ohne Fach');
	const rasterID = $derived(`schulbuecher-${(fach.fach || 'ohne-fach').replace(/\s+/g, '-')}`);
</script>

<div class="border-b border-outline-variant/60 last:border-b-0">
	<div class="flex items-center justify-between gap-4 py-3">
		<!-- Die ganze Zeile schaltet um, nicht nur das Dreieck: Das Ziel ist so groß wie die
		     Aussage, die es betrifft (wie in KlassenKarte). -->
		<button
			type="button"
			onclick={onToggle}
			aria-expanded={offen}
			aria-controls={rasterID}
			class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 rounded-lg py-1 text-left"
		>
			<ChevronDown
				class="h-5 w-5 shrink-0 text-on-surface-variant transition-transform {offen
					? ''
					: '-rotate-90'}"
				aria-hidden="true"
			/>
			<span
				class="inline-flex min-w-[6.5rem] shrink-0 justify-center rounded-full bg-secondary-container px-3 py-0.5 text-xs font-semibold tabular-nums text-on-secondary-container"
				>{fach.gesamt} Exemplare</span
			>
			<span class="truncate text-base font-medium text-on-surface">{name}</span>
			<span class="shrink-0 text-sm tabular-nums text-on-surface-variant">
				{fach.titel} Titel{fach.verliehen ? ` · ${fach.verliehen} verliehen` : ''}
			</span>
		</button>

		<a
			href={exportUrl}
			download
			title="{name} als Excel-Datei"
			class="inline-flex h-9 shrink-0 items-center gap-2 rounded-full px-4 text-label-large font-semibold text-on-surface-variant hover:bg-surface-container"
		>
			<Download size={18} aria-hidden="true" />
			Als Excel
		</a>
	</div>

	{#if offen}
		<!-- Umbrechendes Raster, kein Karussell — dieselbe Entscheidung wie bei den
		     Klassensätzen (08.08.2026): Pfeile auf :hover gibt es am Tablet nicht. -->
		<div
			id={rasterID}
			class="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-x-3 gap-y-4 pt-1 pb-6"
			data-testid="schulbuecher-raster"
		>
			{#each buecher as book (book.id)}
				<KlassenBuchKachel {book} bearbeitbar={false} />
			{/each}
		</div>
	{/if}
</div>

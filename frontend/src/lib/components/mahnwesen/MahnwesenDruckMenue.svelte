<!-- @component MahnwesenDruckMenue — Split-Button „Mahnbriefe" mit Geltungsbereich.

     Eine Primäraktion (Eltern-Briefe drucken) plus ein Menü für alles, was denselben
     Vorgang mit anderem Umfang meint. Ersetzt die vier früher verstreuten PDF-Wege.
     Drucker- und Dokument-Symbole — kein Umschlag, denn hier wird nichts gemailt. -->
<script>
	import { scale } from 'svelte/transition';
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';

	let offen = $state(false);
	let anker = $state(/** @type {HTMLElement | null} */ (null));

	$effect(() => {
		if (!offen) return;
		/** @param {PointerEvent} e */
		const onDown = (e) => {
			if (anker && !anker.contains(/** @type {Node} */ (e.target))) offen = false;
		};
		/** @param {KeyboardEvent} e */
		const onKey = (e) => {
			if (e.key !== 'Escape') return;
			offen = false;
			// Als verarbeitet melden, sonst verlässt derselbe Tastendruck zusätzlich das
			// Mahnwesen (globaler Escape-Kurzbefehl in Router.svelte).
			e.preventDefault();
		};
		document.addEventListener('pointerdown', onDown);
		document.addEventListener('keydown', onKey);
		return () => {
			document.removeEventListener('pointerdown', onDown);
			document.removeEventListener('keydown', onKey);
		};
	});

	const BRIEF =
		'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z';
	const HERUNTERLADEN = 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4';
	const DRUCKER =
		'M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z';

	const eintragKlasse =
		'w-full text-left px-3 py-2.5 rounded-xl hover:bg-slate-50 disabled:opacity-50 text-sm font-semibold text-slate-700 flex items-center gap-2.5';
</script>

{#snippet symbol(d)}
	<svg
		class="h-4 w-4 text-slate-400 shrink-0"
		fill="none"
		viewBox="0 0 24 24"
		stroke="currentColor"
		stroke-width="2"
		aria-hidden="true"
	>
		<path stroke-linecap="round" stroke-linejoin="round" {d} />
	</svg>
{/snippet}

<div class="relative" bind:this={anker}>
	<div class="inline-flex rounded-md shadow-sm">
		<Button
			onclick={mahnwesenStore.downloadElternPDF}
			disabled={mahnwesenStore.elternPdfLoading}
			class="rounded-r-none"
		>
			{#if mahnwesenStore.elternPdfLoading}
				<div
					class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"
				></div>
			{:else}
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
					aria-hidden="true"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d={BRIEF} />
				</svg>
			{/if}
			Mahnbriefe
		</Button>
		<Button
			onclick={() => (offen = !offen)}
			aria-haspopup="menu"
			aria-expanded={offen}
			aria-label="Weitere Druck- und Export-Optionen"
			data-tip="Weitere Druck- und Export-Optionen"
			class="rounded-l-none border-l-white/25 px-2"
		>
			<svg
				class="h-3.5 w-3.5 transition-transform {offen ? 'rotate-180' : ''}"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2.5"
				aria-hidden="true"
			>
				<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
			</svg>
		</Button>
	</div>

	{#if offen}
		<div
			role="menu"
			tabindex="-1"
			transition:scale={{ duration: 130, start: 0.95, opacity: 0 }}
			class="absolute right-0 top-full mt-2 w-72 origin-top-right bg-surface-container rounded-sm shadow-xl p-2 z-30"
		>
			<div class="px-2 pt-1 pb-1 text-xs font-medium text-slate-400">Mahnbriefe an Eltern</div>
			<button
				role="menuitem"
				onclick={() => {
					offen = false;
					mahnwesenStore.downloadElternPDF();
				}}
				disabled={mahnwesenStore.elternPdfLoading}
				class={eintragKlasse}
			>
				{@render symbol(BRIEF)}
				Alle überfälligen
			</button>
			<div class="px-3 pt-2 pb-1">
				<div class="text-label-small font-semibold text-slate-400 mb-1.5">Ganze Klasse</div>
				<div class="flex items-center gap-2">
					<Select
						bind:value={mahnwesenStore.selectedKlasse}
						options={mahnwesenStore.klassen.map((/** @type {any} */ k) => ({
							value: k.klasse,
							label: k.klasse
						}))}
						placeholder="Klasse wählen …"
						class="flex-1 min-w-0"
						aria-label="Klasse für den Sammel-Mahnlauf"
					/>
					<Button
						size="sm"
						onclick={() => {
							mahnwesenStore.downloadKlassePDF(mahnwesenStore.selectedKlasse);
							offen = false;
						}}
						disabled={mahnwesenStore.klassePdfLoading || !mahnwesenStore.selectedKlasse}
						class="shrink-0 disabled:bg-slate-200 disabled:text-slate-400 disabled:opacity-100"
					>
						PDF
					</Button>
				</div>
			</div>

			<div class="border-t border-slate-100 my-1.5"></div>
			<div class="px-2 pt-1 pb-1 text-xs font-medium text-slate-400">Weitere</div>
			<button
				role="menuitem"
				onclick={() => {
					offen = false;
					mahnwesenStore.downloadPDF();
				}}
				disabled={mahnwesenStore.pdfLoading}
				class={eintragKlasse}
			>
				{@render symbol(HERUNTERLADEN)}
				Übersichtsliste (PDF)
			</button>
			<button
				role="menuitem"
				onclick={() => {
					offen = false;
					window.print();
				}}
				class={eintragKlasse}
			>
				{@render symbol(DRUCKER)}
				Diese Seite drucken
			</button>
		</div>
	{/if}
</div>

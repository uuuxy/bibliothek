<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import { scale } from 'svelte/transition';

	// Split-Button-Menü (Mahnbriefe): eigener Open-State, schließt bei Klick außerhalb & Escape.
	let menuOpen = $state(false);
	let menuAnchor = $state(/** @type {HTMLElement | null} */ (null));
	$effect(() => {
		if (!menuOpen) return;
		/** @param {PointerEvent} e */
		const onDown = (e) => {
			if (menuAnchor && !menuAnchor.contains(/** @type {Node} */ (e.target))) menuOpen = false;
		};
		/** @param {KeyboardEvent} e */
		const onKey = (e) => {
			if (e.key === 'Escape') menuOpen = false;
		};
		document.addEventListener('pointerdown', onDown);
		document.addEventListener('keydown', onKey);
		return () => {
			document.removeEventListener('pointerdown', onDown);
			document.removeEventListener('keydown', onKey);
		};
	});

	let countAlle = $derived(mahnwesenStore.klassen.reduce((sum, k) => sum + k.schueler.length, 0));

	let countAkut = $derived(
		mahnwesenStore.klassen.reduce(
			(sum, k) =>
				sum +
				k.schueler.filter((s) => {
					const isLehrer = s.klasse && s.klasse.toLowerCase() === 'lehrer';
					const maxTage = s.medien.reduce(
						(max, m) => (m.tage_ueberfaellig > max ? m.tage_ueberfaellig : max),
						0
					);
					return maxTage > 0 && maxTage <= 14 && !isLehrer;
				}).length,
			0
		)
	);

	let countEskaliert = $derived(
		mahnwesenStore.klassen.reduce(
			(sum, k) =>
				sum +
				k.schueler.filter((s) => {
					const isLehrer = s.klasse && s.klasse.toLowerCase() === 'lehrer';
					const maxTage = s.medien.reduce(
						(max, m) => (m.tage_ueberfaellig > max ? m.tage_ueberfaellig : max),
						0
					);
					return maxTage > 14 && !isLehrer;
				}).length,
			0
		)
	);

	let countKollegium = $derived(
		mahnwesenStore.klassen.reduce(
			(sum, k) =>
				sum + k.schueler.filter((s) => s.klasse && s.klasse.toLowerCase() === 'lehrer').length,
			0
		)
	);
</script>

<div class="flex items-center justify-between gap-4">
	<div class="min-w-0">
		<h1 class="text-2xl font-bold text-slate-800">Mahnwesen</h1>
		<p class="text-sm text-slate-500 mt-0.5">Überfällige Ausleihen nach Klassen sortiert.</p>
	</div>

	<!-- Rechte Aktionsleiste. Bei einer Auswahl übernimmt sie den Auswahl-Modus (wie Gmail/Drive):
	     Nur noch die auf die Markierung bezogenen Aktionen sind sichtbar. -->
	<div class="flex items-center gap-2 print:hidden shrink-0">
		{#if mahnwesenStore.selectedIds.size > 0}
			<button
				onclick={mahnwesenStore.deselectAllSchueler}
				aria-label="Auswahl aufheben"
				title="Auswahl aufheben"
				class="p-2 rounded-xl border border-slate-200 bg-white text-slate-500 hover:bg-slate-50 hover:text-slate-700 transition-colors"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
			<span class="text-sm font-semibold text-slate-700"
				>{mahnwesenStore.selectedIds.size} ausgewählt</span
			>
			<button
				onclick={mahnwesenStore.printSelectedMahnungen}
				disabled={mahnwesenStore.pdfLoading}
				class="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold flex items-center gap-1.5 shadow-sm transition-colors"
			>
				{#if mahnwesenStore.pdfLoading}
					<div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
				{:else}
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
					</svg>
				{/if}
				Mahnbriefe drucken
			</button>
		{:else}
			<div class="flex items-center gap-1 bg-slate-100 p-1 rounded-xl">
				<button
					class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode ===
					'datum'
						? 'bg-white text-slate-800 shadow-sm'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => {
						mahnwesenStore.mahnMode = 'datum';
						mahnwesenStore.fetchData();
					}}
				>
					Datum
				</button>
				<button
					class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode ===
					'jahrgang'
						? 'bg-white text-slate-800 shadow-sm'
						: 'text-slate-500 hover:text-slate-700'}"
					onclick={() => {
						mahnwesenStore.mahnMode = 'jahrgang';
						mahnwesenStore.fetchData();
					}}
				>
					Jahrgang
				</button>
			</div>

			<button
				onclick={mahnwesenStore.fetchData}
				aria-label="Daten neu laden"
				title="Neu laden"
				class="p-2 rounded-xl border border-slate-200 bg-white text-slate-500 hover:bg-slate-50 hover:text-slate-700 transition-colors"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
					/>
				</svg>
			</button>

			<!-- Mahnbriefe: eine Primäraktion (Eltern-Briefe drucken) + Menü mit Geltungsbereich.
			     Ersetzt die vier früher verstreuten PDF-Wege. Drucker-/Dokument-Icons — kein Umschlag,
			     denn hier wird nichts gemailt (nur PDF erzeugt). -->
			<div class="relative" bind:this={menuAnchor}>
				<div class="inline-flex rounded-xl shadow-sm">
					<button
						onclick={mahnwesenStore.downloadElternPDF}
						disabled={mahnwesenStore.elternPdfLoading}
						class="px-4 py-2 rounded-l-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold flex items-center gap-1.5 transition-colors"
					>
						{#if mahnwesenStore.elternPdfLoading}
							<div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
						{:else}
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
							</svg>
						{/if}
						Mahnbriefe
					</button>
					<button
						onclick={() => (menuOpen = !menuOpen)}
						aria-haspopup="menu"
						aria-expanded={menuOpen}
						aria-label="Weitere Druck- und Export-Optionen"
						class="px-2 py-2 rounded-r-xl bg-blue-600 hover:bg-blue-700 text-white border-l border-white/25 flex items-center transition-colors"
					>
						<svg
							class="h-3.5 w-3.5 transition-transform {menuOpen ? 'rotate-180' : ''}"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2.5"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
						</svg>
					</button>
				</div>

				{#if menuOpen}
					<div
						role="menu"
						tabindex="-1"
						transition:scale={{ duration: 130, start: 0.95, opacity: 0 }}
						class="absolute right-0 top-full mt-2 w-72 origin-top-right bg-white border border-slate-200 rounded-2xl shadow-xl p-2 z-30"
					>
						<div class="px-2 pt-1 pb-1 text-[10px] font-bold text-slate-400 uppercase tracking-wider">
							Mahnbriefe an Eltern
						</div>
						<button
							role="menuitem"
							onclick={() => {
								menuOpen = false;
								mahnwesenStore.downloadElternPDF();
							}}
							disabled={mahnwesenStore.elternPdfLoading}
							class="w-full text-left px-3 py-2.5 rounded-xl hover:bg-slate-50 disabled:opacity-50 text-sm font-semibold text-slate-700 flex items-center gap-2.5"
						>
							<svg class="h-4 w-4 text-slate-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
							</svg>
							Alle überfälligen
						</button>
						<div class="px-3 pt-2 pb-1">
							<div class="text-[10px] font-semibold text-slate-400 mb-1.5">Ganze Klasse</div>
							<div class="flex items-center gap-2">
								<select
									bind:value={mahnwesenStore.selectedKlasse}
									class="flex-1 min-w-0 bg-slate-50 border border-slate-200 rounded-lg text-xs font-bold text-slate-700 px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
								>
									<option value="">Klasse wählen …</option>
									{#each mahnwesenStore.klassen as k, _i (_i)}
										<option value={k.klasse}>{k.klasse}</option>
									{/each}
								</select>
								<button
									onclick={() => {
										mahnwesenStore.downloadKlassePDF(mahnwesenStore.selectedKlasse);
										menuOpen = false;
									}}
									disabled={mahnwesenStore.klassePdfLoading || !mahnwesenStore.selectedKlasse}
									class="shrink-0 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-slate-200 disabled:text-slate-400 text-white text-xs font-bold transition-colors"
								>
									PDF
								</button>
							</div>
						</div>

						<div class="border-t border-slate-100 my-1.5"></div>
						<div class="px-2 pt-1 pb-1 text-[10px] font-bold text-slate-400 uppercase tracking-wider">
							Weitere
						</div>
						<button
							role="menuitem"
							onclick={() => {
								menuOpen = false;
								mahnwesenStore.downloadPDF();
							}}
							disabled={mahnwesenStore.pdfLoading}
							class="w-full text-left px-3 py-2.5 rounded-xl hover:bg-slate-50 disabled:opacity-50 text-sm font-semibold text-slate-700 flex items-center gap-2.5"
						>
							<svg class="h-4 w-4 text-slate-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
							</svg>
							Übersichtsliste (PDF)
						</button>
						<button
							role="menuitem"
							onclick={() => {
								menuOpen = false;
								window.print();
							}}
							class="w-full text-left px-3 py-2.5 rounded-xl hover:bg-slate-50 text-sm font-semibold text-slate-700 flex items-center gap-2.5"
						>
							<svg class="h-4 w-4 text-slate-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
							</svg>
							Diese Seite drucken
						</button>
					</div>
				{/if}
			</div>

			<!-- „Alle anmahnen" ist die EINZIGE echte E-Mail-Aktion → nur hier das Umschlag-Icon. -->
			{#if countAlle > 0}
				<button
					onclick={() => {
						if (
							window.confirm(
								'Achtung: Bist du sicher, dass du jetzt Mahnungen an ALLE säumigen Schüler per E-Mail versenden möchtest?'
							)
						) {
							mahnwesenStore.sendBulkOverdueMails();
						}
					}}
					aria-label="Mahnlisten aller Klassen per E-Mail an die Klassenleitungen senden"
					class="px-4 py-2 rounded-xl bg-red-600 hover:bg-red-700 text-white text-xs font-bold transition-all flex items-center gap-1.5 shadow-sm"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
					</svg>
					Alle anmahnen
				</button>
			{/if}
		{/if}
	</div>
</div>

<!-- Tabs Navigation -->
{#if mahnwesenStore.data && !mahnwesenStore.loading}
	<div class="flex space-x-1 border-b border-gray-200 mt-6 print:hidden">
		<!-- Alle Tab -->
		<button
			class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter ===
			'Alle'
				? 'border-b-2 border-blue-600 text-blue-600'
				: 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
			onclick={() => (mahnwesenStore.activeFilter = 'Alle')}
		>
			Alle
			<span
				class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter ===
					'Alle' && countAlle > 0
					? 'bg-blue-100 text-blue-600'
					: 'bg-gray-100 text-gray-600'}"
			>
				{countAlle}
			</span>
		</button>

		<!-- Akut fällig Tab -->
		<button
			class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter ===
			'1. Erinnerung'
				? 'border-b-2 border-blue-600 text-blue-600'
				: 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
			onclick={() => (mahnwesenStore.activeFilter = '1. Erinnerung')}
		>
			Akut fällig
			<span
				class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter ===
					'1. Erinnerung' && countAkut > 0
					? 'bg-blue-100 text-blue-600'
					: 'bg-gray-100 text-gray-600'}"
			>
				{countAkut}
			</span>
		</button>

		<!-- Eskaliert Tab -->
		<button
			class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter ===
			'Mahnung'
				? 'border-b-2 border-blue-600 text-blue-600'
				: 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
			onclick={() => (mahnwesenStore.activeFilter = 'Mahnung')}
		>
			Eskaliert
			<span
				class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter ===
					'Mahnung' && countEskaliert > 0
					? 'bg-blue-100 text-blue-600'
					: 'bg-gray-100 text-gray-600'}"
			>
				{countEskaliert}
			</span>
		</button>

		<!-- Kollegium Tab -->
		<button
			class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter ===
			'Lehrerkollegium'
				? 'border-b-2 border-blue-600 text-blue-600'
				: 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
			onclick={() => (mahnwesenStore.activeFilter = 'Lehrerkollegium')}
		>
			Kollegium
			<span
				class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter ===
					'Lehrerkollegium' && countKollegium > 0
					? 'bg-blue-100 text-blue-600'
					: 'bg-gray-100 text-gray-600'}"
			>
				{countKollegium}
			</span>
		</button>
	</div>
{/if}

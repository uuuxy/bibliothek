<script>
	import { apiFetch } from './apiFetch.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import OverdueWidget from './OverdueWidget.svelte';
	import StatsTrendChart from './components/stats/StatsTrendChart.svelte';

	// State Runes (Svelte 5)
	/** @type {any} */
	let stats = $state(null);
	let loading = $state(true);
	let selectedTimeframe = $state('all');
	/** Bestandsfilter: '' = Gesamt, 'freihand' = Schülerbücherei, 'lmf' = Lernmittel */
	let selectedType = $state('');

	/** Drill-Down: navigiert auf die eigene Detailseite (deep-linkbar), kein Slide-in mehr.
	 *  @param {'renner' | 'ladenhueter'} kind */
	function openDetail(kind) {
		uiStore.statsDetailKind = kind;
		uiStore.activeTab = 'stats_detail';
	}

	const TIMEFRAMES = [
		{ value: 'all', label: 'Alle' },
		{ value: 'schuljahr', label: 'Schuljahr' },
		{ value: 'monat', label: 'Monat' }
	];

	const BESTAND_TYPES = [
		{ value: '', label: 'Gesamt' },
		{ value: 'freihand', label: 'Freihand' },
		{ value: 'lmf', label: 'LMF' }
	];

	/** @param {number} v */
	const euro = (v) => (v ?? 0).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });

	/** Ganzzahl mit deutscher Tausender-Trennung (33166 → „33.166"). @param {number} v */
	const num = (v) => (v ?? 0).toLocaleString('de-DE');

	// Kacheln zeigen Top 5; das Drill-Down-Panel filtert die volle Liste clientseitig
	const topRenner = $derived(stats?.popular_titles?.slice(0, 5) ?? []);
	const topWarmers = $derived(stats?.shelf_warmers?.slice(0, 5) ?? []);

	// Farbe folgt dem Wert: „Schaden"-Kennzahlen werden erst rot/gelb, wenn es wirklich
	// etwas zu melden gibt. Bei 0 bleiben sie ruhig-grün — so behält Rot seine Signalwirkung
	// und der Bestzustand sieht nicht wie eine Alarmtafel aus.
	const verlusteFarbe = $derived(
		(stats?.loss_stats?.verlorene_exemplare ?? 0) > 0 ? 'text-rose-600' : 'text-emerald-600'
	);
	const verlustquoteFarbe = $derived.by(() => {
		const q = stats?.loss_stats?.verlust_quote ?? 0;
		if (q <= 0) return 'text-emerald-600';
		if (q < 5) return 'text-amber-600';
		return 'text-rose-600';
	});
	const wiederbeschaffungFarbe = $derived(
		(stats?.wiederbeschaffungswert_defekt ?? 0) > 0 ? 'text-rose-700' : 'text-emerald-600'
	);

	// Zweiter, farbunabhängiger Kanal (WCAG 1.4.1): Bei Handlungsbedarf erscheint zusätzlich
	// ein Warn-Icon. Im Bestzustand (0) bleibt es icon-frei — die Ampelfarbe steht dann nicht
	// mehr allein für die Aussage; die Zahl selbst und das (fehlende) Icon tragen sie mit.
	const verlusteStatus = $derived((stats?.loss_stats?.verlorene_exemplare ?? 0) > 0 ? 'warn' : null);
	const verlustquoteStatus = $derived((stats?.loss_stats?.verlust_quote ?? 0) > 0 ? 'warn' : null);
	const wiederbeschaffungStatus = $derived(
		(stats?.wiederbeschaffungswert_defekt ?? 0) > 0 ? 'warn' : null
	);

	// Fetch statistics from backend API.
	// limit=100 lädt die Drill-Down-Daten gleich mit — das Panel braucht
	// dadurch keinen einzigen weiteren API-Call.
	async function fetchStats() {
		loading = true;
		try {
			// eslint-disable-next-line svelte/prefer-svelte-reactivity
			const params = new URLSearchParams({ limit: '100' });
			if (selectedTimeframe !== 'all') params.set('zeitraum', selectedTimeframe);
			if (selectedType) params.set('type', selectedType);
			const res = await apiFetch(`/api/statistiken?${params}`);
			if (!res.ok) throw new Error('Fehler beim Laden');
			stats = await res.json();
		} catch (err) {
			console.error('Stats loading error:', err);
		} finally {
			loading = false;
		}
	}

	// Re-fetch whenever timeframe or Bestandsfilter changes
	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		selectedTimeframe; // track dependency
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		selectedType;
		fetchStats();
	});
</script>

{#snippet kennzahl(label, value, hint, valueClass, status = null)}
	<div
		class="p-6 flex flex-col justify-between space-y-2 text-left border border-gray-200 sm:border-0 sm:border-l sm:first:border-l-0"
	>
		<span class="text-sm font-semibold uppercase tracking-wider text-slate-400 font-sans"
			>{label}</span
		>
		<span class="text-4xl font-extrabold {valueClass} leading-none py-1 flex items-center gap-2">
			{#if status === 'warn'}
				<!-- Warn-Dreieck: farbunabhängiger Zweitkanal (WCAG 1.4.1), nur bei Handlungsbedarf -->
				<svg class="w-6 h-6 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 9v3.75m0 3.75h.01M10.29 3.86l-8.48 14.7A1.5 1.5 0 003.11 21h17.78a1.5 1.5 0 001.3-2.44l-8.48-14.7a1.5 1.5 0 00-2.6 0z"
					/>
				</svg>
			{/if}
			<span>{value}</span>
		</span>
		<span class="text-sm text-slate-500 font-medium">{hint}</span>
	</div>
{/snippet}

{#snippet drillDownHeader(label, panel)}
	<button
		onclick={() => openDetail(panel)}
		class="w-full flex items-center justify-between gap-3 border-b border-slate-100 pb-2 -mx-2 px-2 rounded-md group cursor-pointer text-left hover:bg-slate-50 transition-colors"
		aria-label="{label} — Detailansicht öffnen"
	>
		<h3
			class="font-bold text-slate-700 text-sm uppercase tracking-wider font-sans group-hover:text-slate-900 transition-colors"
		>
			{label}
		</h3>
		<!-- Klar als klickbar erkennbar: dauerhaft blaues Pill statt zartem Hover-Grau.
		     Chevron (→) statt Außen-Pfeil (↗), weil es jetzt auf eine interne Seite navigiert. -->
		<span
			class="shrink-0 flex items-center gap-1 text-[11px] font-bold text-blue-600 bg-blue-50 group-hover:bg-blue-100 px-2.5 py-1 rounded-full transition-colors"
		>
			Alle anzeigen
			<svg
				class="w-3 h-3 transition-transform group-hover:translate-x-0.5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" /></svg
			>
		</span>
	</button>
{/snippet}

<div class="w-full space-y-6 text-slate-800">
	<!-- Header: Bestandsfilter + Zeitraum -->
	<div
		class="flex flex-col md:flex-row md:items-center md:justify-end gap-4 border-b border-slate-100 pb-5"
	>
		<!-- Bestandsfilter (LMF / Freihand / Gesamt) — filtert serverseitig ALLE Kennzahlen -->
		<div class="flex items-center gap-2 self-start md:self-center">
			<span class="text-sm font-semibold text-slate-400 uppercase tracking-wider font-sans"
				>Bestand:</span
			>
			<div class="flex bg-slate-100 p-0.5 rounded-xl border border-slate-200">
				{#each BESTAND_TYPES as bt, _i (_i)}
					<button
						onclick={() => (selectedType = bt.value)}
						class="px-4 py-1.5 text-sm font-bold rounded-lg cursor-pointer transition-all {selectedType ===
						bt.value
							? 'bg-white text-slate-900 shadow-xs'
							: 'text-slate-500 hover:text-slate-700'}">{bt.label}</button
					>
				{/each}
			</div>
		</div>

		<!-- Time Filter Buttons -->
		<div class="flex items-center gap-2 self-start md:self-center">
			<span class="text-sm font-semibold text-slate-400 uppercase tracking-wider font-sans"
				>Zeitraum:</span
			>
			<div class="flex bg-slate-100 p-0.5 rounded-xl border border-slate-200">
				{#each TIMEFRAMES as tf, _i (_i)}
					<button
						onclick={() => (selectedTimeframe = tf.value)}
						class="px-4 py-1.5 text-sm font-bold rounded-lg cursor-pointer transition-all {selectedTimeframe ===
						tf.value
							? 'bg-white text-slate-900 shadow-xs'
							: 'text-slate-500 hover:text-slate-700'}">{tf.label}</button
					>
				{/each}
			</div>
		</div>
	</div>

	{#if loading}
		<div class="py-12 flex justify-center items-center">
			<div
				class="w-8 h-8 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin"
			></div>
		</div>
	{:else if stats}
		<!-- Kennzahlen: zwei flache Dreierreihen (Bestand/Zirkulation · Verluste/Finanzen) -->
		<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
			{@render kennzahl(
				'Gesamtbestand',
				num(stats.loss_stats.gesamt_bestand),
				'Physische Buchkopien im System',
				'text-slate-900'
			)}
			{@render kennzahl(
				'Aktuell verliehen',
				num(stats.zirkulation?.aktuell_verliehen ?? 0),
				`von ${num(stats.zirkulation?.aktiver_bestand ?? 0)} aktiven Exemplaren`,
				'text-blue-600'
			)}
			<!-- Momentaufnahme, keine echte Zeitraum-Quote: neutral gefärbt statt grün, damit
			     kein „gut/schlecht" suggeriert wird (5 % ist für eine Bibliothek nicht per se gut). -->
			{@render kennzahl(
				'Zirkulationsquote',
				`${stats.zirkulationsquote ?? 0}%`,
				'Momentaufnahme: verliehen ÷ aktiver Bestand',
				'text-slate-900'
			)}
		</div>
		<div class="grid grid-cols-1 sm:grid-cols-3 gap-4 border-t border-slate-100">
			{@render kennzahl(
				'Verlorene / Defekte Bücher',
				num(stats.loss_stats.verlorene_exemplare),
				'Exemplare mit Schadensfällen',
				verlusteFarbe,
				verlusteStatus
			)}
			{@render kennzahl(
				'Verlustquote',
				`${stats.loss_stats.verlust_quote}%`,
				'Prozentsatz verlorener Lehrmittel',
				verlustquoteFarbe,
				verlustquoteStatus
			)}
			{@render kennzahl(
				'Wiederbeschaffungswert',
				euro(stats.wiederbeschaffungswert_defekt),
				'Einkaufspreise verlorener/defekter Exemplare',
				wiederbeschaffungFarbe,
				wiederbeschaffungStatus
			)}
		</div>

		<!-- Aktivitäts-Zeitreihe: gibt der Seite die fehlende Zeitdimension (Trend) -->
		<div class="pt-6 border-t border-slate-100">
			<StatsTrendChart data={stats.monats_trend ?? []} />
		</div>

		<!-- Stats Tables Layout -->
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6 pt-6">
			<!-- Overdue Widget -->
			<div class="space-y-3 text-left h-full">
				<OverdueWidget />
			</div>
			<!-- Top Borrowed Books Section ("Die Renner") -->
			<div class="space-y-3 text-left">
				{@render drillDownHeader('Beliebteste Titel (Die Renner)', 'renner')}

				<div class="w-full">
					<table class="w-full text-left text-base border-collapse">
						<thead>
							<tr
								class="bg-slate-50 border-b border-slate-100 text-sm font-bold text-slate-400 font-sans uppercase tracking-wider"
							>
								<th class="py-3 px-4">Buchtitel</th>
								<th class="py-3 px-4 text-right">Ausleihen</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-slate-100 text-sm text-slate-650 font-semibold">
							{#if !stats.popular_titles || stats.popular_titles.length === 0}
								<tr>
									<td colspan="2" class="py-12 text-center text-xs text-slate-400 font-medium">
										<span class="text-2xl block mb-2">📊</span>
										Noch keine Ausleihen registriert
									</td>
								</tr>
							{:else}
								{#each topRenner as book, _i (_i)}
									<tr class="hover:bg-slate-50/50 transition-colors">
										<td class="py-3 px-4 flex items-center gap-3">
											<!-- Cover Thumbnail -->
											{#if book.cover_url}
												<img
													src={book.cover_url}
													alt="Cover"
													class="w-10 aspect-3/4 object-cover rounded shadow-sm border border-slate-100/50 shrink-0"
												/>
											{:else}
												<div
													class="w-10 aspect-3/4 bg-slate-50 border border-slate-150 rounded flex items-center justify-center text-slate-400 text-xs shadow-sm shrink-0 font-medium"
												>
													📖
												</div>
											{/if}

											<!-- Title & Author -->
											<div class="min-w-0">
												<span
													class="font-bold text-slate-800 text-sm truncate block"
													title={book.titel}>{book.titel}</span
												>
												<span
													class="text-slate-450 text-xs block font-medium truncate"
													title={book.autor}>{book.autor}</span
												>
											</div>
										</td>
										<td class="py-3 px-4 text-slate-900 font-bold text-right shrink-0">
											{book.count}x geliehen
										</td>
									</tr>
								{/each}
							{/if}
						</tbody>
					</table>
				</div>
			</div>

			<!-- Shelf Warmers Table -->
			<div class="space-y-3 text-left">
				{@render drillDownHeader('Ladenhüter', 'ladenhueter')}

				<div class="w-full">
					<table class="w-full text-left text-base border-collapse">
						<thead>
							<tr
								class="bg-slate-50 border-b border-slate-100 text-sm font-bold text-slate-400 font-sans uppercase tracking-wider"
							>
								<th class="py-3 px-4">Buchtitel</th>
								<th class="py-3 px-4">Autor</th>
								<th class="py-3 px-4 text-right">Zuletzt geliehen</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-slate-100 text-sm text-slate-650 font-semibold">
							{#if !stats.shelf_warmers || stats.shelf_warmers.length === 0}
								<!-- „Keine Ladenhüter" ist ein GUTER Zustand (kein toter Bestand): ruhiges Grün
								     statt des früheren 🕳️-Emojis, das wie ein Render-Artefakt/Tintenklecks wirkte. -->
								<tr>
									<td colspan="3" class="py-12 text-center text-xs text-slate-400 font-medium">
										<svg
											class="w-7 h-7 mx-auto mb-2 text-emerald-500"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
											aria-hidden="true"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
											/></svg
										>
										Keine Ladenhüter — der Bestand ist in Bewegung.
									</td>
								</tr>
							{:else}
								{#each topWarmers as book, _i (_i)}
									<tr class="hover:bg-slate-50/50 transition-colors">
										<td
											class="py-3.5 px-4 text-slate-800 font-bold truncate max-w-40"
											title={book.titel}>{book.titel}</td
										>
										<td class="py-3.5 px-4 text-slate-500 truncate max-w-30" title={book.autor}
											>{book.autor}</td
										>
										<td class="py-3.5 px-4 text-amber-600 font-bold text-right"
											>{book.letzte_aus}</td
										>
									</tr>
								{/each}
							{/if}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}
</div>

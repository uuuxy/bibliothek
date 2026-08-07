<script>
	import { apiFetch } from './apiFetch.js';
	import { coverSrc } from './utils/coverSrc.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import OverdueWidget from './OverdueWidget.svelte';
	import StatsTrendChart from './components/stats/StatsTrendChart.svelte';
	import Button from './components/ui/Button.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import { ChevronRight, CircleCheck, TriangleAlert } from '@lucide/svelte';

	// State Runes (Svelte 5)
	/** @type {any} */
	let stats = $state(null);
	let loading = $state(true);
	let selectedTimeframe = $state('all');
	/** Bestandsfilter: '' = Gesamt, 'freihand' = Schülerbücherei, 'lmf' = Lernmittel */
	let selectedType = $state('');
	/** Segmented Control der „Bestands-Analysen"-Card. @type {'renner' | 'ladenhueter'} */
	let analyse = $state('renner');

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

	// Die beiden Sichten der „Bestands-Analysen"-Card. detailLabel ist der Name, unter dem
	// die Detailseite firmiert — er trägt auch das aria-label des Drill-Down-Buttons.
	const ANALYSEN = [
		{ value: 'renner', label: 'Renner', detailLabel: 'Beliebteste Titel (Die Renner)' },
		{ value: 'ladenhueter', label: 'Ladenhüter', detailLabel: 'Ladenhüter' }
	];
	const aktiveAnalyse = $derived(ANALYSEN.find((a) => a.value === analyse) ?? ANALYSEN[0]);

	/** @param {number} v */
	const euro = (v) => (v ?? 0).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });

	/** Ganzzahl mit deutscher Tausender-Trennung (33166 → „33.166"). @param {number} v */
	const num = (v) => (v ?? 0).toLocaleString('de-DE');

	// Die Card zeigt Top 8 (eine Tabelle statt zwei → mehr Platz pro Zeile);
	// die volle Liste liegt auf der Detailseite.
	const topRenner = $derived(stats?.popular_titles?.slice(0, 8) ?? []);
	const topWarmers = $derived(stats?.shelf_warmers?.slice(0, 8) ?? []);
	const aktiveListe = $derived(analyse === 'renner' ? topRenner : topWarmers);

	// Farbe folgt dem Wert: „Schaden"-Kennzahlen werden erst rot, wenn es wirklich etwas zu
	// melden gibt. Bei 0 bleiben sie ruhig-grün — so behält Rot seine Signalwirkung und der
	// Bestzustand sieht nicht wie eine Alarmtafel aus.
	const verlusteFarbe = $derived(
		(stats?.loss_stats?.verlorene_exemplare ?? 0) > 0 ? 'text-rose-600' : 'text-emerald-600'
	);

	// Zweiter, farbunabhängiger Kanal (WCAG 1.4.1): Bei Handlungsbedarf erscheint zusätzlich
	// ein Warn-Icon. Im Bestzustand (0) bleibt es icon-frei — die Ampelfarbe steht dann nicht
	// mehr allein für die Aussage; die Zahl selbst und das (fehlende) Icon tragen sie mit.
	const verlusteStatus = $derived(
		(stats?.loss_stats?.verlorene_exemplare ?? 0) > 0 ? 'warn' : null
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

{#snippet warnIcon(klasse)}
	<!-- Warn-Dreieck: farbunabhängiger Zweitkanal (WCAG 1.4.1), nur bei Handlungsbedarf -->
	<TriangleAlert class="w-5 h-5" aria-hidden="true" />
{/snippet}

<!-- Segmented Control (Material 3): eine Pillen-Gruppe, aktives Segment als weiße Kapsel.
     Ein Snippet für alle drei Vorkommen (Bestand, Zeitraum, Analyse-Umschalter). -->
{#snippet pills(items, current, select, ariaLabel)}
	<div
		class="flex items-center bg-slate-100 p-1 rounded-full border border-slate-200/70"
		role="group"
		aria-label={ariaLabel}
	>
		{#each items as it (it.value)}
			<button
				type="button"
				onclick={() => select(it.value)}
				aria-pressed={current === it.value}
				class="px-3.5 py-1 text-xs font-bold rounded-full cursor-pointer transition-all whitespace-nowrap {current ===
				it.value
					? 'bg-white text-slate-900 shadow-xs'
					: 'text-slate-500 hover:text-slate-800'}">{it.label}</button
			>
		{/each}
	</div>
{/snippet}

<!-- KPI-Kachel: Zahl groß und dünn, Label winzig und fett (Material-3-Typografie). -->
{#snippet kpi(label, value, hint, valueClass, status = /** @type {'warn' | null} */ (null))}
	<div
		class="bg-white rounded-xl border border-slate-200/80 shadow-sm p-5 flex flex-col justify-between gap-3 text-left"
	>
		<span class="text-xs font-medium text-slate-500">{label}</span>
		<span
			class="text-4xl font-light tracking-tight tabular-nums leading-none flex items-center gap-2 {valueClass}"
		>
			{#if status === 'warn'}{@render warnIcon('w-6 h-6 shrink-0')}{/if}
			<span class="truncate">{value}</span>
		</span>
		<span class="text-xs text-slate-400 leading-snug">{hint}</span>
	</div>
{/snippet}

<!-- Kopfzeile jeder großen Card: Label links, optionale Aktionen rechts. -->
{#snippet cardTitel(label)}
	<h3 class="text-xs font-medium text-slate-500">{label}</h3>
{/snippet}

{#snippet drillDownButton()}
	<Button
		variant="secondary"
		size="sm"
		type="button"
		onclick={() => openDetail(analyse)}
		class="shrink-0 gap-1 border-blue-100 bg-blue-50 text-label-small text-blue-600 hover:bg-blue-100"
		aria-label="{aktiveAnalyse.detailLabel} — Detailansicht öffnen"
	>
		Alle anzeigen
		<ChevronRight class="w-3 h-3" aria-hidden="true" />
	</Button>
{/snippet}

<!-- Leerzustand als eigene Fläche statt als Tabellenzeile: nur so lässt er sich in der
     fixen Boxhöhe zentrieren (eine <td> würde oben kleben). -->
{#snippet leerFlaeche(inhalt)}
	<div
		class="h-full flex flex-col items-center justify-center text-center text-xs text-slate-400 font-medium"
	>
		{@render inhalt()}
	</div>
{/snippet}

{#snippet spaltenKopf(spalten)}
	<thead class="sticky top-0 z-10 bg-white">
		<tr class="text-label-small font-medium text-slate-400">
			{#each spalten as s (s.label)}
				<th class="py-2 px-4 font-bold {s.right ? 'text-right' : 'text-left'}">{s.label}</th>
			{/each}
		</tr>
	</thead>
{/snippet}

{#snippet keineAusleihen()}
	<span class="text-2xl block mb-2">📊</span>
	Noch keine Ausleihen registriert
{/snippet}

{#snippet keineLadenhueter()}
	<!-- „Keine Ladenhüter" ist ein GUTER Zustand (kein toter Bestand): ruhiges Grün. -->
	<CircleCheck class="w-7 h-7 mx-auto mb-2 text-emerald-500" aria-hidden="true" />
	Keine Ladenhüter — der Bestand ist in Bewegung.
{/snippet}

{#snippet rennerTabelle()}
	<table class="w-full border-collapse">
		{@render spaltenKopf([
			{ label: 'Buchtitel' },
			{ label: 'Autor' },
			{ label: 'Ausleihen', right: true }
		])}
		<tbody class="divide-y divide-slate-100">
			{#each topRenner as book (book.id)}
				<tr class="hover:bg-slate-50 transition-colors">
					<td class="py-2 px-4">
						<div class="flex items-center gap-3 min-w-0">
							{#if coverSrc(book.cover_url, book.isbn)}
								<img
									src={coverSrc(book.cover_url, book.isbn)}
									alt=""
									class="w-8 aspect-3/4 object-cover rounded shadow-xs border border-slate-100 shrink-0"
								/>
							{:else}
								<div
									class="w-8 aspect-3/4 bg-slate-50 border border-slate-200 rounded flex items-center justify-center text-slate-300 text-xs shrink-0"
								>
									📖
								</div>
							{/if}
							<span class="font-semibold text-slate-800 text-sm truncate" title={book.titel}
								>{book.titel}</span
							>
						</div>
					</td>
					<td class="py-2 px-4 text-sm text-slate-500 truncate max-w-48" title={book.autor}
						>{book.autor}</td
					>
					<td class="py-2 px-4 text-sm font-semibold text-slate-900 tabular-nums text-right"
						>{num(book.count)}×</td
					>
				</tr>
			{/each}
		</tbody>
	</table>
{/snippet}

{#snippet ladenhueterTabelle()}
	<table class="w-full border-collapse">
		{@render spaltenKopf([
			{ label: 'Buchtitel' },
			{ label: 'Autor' },
			{ label: 'Zuletzt geliehen', right: true }
		])}
		<tbody class="divide-y divide-slate-100">
			{#each topWarmers as book (book.id)}
				<tr class="hover:bg-slate-50 transition-colors">
					<td
						class="py-3 px-4 text-sm font-semibold text-slate-800 truncate max-w-64"
						title={book.titel}>{book.titel}</td
					>
					<td class="py-3 px-4 text-sm text-slate-500 truncate max-w-48" title={book.autor}
						>{book.autor}</td
					>
					<td class="py-3 px-4 text-sm font-semibold text-amber-600 tabular-nums text-right"
						>{book.letzte_aus}</td
					>
				</tr>
			{/each}
		</tbody>
	</table>
{/snippet}

<!-- Skeleton in der Geometrie des fertigen Layouts: beim Filterwechsel springt nichts. -->
{#snippet skeleton()}
	<div class="flex-1 min-h-0 flex flex-col gap-4 animate-pulse" aria-hidden="true">
		<div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
			{#each [0, 1, 2, 3] as i (i)}
				<div class="h-28 bg-white rounded-xl border border-slate-200/80"></div>
			{/each}
		</div>
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
			<div class="lg:col-span-2 h-72 bg-white rounded-xl border border-slate-200/80"></div>
			<div class="h-72 bg-white rounded-xl border border-slate-200/80"></div>
		</div>
		<div class="flex-1 min-h-48 bg-white rounded-xl border border-slate-200/80"></div>
	</div>
{/snippet}

<!-- Graue Fläche (Material-Kontrast): die weißen Cards heben sich davon ab (Bento-Struktur).
     Das Dashboard ist eine Höhen-Flexbox: die Bento-Reihen oben haben feste Höhen, die
     „Bestands-Analysen"-Card unten nimmt den Rest. Dadurch endet das Layout genau am
     unteren Viewport-Rand statt in einen Scroll-Wurm auszulaufen.
     flex-1/min-h-0 statt min-h-full: Die Deckelung braucht eine DEFINITE Höhe. min-height:100%
     setzt nur eine Untergrenze — die Card unten wäre trotzdem auf Inhaltshöhe gewachsen und
     hätte die Seite wieder scrollen lassen. Die Höhe kommt aus der Flex-Kette des Routers. -->
<PageShell
	breite="voll"
	titel="Statistiken"
	beschreibung="Bestand, Ausleihen und Verluste im Überblick."
>
	<div class="flex-1 min-h-0 flex flex-col gap-4">
		<!-- Filterleiste: kompakt in EINER Zeile, direkt auf der grauen Fläche. -->
		<div class="shrink-0 flex flex-wrap items-center justify-end gap-x-6 gap-y-3">
			<div class="flex items-center gap-2">
				<span class="text-xs font-medium text-slate-500">Bestand</span>
				{@render pills(BESTAND_TYPES, selectedType, (v) => (selectedType = v), 'Bestand filtern')}
			</div>
			<div class="flex items-center gap-2">
				<span class="text-xs font-medium text-slate-500">Zeitraum</span>
				{@render pills(
					TIMEFRAMES,
					selectedTimeframe,
					(v) => (selectedTimeframe = v),
					'Zeitraum filtern'
				)}
			</div>
		</div>

		{#if loading}
			{@render skeleton()}
		{:else if stats}
			<!-- 1) KPI-Reihe: vier gleichwertige Kacheln in EINEM durchgehenden 4er-Grid. -->
			<div class="shrink-0 grid grid-cols-2 lg:grid-cols-4 gap-4">
				{@render kpi(
					'Gesamtbestand',
					num(stats.loss_stats.gesamt_bestand),
					'Physische Buchkopien im System',
					'text-slate-900'
				)}
				{@render kpi(
					'Aktuell verliehen',
					num(stats.zirkulation?.aktuell_verliehen ?? 0),
					`von ${num(stats.zirkulation?.aktiver_bestand ?? 0)} aktiven Exemplaren`,
					'text-blue-600'
				)}
				<!-- Momentaufnahme, keine echte Zeitraum-Quote: neutral gefärbt statt grün, damit
				     kein „gut/schlecht" suggeriert wird (5 % ist für eine Bibliothek nicht per se gut). -->
				{@render kpi(
					'Zirkulationsquote',
					`${stats.zirkulationsquote ?? 0}%`,
					'verliehen ÷ aktiver Bestand',
					'text-slate-900'
				)}
				{@render kpi(
					'Verluste & Schäden',
					num(stats.loss_stats.verlorene_exemplare),
					`${stats.loss_stats.verlust_quote}% Quote · ${euro(stats.wiederbeschaffungswert_defekt)} Wiederbeschaffungswert`,
					verlusteFarbe,
					verlusteStatus
				)}
			</div>

			<!-- 2) Asymmetrische Reihe: Chart (2 Spalten) + Mahnungs-Widget (1 Spalte).
			     Beide Cards teilen sich exakt dieselbe feste Höhe → die Reihe kippt nie. -->
			<div class="shrink-0 grid grid-cols-1 lg:grid-cols-3 gap-4">
				<div
					class="lg:col-span-2 bg-white rounded-xl border border-slate-200/80 shadow-sm p-5 h-72 flex flex-col"
				>
					<StatsTrendChart data={stats.monats_trend ?? []} />
				</div>
				<!-- Überfälligkeit NEUTRAL (kein Rot-Alarm): Analyse-Kontext, kein Einsatzleitstand. -->
				<div
					class="bg-white rounded-xl border border-slate-200/80 shadow-sm p-5 h-72 flex flex-col"
				>
					<OverdueWidget aktuellVerliehen={stats.zirkulation?.aktuell_verliehen ?? 0} />
				</div>
			</div>

			<!-- 3) Bestands-Analysen: EINE Card, Segmented Control schaltet Renner ↔ Ladenhüter.
			     flex-1: füllt die Resthöhe bis zum Viewport-Rand aus. -->
			<div
				class="flex-1 min-h-0 bg-white rounded-xl border border-slate-200/80 shadow-sm p-5 flex flex-col"
			>
				<div class="shrink-0 flex flex-wrap items-center justify-between gap-3 mb-3">
					{@render cardTitel('Bestands-Analysen')}
					<div class="flex items-center gap-2">
						{@render pills(ANALYSEN, analyse, (v) => (analyse = v), 'Analyse umschalten')}
						{@render drillDownButton()}
					</div>
				</div>

				<!-- Die Liste bekommt exakt die Resthöhe (nie mehr) und scrollt INNERHALB der Box —
				     die Kopfzeile bleibt dabei stehen. min-h-48 ist die Untergrenze: erst wenn das
				     Fenster darunter schrumpft, scrollt überhaupt die Seite. Feste Höhe statt max-h,
				     damit beim Umschalten Renner ↔ Ladenhüter nichts springt. -->
				<div class="flex-1 min-h-48 overflow-y-auto overflow-x-auto">
					{#if aktiveListe.length === 0}
						{@render leerFlaeche(analyse === 'renner' ? keineAusleihen : keineLadenhueter)}
					{:else if analyse === 'renner'}
						{@render rennerTabelle()}
					{:else}
						{@render ladenhueterTabelle()}
					{/if}
				</div>
			</div>
		{/if}
	</div>
</PageShell>

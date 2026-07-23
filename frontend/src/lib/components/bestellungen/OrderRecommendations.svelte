<script>
	let { recommendations, onAddToCart } = $props();

	// Nur die ersten Einträge ins DOM (Muster wie BookTable/Inventur-Startseite).
	// Der Bestellbedarf umfasst schnell den halben Katalog — jeder Titel unter seinem
	// Meldebestand landet hier. Alle zu rendern hiess: tausende DOM-Knoten und ebenso
	// viele Cover-Requests. Das Scrollfenster wächst mit dem Viewport, damit man die
	// dringenden Titel sieht statt drei Zeilen durch ein Guckloch.
	let maxVisible = $state(60);

	// Schnellfilter: bei 335 Titeln ist Suchen schneller als Scrollen.
	let filter = $state('');

	let gefiltert = $derived(
		filter.trim()
			? recommendations.filter((/** @type {any} */ r) => {
					const q = filter.trim().toLowerCase();
					return (
						(r.titel || '').toLowerCase().includes(q) ||
						(r.isbn || '').toLowerCase().includes(q) ||
						(r.verlag || '').toLowerCase().includes(q) ||
						(r.signatur || '').toLowerCase().includes(q)
					);
				})
			: recommendations
	);

	let sichtbare = $derived(gefiltert.slice(0, maxVisible));

	// Nach einem Datenwechsel (z. B. Wareneingang) oder neuem Filter wieder von vorn.
	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		recommendations, filter;
		maxVisible = 60;
	});

	/**
	 * Farbstufe einer Zeile. Die Liste ist bereits die Bestellbedarf-Liste (Backend:
	 * gesamt < konfigurierbare Schwelle). Herausgehoben wird nur der echte Notfall:
	 * 0 eigene Exemplare (Titel komplett weg) = kritisch, alles andere = knapp. Basis ist
	 * der Gesamtbestand (Besitz), nicht der Verfügbarbestand — ein verliehener Klassensatz
	 * (0 verfügbar, 30 vorhanden) taucht hier ohnehin nicht auf.
	 * @param {any} r
	 */
	function stufe(r) {
		return r.gesamt_bestand === 0 ? 'kritisch' : 'knapp';
	}

	let kritischeAnzahl = $derived(
		recommendations.filter((/** @type {any} */ r) => stufe(r) === 'kritisch').length
	);
</script>

<section class="bg-white rounded-2xl border border-slate-200/80 shadow-sm flex flex-col overflow-hidden">
	<!-- Header -->
	<header class="px-5 pt-5 pb-4 border-b border-slate-100">
		<div class="flex items-start justify-between gap-4">
			<div class="min-w-0">
				<h2 class="text-lg font-bold text-slate-900 tracking-tight flex items-center gap-2">
					Bestellbedarf
					{#if recommendations.length}
						<span
							class="text-xs font-bold text-slate-500 bg-slate-100 rounded-full px-2 py-0.5 tabular-nums"
							>{recommendations.length}</span
						>
					{/if}
				</h2>
				{#if kritischeAnzahl}
					<p class="text-[13px] font-semibold text-rose-600 mt-1 flex items-center gap-1.5">
						<span class="w-1.5 h-1.5 rounded-full bg-rose-500"></span>
						{kritischeAnzahl}× komplett fehlend · 0 Exemplare
					</p>
				{:else if recommendations.length}
					<p class="text-[13px] text-slate-400 mt-1">Alle unter der Bestellbedarf-Schwelle.</p>
				{/if}
			</div>
			<a
				href="/api/bestellungen/pdf"
				download
				class="shrink-0 flex items-center gap-2 text-xs font-bold text-slate-600 bg-slate-50 hover:bg-slate-100 border border-slate-200 px-3 py-2 rounded-xl transition-colors"
			>
				<svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
					/>
				</svg>
				<span class="hidden sm:inline">PDF-Bestellliste</span>
			</a>
		</div>

		{#if recommendations.length}
			<!-- Schnellfilter -->
			<div class="relative mt-4">
				<svg
					class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400 pointer-events-none"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
				<input
					type="search"
					bind:value={filter}
					placeholder="In {recommendations.length} Titeln filtern …"
					class="w-full pl-9 pr-3 py-2 rounded-xl bg-slate-50 border border-slate-200 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 focus:bg-white transition-all"
				/>
			</div>
		{/if}
	</header>

	<!-- List -->
	{#if !recommendations.length}
		<div class="flex flex-col items-center justify-center text-center py-16 px-6 text-slate-400">
			<span class="text-3xl mb-2">✅</span>
			<p class="text-sm font-semibold text-slate-500">Bestände ausreichend</p>
			<p class="text-xs mt-1">Kein Titel liegt unter der Bestellbedarf-Schwelle.</p>
		</div>
	{:else if !gefiltert.length}
		<div class="text-center py-14 px-6 text-slate-400">
			<p class="text-sm font-medium">Kein Treffer für <em>„{filter}"</em></p>
		</div>
	{:else}
		<div class="overflow-y-auto max-h-[calc(100vh-19rem)] px-3 py-3 space-y-1.5">
			{#each sichtbare as r, _i (_i)}
				<!-- Kein Zeilen-Alarm mehr: die Liste IST per Definition die Bestellbedarf-Liste
				     (sortiert nach Fehlbestand). Dringlichkeit trägt die farbige Zahl rechts
				     + das „Fehlt komplett"-Pill — nicht 335 rote Balken (Alarm-Müdigkeit). -->
				<div
					class="group flex items-center gap-3 rounded-xl border border-transparent px-3 py-2.5 hover:bg-slate-50 hover:border-slate-200 transition-colors"
				>
					<!-- Platzhalter liegt IMMER dahinter; das Cover legt sich drüber und blendet
					     sich bei Ladefehler aus (DNB liefert für viele LMF-Titel kein Bild —
					     sonst blieben leere Kästen zurück, die kaputt aussehen). -->
					<div class="relative w-10 aspect-3/4 shrink-0">
						<div
							class="absolute inset-0 rounded-md bg-slate-100 flex items-center justify-center text-slate-400 text-sm ring-1 ring-slate-200/70"
						>
							📖
						</div>
						{#if r.cover_url}
							<img
								src="/api/images/cover?isbn={r.isbn || ''}&url={encodeURIComponent(r.cover_url)}"
								class="absolute inset-0 w-full h-full object-cover rounded-md bg-white ring-1 ring-slate-200/70"
								loading="lazy"
								decoding="async"
								alt=""
								onload={(e) => {
									// Der Cover-Proxy liefert bei fehlendem Bild ein transparentes 1×1-GIF
									// (bewusst 200 statt 404, gegen Konsolen-Spam). Das ist KEIN Fehler,
									// onerror greift nicht — an der Winzgröße erkennen wir den Platzhalter.
									if (e.currentTarget.naturalWidth <= 1) e.currentTarget.style.display = 'none';
								}}
								onerror={(e) => (e.currentTarget.style.display = 'none')}
							/>
						{/if}
					</div>

					<div class="min-w-0 flex-1">
						<h4 class="font-semibold text-slate-900 text-sm truncate leading-snug">{r.titel}</h4>
						<p class="text-xs text-slate-500 truncate mt-0.5">
							{#if r.isbn}<span class="font-mono text-slate-400">{r.isbn}</span>{/if}
							{#if r.verlag}<span class="mx-1.5 text-slate-300">·</span>{r.verlag}{/if}
							{#if r.signatur}<span class="mx-1.5 text-slate-300">·</span>{r.signatur}{/if}
						</p>
					</div>

					<!-- Bestandsstatus: bei 0 eigenen Exemplaren die klare Ansage statt "0/0".
					     „Fehlt komplett" (nicht „Vergriffen"): 0 im eigenen Bestand ist keine
					     Aussage über die Lieferbarkeit beim Verlag. -->
					<div class="text-right shrink-0 leading-tight">
						{#if r.gesamt_bestand === 0}
							<span
								class="inline-block text-[11px] font-bold text-rose-700 bg-rose-50 border border-rose-200 rounded-full px-2 py-0.5"
								>Fehlt komplett</span
							>
						{:else}
							<div class="text-sm font-bold tabular-nums text-amber-600" title="verfügbar / im Bestand">
								{r.verfuegbarer_bestand}<span class="text-slate-300 font-medium">/</span>{r.gesamt_bestand}
							</div>
						{/if}
					</div>

					<button
						onclick={() => onAddToCart(r)}
						aria-label="{r.titel} zur Bestellung hinzufügen"
						class="shrink-0 w-9 h-9 rounded-full border border-slate-200 text-slate-400 flex items-center justify-center hover:border-blue-500 hover:text-white hover:bg-blue-600 active:scale-90 transition-all cursor-pointer"
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14M5 12h14" />
						</svg>
					</button>
				</div>
			{/each}

			{#if gefiltert.length > maxVisible}
				<button
					onclick={() => (maxVisible += 60)}
					class="w-full py-2.5 text-xs font-bold text-slate-500 hover:text-slate-800 hover:bg-slate-50 rounded-xl transition-colors cursor-pointer"
				>
					Weitere {gefiltert.length - maxVisible} anzeigen
				</button>
			{/if}
		</div>
	{/if}
</section>

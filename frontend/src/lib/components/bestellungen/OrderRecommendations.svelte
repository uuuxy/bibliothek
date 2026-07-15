<script>
	let { recommendations, onAddToCart } = $props();

	// Nur die ersten Einträge ins DOM (Muster wie BookTable/Inventur-Startseite).
	// Der Bestellbedarf umfasst schnell den halben Katalog — jeder Titel unter seinem
	// Meldebestand landet hier. Alle zu rendern hiess: tausende DOM-Knoten und ebenso
	// viele Cover-Requests für eine Liste, die in ein 240px-Fenster schaut.
	let maxVisible = $state(50);

	let sichtbare = $derived(recommendations.slice(0, maxVisible));

	// Nach einem Datenwechsel (z. B. Wareneingang) wieder von vorn beginnen.
	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		recommendations;
		maxVisible = 50;
	});

	/** Ab hier wird es eng: unter 3 verfügbaren Exemplaren reicht ein Zugang nicht mehr. */
	const KRITISCH_AB = 3;

	/**
	 * Farbstufe einer Zeile. Der Meldebestand ist die eigentliche Schwelle (je Titel
	 * konfigurierbar, Standard 5); KRITISCH_AB hebt daraus die Fälle hervor, die keinen
	 * Aufschub dulden.
	 * @param {any} r
	 */
	function stufe(r) {
		return r.verfuegbarer_bestand < KRITISCH_AB ? 'kritisch' : 'knapp';
	}

	let kritischeAnzahl = $derived(
		recommendations.filter((/** @type {any} */ r) => stufe(r) === 'kritisch').length
	);
</script>

<div class="space-y-4">
	<div class="border-b border-gray-200 pb-3 flex items-center justify-between">
		<div>
			<h2 class="text-base font-bold text-slate-800">
				Bestellbedarf Lernmittel
				{#if recommendations.length}
					<span class="text-slate-400 font-semibold">({recommendations.length})</span>
				{/if}
			</h2>
			{#if kritischeAnzahl}
				<p class="text-[11px] font-semibold text-red-600 mt-0.5">
					{kritischeAnzahl}× dringend (unter {KRITISCH_AB} verfügbar)
				</p>
			{/if}
		</div>
		<a
			href="/api/bestellungen/pdf"
			download
			class="flex items-center gap-1.5 text-xs font-bold text-slate-500 hover:text-slate-800 transition-colors"
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-3.5 w-3.5 shrink-0"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
				/>
			</svg>
			PDF-Bestellliste
		</a>
	</div>
	{#if !recommendations.length}
		<p class="text-xs text-slate-400 text-center py-4">Bestände ausreichend.</p>
	{:else}
		<div class="max-h-60 overflow-y-auto space-y-2">
			{#each sichtbare as r, _i (_i)}
				{@const ist_kritisch = stufe(r) === 'kritisch'}
				<div
					class="p-2.5 border rounded-lg flex items-center justify-between gap-3 text-[11px] {ist_kritisch
						? 'bg-red-50 border-red-200'
						: 'bg-amber-50 border-amber-200'}"
				>
					<div class="flex items-center gap-2.5 min-w-0">
						{#if r.cover_url}
							<!-- lazy: die Liste ist ein schmales Scrollfenster; ohne dies fordert
							     der Browser die Cover aller gerenderten Zeilen auf einmal an. -->
							<img
								src="/api/images/cover?isbn={r.isbn || ''}&url={encodeURIComponent(r.cover_url)}"
								class="w-9 aspect-3/4 object-cover rounded-sm shrink-0 bg-white"
								loading="lazy"
								decoding="async"
								alt=""
							/>
						{:else}
							<div
								class="w-9 aspect-3/4 rounded bg-slate-200 flex items-center justify-center text-slate-400 shrink-0 text-[9px]"
							>
								📖
							</div>
						{/if}
						<div class="min-w-0">
							<h4 class="font-bold text-slate-800 truncate leading-tight">{r.titel}</h4>
							<!-- Beschaffungsdaten direkt sichtbar: ISBN und Verlag braucht man zum
							     Bestellen, die Signatur zum Wiederfinden im Regal. -->
							<p class="text-slate-500 truncate mt-0.5">
								{#if r.isbn}<span class="font-mono">{r.isbn}</span>{/if}
								{#if r.verlag}<span class="mx-1">·</span>{r.verlag}{/if}
								{#if r.signatur}<span class="mx-1">·</span>{r.signatur}{/if}
								{#if r.erscheinungsjahr}<span class="mx-1">·</span>{r.erscheinungsjahr}{/if}
							</p>
						</div>
					</div>
					<div class="flex items-center gap-2 shrink-0">
						<!-- Verfügbar UND gesamt: 0 von 30 heisst "Klassensatz unterwegs",
						     0 von 1 heisst "fehlt wirklich". -->
						<div class="text-right leading-tight">
							<div class="font-bold {ist_kritisch ? 'text-red-700' : 'text-amber-700'}">
								{r.verfuegbarer_bestand} von {r.gesamt_bestand}
							</div>
							<div class="text-[9px] text-slate-500">Melde: {r.meldebestand}</div>
						</div>
						<button
							onclick={() => onAddToCart(r)}
							class="px-2 py-1 bg-blue-50 hover:bg-blue-100 text-blue-700 font-bold rounded-md text-[9px] cursor-pointer"
							>+ Add</button
						>
					</div>
				</div>
			{/each}
			{#if recommendations.length > maxVisible}
				<button
					onclick={() => (maxVisible += 50)}
					class="w-full py-2 text-[11px] font-bold text-slate-500 hover:text-slate-800 hover:bg-slate-50 rounded-lg transition-colors cursor-pointer"
				>
					Mehr laden ({recommendations.length - maxVisible} weitere)
				</button>
			{/if}
		</div>
	{/if}
</div>

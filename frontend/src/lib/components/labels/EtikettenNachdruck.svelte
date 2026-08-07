<!-- @component EtikettenNachdruck — findet Exemplare, deren Barcode-Etikett nie gedruckt
     wurde, und übergibt sie an den Etikettendruck.

     Der Anlass: Eine Lieferung kann im System freigegeben sein, ohne dass die Etiketten
     je aus dem Drucker kamen — der Hinweis nach dem Wareneingang wird weggeklickt, und
     danach führte kein Weg mehr zu genau diesen Exemplaren zurück. Man hätte jeden Titel
     einzeln suchen müssen, ohne zu wissen, welche es überhaupt sind.

     Kein zweiter Druckweg: Die Auswahl geht in dieselbe printQueue, die auch der
     Wareneingang benutzt, und wird vom Etikettendruck nebenan gesetzt. -->
<script>
	import { onMount } from 'svelte';
	import { apiGet, apiPost } from '../../apiFetch.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import { Printer, Search, Eraser } from '@lucide/svelte';

	/** @type {{ onUebergeben?: () => void }} */
	let { onUebergeben } = $props();

	/** @type {{ barcode_id: string, titel: string, autor: string, erworben_am: string, etikett_gedruckt: boolean }[]} */
	let offen = $state.raw([]);
	let laedt = $state(true);
	let suche = $state('');

	/**
	 * Welche Exemplare die Liste zeigt: 'offen' (Vorgabe), 'erledigt' oder 'alle'.
	 *
	 * Erledigte sichtbar zu machen ist der WEG ZURÜCK. Bisher setzten alle drei Wege das
	 * Kennzeichen nur in eine Richtung — Stapeldruck, Einzeldruck, Altbestand-Stichtag.
	 * Blieb der Bogen im Drucker stecken oder war der Stichtag zu weit gefasst, galten die
	 * Exemplare als erledigt, ohne dass ein Etikett existierte, und verschwanden dauerhaft
	 * aus dieser Liste. Wer sie nicht sehen kann, kann sie auch nicht zurückholen.
	 */
	let status = $state(/** @type {'offen' | 'erledigt' | 'alle'} */ ('offen'));
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let sucheTimer;

	/** Barcodes der angehakten Zeilen. */
	let gewaehlt = $state(/** @type {string[]} */ ([]));

	let alleGewaehlt = $derived(offen.length > 0 && gewaehlt.length === offen.length);

	async function laden() {
		laedt = true;
		try {
			const q = suche.trim();
			// Von Hand zusammengesetzt statt per URLSearchParams: Das ist hier ein
			// Wegwerf-String, und die Svelte-Lint-Regel verlangt für die eingebaute Klasse
			// sonst die reaktive Variante — Aufwand ohne Gegenwert.
			const teile = [];
			if (q) teile.push(`q=${encodeURIComponent(q)}`);
			if (status !== 'offen') teile.push(`status=${status}`);
			const query = teile.length > 0 ? `?${teile.join('&')}` : '';
			offen = (await apiGet(`/api/exemplare/etiketten-offen${query}`)) || [];
			// Auswahl auf das beschränken, was noch in der Liste steht — sonst übergäbe ein
			// Klick auf "Drucken" Exemplare, die der Benutzer gar nicht mehr sieht.
			const sichtbar = new Set(offen.map((e) => e.barcode_id));
			gewaehlt = gewaehlt.filter((b) => sichtbar.has(b));
		} catch (err) {
			console.error('Offene Etiketten konnten nicht geladen werden', err);
			toastStore.addToast('Liste konnte nicht geladen werden.', 'error');
		} finally {
			laedt = false;
		}
	}

	onMount(() => {
		// Ein Verweis aus der Bestellhistorie bringt den Titel mit. Sofort zurücksetzen,
		// sonst klebt der Filter am nächsten Aufruf der Liste, und der Benutzer sucht den
		// Grund für eine halbleere Ansicht.
		if (uiStore.requestedEtikettenFilter) {
			suche = uiStore.requestedEtikettenFilter;
			uiStore.requestedEtikettenFilter = null;
		}
		laden();
	});

	function sucheAngestossen() {
		clearTimeout(sucheTimer);
		sucheTimer = setTimeout(laden, 300);
	}

	/** @param {string} barcode */
	function umschalten(barcode) {
		gewaehlt = gewaehlt.includes(barcode)
			? gewaehlt.filter((b) => b !== barcode)
			: [...gewaehlt, barcode];
	}

	function alleUmschalten() {
		gewaehlt = alleGewaehlt ? [] : offen.map((e) => e.barcode_id);
	}

	function uebergeben() {
		const auswahl = offen.filter((e) => gewaehlt.includes(e.barcode_id));
		if (auswahl.length === 0) return;
		printQueue.copies = auswahl.map((e) => ({
			barcode_id: e.barcode_id,
			titel: e.titel,
			autor: e.autor
		}));
		toastStore.addToast(
			`${auswahl.length} ${auswahl.length === 1 ? 'Etikett' : 'Etiketten'} im Druck übernommen.`,
			'success'
		);
		onUebergeben?.();
	}

	let arbeitet = $state(false);

	/** Die angehakten Zeilen, getrennt nach dem, was mit ihnen möglich ist. */
	let gewaehltOffen = $derived(
		offen.filter((e) => gewaehlt.includes(e.barcode_id) && !e.etikett_gedruckt)
	);
	let gewaehltErledigt = $derived(
		offen.filter((e) => gewaehlt.includes(e.barcode_id) && e.etikett_gedruckt)
	);

	/**
	 * Von Hand als gedruckt vermerken — OHNE zu drucken.
	 *
	 * Für den Fall, dass die Etiketten anderswo entstanden sind: aus dem Altsystem, von
	 * Hand geschrieben, oder weil der Lieferant beklebt geliefert hat und die Einstellung
	 * beim Bestellen noch nicht gesetzt war.
	 * @param {string[]} barcodes
	 */
	async function alsGedrucktMarkieren(barcodes) {
		if (barcodes.length === 0 || arbeitet) return;
		arbeitet = true;
		try {
			const daten = await apiPost('/api/exemplare/etiketten-gedruckt', { barcode_ids: barcodes });
			toastStore.addToast(`${daten?.markiert ?? 0} als erledigt vermerkt.`, 'success');
			await laden();
		} catch {
			toastStore.addToast('Vermerken nicht möglich.', 'error');
		} finally {
			arbeitet = false;
		}
	}

	/**
	 * Den Vermerk zurücknehmen — der eigentliche Notfallweg.
	 *
	 * Der Druck wird gegengebucht, sobald das PDF erzeugt ist; ob das Etikett wirklich aus
	 * dem Gerät kam, weiss das Programm nicht. Nach einem Papierstau stehen die Exemplare
	 * also als erledigt da, ohne Etikett am Buch. Dasselbe nach einem zu weit gefassten
	 * Stichtag beim Altbestand-Aufräumen.
	 * @param {string[]} barcodes
	 */
	async function wiederOeffnen(barcodes) {
		if (barcodes.length === 0 || arbeitet) return;
		arbeitet = true;
		try {
			const daten = await apiPost('/api/exemplare/etiketten-zuruecksetzen', {
				barcode_ids: barcodes
			});
			toastStore.addToast(`${daten?.zurueckgesetzt ?? 0} wieder als offen vermerkt.`, 'success');
			await laden();
		} catch {
			toastStore.addToast('Zurücksetzen nicht möglich.', 'error');
		} finally {
			arbeitet = false;
		}
	}

	// --- Altbestand ---
	//
	// etikett_gedruckt wurde bis vor Kurzem NIRGENDS gesetzt. Fuer den gesamten Altbestand
	// steht deshalb "kein Etikett" — nicht weil keins da waere, sondern weil es nie jemand
	// vermerkt hat. Ohne Aufraeummoeglichkeit zeigte diese Liste dauerhaft den ganzen
	// Bestand, und der Hinweis im Bestellwesen nennte eine Zahl ohne Bedeutung.
	//
	// Der Stichtag gehoert dem Betreiber: Er weiss, ab wann sein Regal beklebt ist. Deshalb
	// hier und nicht in einer Migration, die beim Update stillschweigend zugeschlagen und
	// dabei genau die Exemplare mitversteckt haette, wegen denen die Liste entstand.
	let stichtag = $state('');
	let betroffen = $state(0);
	let raeumtAuf = $state(false);

	async function zaehleAltbestand() {
		if (!stichtag) {
			betroffen = 0;
			return;
		}
		try {
			const daten = await apiGet(
				`/api/exemplare/etiketten-offen/anzahl?bis=${encodeURIComponent(stichtag)}`
			);
			betroffen = daten?.anzahl ?? 0;
		} catch {
			betroffen = 0;
		}
	}

	async function raeumeAltbestandAuf() {
		if (!stichtag || betroffen === 0) return;
		raeumtAuf = true;
		try {
			const daten = await apiPost('/api/exemplare/etiketten-altbestand', { bis: stichtag });
			toastStore.addToast(`${daten?.markiert ?? 0} Exemplare als erledigt vermerkt.`, 'success');
			stichtag = '';
			betroffen = 0;
			await laden();
		} catch {
			toastStore.addToast('Der Altbestand konnte nicht vermerkt werden.', 'error');
		} finally {
			raeumtAuf = false;
		}
	}

	/** @param {string} iso */
	function datum(iso) {
		return iso ? new Date(iso).toLocaleDateString('de-DE') : '—';
	}
</script>

<div class="w-full space-y-6 no-print animate-fade-in">
	<div class="flex flex-wrap items-center gap-4 border-b border-slate-200 pb-5">
		<div class="relative flex-1 min-w-64 max-w-md">
			<Search
				class="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
				aria-hidden="true"
			/>
			<input
				type="text"
				aria-label="Exemplare filtern"
				placeholder="Nach Titel oder Barcode filtern..."
				bind:value={suche}
				oninput={sucheAngestossen}
				class="w-full h-9 pl-10 pr-4 bg-white border border-slate-200 rounded-xl text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
			/>
		</div>

		<!-- Was die Liste zeigt. Vorgabe bleibt „Offen" — die Ansicht heisst „Fehlende
		     Etiketten" und soll ohne Zutun genau das zeigen. Die beiden anderen Stufen sind
		     Werkzeug für den Notfall, nicht für den Alltag. -->
		<div class="flex rounded-lg border border-slate-200 bg-white p-0.5 text-xs font-semibold">
			{#each [['offen', 'Offen'], ['erledigt', 'Erledigt'], ['alle', 'Alle']] as [wert, text] (wert)}
				<button
					type="button"
					onclick={() => {
						status = /** @type {'offen'|'erledigt'|'alle'} */ (wert);
						gewaehlt = [];
						laden();
					}}
					class="cursor-pointer rounded-md px-3 py-1.5 transition-colors {status === wert
						? 'bg-slate-800 text-white'
						: 'text-slate-600 hover:bg-slate-100'}"
				>
					{text}
				</button>
			{/each}
		</div>

		<Button size="lg" onclick={uebergeben} disabled={gewaehltOffen.length === 0} class="px-5">
			<Printer class="h-4 w-4" aria-hidden="true" />
			{gewaehltOffen.length === 0
				? 'Nichts ausgewählt'
				: `${gewaehltOffen.length} an den Druck übergeben`}
		</Button>
	</div>

	<!-- Notfallwege. Bewusst zurückhaltend und nur sichtbar, wenn etwas ausgewählt ist:
	     Der tägliche Weg ist der Druck darüber. -->
	{#if gewaehltOffen.length > 0 || gewaehltErledigt.length > 0}
		<div
			class="flex flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-slate-50/60 px-4 py-3"
		>
			<span class="text-xs font-semibold text-slate-500">Ohne zu drucken:</span>
			{#if gewaehltOffen.length > 0}
				<Button
					variant="secondary"
					size="sm"
					disabled={arbeitet}
					onclick={() => alsGedrucktMarkieren(gewaehltOffen.map((e) => e.barcode_id))}
					data-tip="Etikett ist schon am Buch — z. B. vom Lieferanten oder aus dem Altsystem"
				>
					{gewaehltOffen.length} als erledigt vermerken
				</Button>
			{/if}
			{#if gewaehltErledigt.length > 0}
				<Button
					variant="secondary"
					size="sm"
					disabled={arbeitet}
					onclick={() => wiederOeffnen(gewaehltErledigt.map((e) => e.barcode_id))}
					data-tip="Etikett fehlt doch — z. B. nach Papierstau oder zu weitem Stichtag"
				>
					{gewaehltErledigt.length} wieder als offen vermerken
				</Button>
			{/if}
		</div>
	{/if}

	{#if laedt}
		<p class="py-16 text-center text-slate-400 animate-pulse">Lade Exemplare…</p>
	{:else if offen.length === 0}
		<div class="py-16 text-center text-slate-400">
			{#if suche.trim() && status === 'erledigt'}
				Kein erledigtes Exemplar passt zu „{suche.trim()}".
			{:else if suche.trim() && status === 'alle'}
				Kein Exemplar passt zu „{suche.trim()}".
			{:else if suche.trim()}
				Kein Exemplar ohne Etikett passt zu „{suche.trim()}".
			{:else if status === 'erledigt'}
				Noch kein Exemplar als erledigt vermerkt.
			{:else if status === 'alle'}
				Keine Exemplare im Bestand.
			{:else}
				Für alle Exemplare wurden Etiketten gedruckt.
			{/if}
		</div>
	{:else}
		<div class="overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-xs">
			<table class="w-full border-collapse text-sm">
				<thead>
					<tr class="border-b border-slate-200 bg-slate-50/60 text-xs font-semibold text-slate-400">
						<th class="w-10 px-3 py-2">
							<input
								type="checkbox"
								aria-label="Alle auswählen"
								checked={alleGewaehlt}
								onchange={alleUmschalten}
								class="accent-blue-600"
							/>
						</th>
						<th class="px-3 py-2 text-left font-semibold">Titel</th>
						<th class="px-3 py-2 text-left font-semibold">Barcode</th>
						<th class="px-3 py-2 text-right font-semibold">Zugang</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each offen as e (e.barcode_id)}
						{@const markiert = gewaehlt.includes(e.barcode_id)}
						<tr class="transition-colors {markiert ? 'bg-blue-50/50' : 'hover:bg-slate-50/60'}">
							<td class="px-3 py-2">
								<input
									type="checkbox"
									aria-label="{e.titel} ({e.barcode_id}) auswählen"
									checked={markiert}
									onchange={() => umschalten(e.barcode_id)}
									class="accent-blue-600"
								/>
							</td>
							<td class="max-w-0 px-3 py-2">
								<span class="block truncate font-semibold text-slate-800">
									{e.titel}
									<!-- Nur in den gemischten Ansichten: In „Offen" wäre der Vermerk an
									     jeder Zeile derselbe und damit ohne Aussage. -->
									{#if e.etikett_gedruckt && status !== 'erledigt'}
										<span class="ml-1.5 text-xs font-medium text-slate-400">· erledigt</span>
									{/if}
								</span>
								{#if e.autor}
									<span class="block truncate text-xs text-slate-400">{e.autor}</span>
								{/if}
							</td>
							<td class="px-3 py-2 font-mono text-xs whitespace-nowrap text-slate-600"
								>{e.barcode_id}</td
							>
							<td class="px-3 py-2 text-right whitespace-nowrap text-slate-500 tabular-nums"
								>{datum(e.erworben_am)}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="text-xs text-slate-400">
			Neueste zuerst. Nach dem Druck verschwinden die Exemplare aus dieser Liste.
		</p>
	{/if}

	<!-- Altbestand aufräumen. Bewusst unten und optisch zurückhaltend: Es ist eine einmalige
	     Aufräumaktion, keine tägliche Arbeit — und sie lässt sich nicht rückgängig machen,
	     deshalb nennt sie die betroffene Zahl, bevor sie etwas tut. -->
	<details class="rounded-xl border border-slate-200 bg-slate-50/60 px-4 py-3">
		<summary class="cursor-pointer text-sm font-semibold text-slate-600 select-none">
			Altbestand aufräumen
		</summary>
		<div class="mt-3 space-y-3">
			<p class="text-xs leading-relaxed text-slate-500">
				Für Exemplare aus der Zeit vor dieser Funktion wurde nie vermerkt, ob ein Etikett gedruckt
				wurde — sie stehen deshalb alle in dieser Liste, auch die längst beklebten. Wähle den Tag,
				bis zu dem dein Bestand beklebt ist; alles bis dahin gilt danach als erledigt. <span
					class="font-semibold text-slate-600">Das lässt sich nicht rückgängig machen.</span
				>
			</p>
			<div class="flex flex-wrap items-center gap-3">
				<label class="text-xs font-medium text-slate-600" for="stichtag">Beklebt bis</label>
				<input
					id="stichtag"
					type="date"
					bind:value={stichtag}
					onchange={zaehleAltbestand}
					class="h-9 rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-800 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none"
				/>
				<Button
					variant="secondary"
					onclick={raeumeAltbestandAuf}
					disabled={!stichtag || betroffen === 0 || raeumtAuf}
				>
					<Eraser class="h-4 w-4" aria-hidden="true" />
					{#if !stichtag}
						Datum wählen
					{:else if betroffen === 0}
						Nichts zu vermerken
					{:else}
						{betroffen} Exemplare als erledigt vermerken
					{/if}
				</Button>
			</div>
		</div>
	</details>
</div>

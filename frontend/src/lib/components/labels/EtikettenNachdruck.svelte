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
	import { apiGet } from '../../apiFetch.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import { Printer, Search } from '@lucide/svelte';

	/** @type {{ onUebergeben?: () => void }} */
	let { onUebergeben } = $props();

	/** @type {{ barcode_id: string, titel: string, autor: string, erworben_am: string }[]} */
	let offen = $state.raw([]);
	let laedt = $state(true);
	let suche = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let sucheTimer;

	/** Barcodes der angehakten Zeilen. */
	let gewaehlt = $state(/** @type {string[]} */ ([]));

	let alleGewaehlt = $derived(offen.length > 0 && gewaehlt.length === offen.length);

	async function laden() {
		laedt = true;
		try {
			const q = suche.trim();
			offen =
				(await apiGet(`/api/exemplare/etiketten-offen${q ? `?q=${encodeURIComponent(q)}` : ''}`)) ||
				[];
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

	onMount(laden);

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

		<Button size="lg" onclick={uebergeben} disabled={gewaehlt.length === 0} class="px-5">
			<Printer class="h-4 w-4" aria-hidden="true" />
			{gewaehlt.length === 0 ? 'Nichts ausgewählt' : `${gewaehlt.length} an den Druck übergeben`}
		</Button>
	</div>

	{#if laedt}
		<p class="py-16 text-center text-slate-400 animate-pulse">Lade Exemplare…</p>
	{:else if offen.length === 0}
		<div class="py-16 text-center text-slate-400">
			{#if suche.trim()}
				Kein Exemplar ohne Etikett passt zu „{suche.trim()}".
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
								<span class="block truncate font-semibold text-slate-800">{e.titel}</span>
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
</div>

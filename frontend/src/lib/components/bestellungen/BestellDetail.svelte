<!-- @component Eine Bestellung als Beleg — Kopf, bestellte Titel, gelieferte Exemplare.

     Ersetzt die aufklappende Zeile der Bestellhistorie. Die zeigte dieselben Angaben wie
     die Tabellenzeile darüber, nur untereinander; Peter: „geht das Feld nach unten aber
     nicht mit merklichem Mehrwert". Zwei Dinge fehlten und passen auch nicht in eine
     Tabellenzeile: das Cover und die Exemplarnummern.

     Eigener Endpunkt statt breiterer Liste: /api/bestellhistorie ist auf 200 Bestellungen
     gedeckelt, WEIL sie ungedeckelt 2,45 MB gebraucht hat. Cover und Exemplarlisten an
     alle 200 zu hängen, holte genau das zurück — für Daten, von denen jeweils eine
     einzige Bestellung angesehen wird. -->
<script>
	import { onMount } from 'svelte';
	import { apiGet } from '../../apiFetch.js';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { appState } from '../../../inventur/lib/store.svelte.js';
	import BestellStatusBlock from './BestellStatusBlock.svelte';
	import BestellDetailPositionen from './BestellDetailPositionen.svelte';
	import BestellDetailExemplare from './BestellDetailExemplare.svelte';
	import Button from '../ui/Button.svelte';
	import { ArrowLeft } from '@lucide/svelte';

	/** @type {{ bestellungId: string, onBack: () => void }} */
	let { bestellungId, onBack } = $props();

	/** @type {any} */
	let bestellung = $state(null);
	let laedt = $state(true);
	/** @type {string} */
	let fehler = $state('');

	async function laden() {
		laedt = true;
		fehler = '';
		try {
			bestellung = await apiGet(`/api/bestellhistorie/${bestellungId}`);
		} catch (e) {
			// Ein veralteter Verweis liefert 404. Die Meldung nennt den Rückweg, statt eine
			// leere Seite stehen zu lassen, die aussieht wie eine leere Bestellung.
			fehler = /** @type {any} */ (e)?.message || 'Bestellung konnte nicht geladen werden.';
		} finally {
			laedt = false;
		}
	}

	// onMount statt $effect auf bestellungId: Die Historie zeigt Liste ODER Detail, der
	// Weg von einer Bestellung zur nächsten führt zwingend über den Zurück-Knopf. Damit
	// wird diese Komponente je Bestellung neu aufgebaut, und ein Effekt, der auf einen
	// Wechsel horcht, der nicht stattfinden kann, wäre nur eine Behauptung über den Ablauf.
	onMount(laden);

	/** @param {number} n */
	function euro(n) {
		return n.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €';
	}

	/** @param {string} iso */
	function langdatum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', {
			day: '2-digit',
			month: 'long',
			year: 'numeric'
		});
	}

	/** @param {any} pos */
	function zumNachdruck(pos) {
		uiStore.requestedEtikettenFilter = pos.titel_name;
		uiStore.requestedDruckCenterTab = 'nachdruck';
		uiStore.activeTab = 'druck-center';
	}

	/** @param {any} pos */
	function zumTitel(pos) {
		appState.activeBookId = pos.titel_id;
		uiStore.activeTab = 'book_detail';
	}
</script>

<div class="space-y-6">
	<Button variant="secondary" onclick={onBack}>
		<ArrowLeft class="h-4 w-4" aria-hidden="true" />
		Zurück zur Bestellhistorie
	</Button>

	{#if laedt}
		<div class="animate-pulse py-16 text-center text-base text-slate-400">Lade Bestellung…</div>
	{:else if fehler}
		<div class="rounded-xl border border-red-200 bg-red-50 py-8 text-center text-red-650">
			{fehler}
		</div>
	{:else if bestellung}
		<div class="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
			<div>
				<h2 class="text-base font-bold text-slate-800">{bestellung.lieferant_name}</h2>
				<p class="mt-0.5 text-sm text-slate-500">
					{langdatum(bestellung.bestelldatum)}
					{#if bestellung.kundennummer}· Kd.-Nr. {bestellung.kundennummer}{/if}
				</p>
				<p class="text-sm text-slate-400">{bestellung.lieferant_email}</p>
			</div>
			<div class="text-right">
				{#if orderStore.preiseErfassen}
					<div class="text-xs font-semibold text-slate-400">Bestellwert</div>
					<div class="text-2xl font-black text-slate-800">{euro(bestellung.gesamtbetrag)}</div>
				{/if}
				<div class="text-sm text-slate-500 tabular-nums">
					{bestellung.anzahl_exemplare} Exemplare bestellt
				</div>
			</div>
		</div>

		<!-- Unverändert dieselbe Komponente wie in der Historie: Der Bestätigungs-Ablauf
		     gehört an EINE Stelle. -->
		{#if bestellung.mit_bestaetigung}
			<BestellStatusBlock b={bestellung} onAktualisieren={laden} />
		{/if}

		<section>
			<h3 class="mb-2 text-base font-bold text-slate-700">Bestellte Titel</h3>
			<BestellDetailPositionen
				positionen={bestellung.positionen}
				{euro}
				onNachdruck={zumNachdruck}
				onTitel={zumTitel}
			/>
		</section>

		<section>
			<h3 class="mb-2 text-base font-bold text-slate-700">
				Exemplare aus dieser Bestellung
				{#if bestellung.exemplare.length > 0}
					<span class="font-normal text-slate-400">({bestellung.exemplare.length})</span>
				{/if}
			</h3>
			<BestellDetailExemplare exemplare={bestellung.exemplare} />
		</section>
	{/if}
</div>

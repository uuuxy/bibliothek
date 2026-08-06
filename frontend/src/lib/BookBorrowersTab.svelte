<script>
	import Button from './components/ui/Button.svelte';
	import Select from './components/ui/Select.svelte';
	import BorrowersListe from './components/BorrowersListe.svelte';
	import { baueAusleiherDruckHtml } from './utils/ausleiherDruck.js';
	import { fmtDateDE as fmtDate } from './utils/dates.js';

	/** @type {{ borrowers: any[], book: any, onBack: () => void }} */
	let { borrowers, book, onBack } = $props();

	let filterKlasse = $state('Alle');
	let filterName = $state('');

	let availableKlassen = $derived([
		'Alle',
		...Array.from(new Set(borrowers.map((b) => b.klasse || 'Unbekannt'))).sort()
	]);

	let filteredBorrowers = $derived(
		borrowers.filter((b) => {
			const matchKlasse = filterKlasse === 'Alle' || (b.klasse || 'Unbekannt') === filterKlasse;
			const matchName =
				filterName === '' ||
				`${b.schueler_name} ${b.schueler_nachname}`
					.toLowerCase()
					.includes(filterName.toLowerCase());
			return matchKlasse && matchName;
		})
	);

	function printAusleiher() {
		const printWindow = window.open('', '_blank', 'width=800,height=600');
		if (!printWindow) {
			alert('Bitte erlaube Popups, um die Liste zu drucken.');
			return;
		}

		printWindow.document.open();
		printWindow.document.write(baueAusleiherDruckHtml(filteredBorrowers, book, filterKlasse));
		printWindow.document.close();

		// Drucken wird VON HIER angestoßen, nicht mehr von einem <script> im geschriebenen
		// Dokument. Der Grund ist keine Stilfrage: Ein per window.open('') erzeugtes
		// about:blank erbt die CSP des Openers, und die erlaubt nur script-src 'self'.
		// Das eingebettete Skript wurde also nie ausgeführt — das Fenster ging auf und
		// blieb stehen, drucken musste man von Hand (gemessen am 06.08.2026).
		//
		// Aus dem Opener heraus ist es kein Inline-Skript mehr, sondern ein gewöhnlicher
		// Aufruf auf dem gleichen Origin. document.write ist synchron und das Dokument
		// enthält keine nachzuladenden Ressourcen, deshalb steht der Inhalt hier bereits.
		printWindow.focus();
		printWindow.print();
	}
</script>

{#if borrowers.length === 0}
	<div class="py-16 flex flex-col items-center text-slate-400 gap-3">
		<svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="1.5"
				d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"
			/>
		</svg>
		<p class="font-semibold text-sm">Aktuell niemand hat dieses Buch ausgeliehen.</p>
	</div>
{:else}
	<!-- Filters -->
	<div class="flex gap-3 mb-4">
		<Select
			bind:value={filterKlasse}
			options={availableKlassen.map((/** @type {string} */ k) => ({ value: k, label: k }))}
			class="w-44"
			aria-label="Nach Klasse filtern"
		/>
		<div class="relative flex-1 max-w-xs">
			<svg
				aria-hidden="true"
				class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				/></svg
			>
			<input
				type="text"
				bind:value={filterName}
				aria-label="Nach Name filtern"
				placeholder="Name filtern..."
				class="w-full pl-9 pr-3 py-2 bg-white border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/30 placeholder:text-slate-400"
			/>
		</div>
		{#if filteredBorrowers.length !== borrowers.length}
			<span class="text-xs text-slate-400 self-center"
				>{filteredBorrowers.length} von {borrowers.length}</span
			>
		{/if}
		<div class="flex-1"></div>
		<Button variant="secondary" onclick={printAusleiher}>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
				/></svg
			>
			Mahnliste drucken
		</Button>
	</div>

	<BorrowersListe zeilen={filteredBorrowers} {onBack} {fmtDate} />
{/if}

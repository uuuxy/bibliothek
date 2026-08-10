<script>
	import Button from './components/ui/Button.svelte';
	import Select from './components/ui/Select.svelte';
	import BorrowersListe from './components/BorrowersListe.svelte';
	import { baueAusleiherDruckHtml } from './utils/ausleiherDruck.js';
	import { fmtDateDE as fmtDate } from './utils/dates.js';
	import { Printer, Users } from '@lucide/svelte';
	import Suchfeld from './components/ui/Suchfeld.svelte';

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
		<Users class="w-10 h-10" aria-hidden="true" />
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
		<Suchfeld
			bind:wert={filterName}
			platzhalter="Name eingeben …"
			etikett="Nach Name filtern"
			klasse="flex-1 max-w-xs"
		/>
		{#if filteredBorrowers.length !== borrowers.length}
			<span class="text-xs text-slate-400 self-center"
				>{filteredBorrowers.length} von {borrowers.length}</span
			>
		{/if}
		<div class="flex-1"></div>
		<Button variant="secondary" onclick={printAusleiher}>
			<Printer class="w-4 h-4" aria-hidden="true" />
			Mahnliste drucken
		</Button>
	</div>

	<BorrowersListe zeilen={filteredBorrowers} {onBack} {fmtDate} />
{/if}

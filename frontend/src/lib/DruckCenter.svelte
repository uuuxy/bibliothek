<script>
	import StudentIdDesigner from './StudentIdDesigner.svelte';
	import LabelPrinter from './LabelPrinter.svelte';
	import EtikettenNachdruck from './components/labels/EtikettenNachdruck.svelte';
	import { uiStore } from './stores/uiStore.svelte.js';

	let activeTab = $state('labels');

	// Ein Verweis aus einer anderen Ansicht (z. B. der Hinweis auf offene Etiketten im
	// Bestellwesen) bestimmt den Reiter. Sofort zurücksetzen, sonst klebt die Wahl am
	// nächsten Aufruf des Druck-Centers, ohne dass jemand versteht, warum.
	$effect(() => {
		if (!uiStore.requestedDruckCenterTab) return;
		activeTab = uiStore.requestedDruckCenterTab;
		uiStore.requestedDruckCenterTab = null;
	});
</script>

<div class="w-full flex flex-col h-full bg-slate-50">
	<div class="px-8 pt-6 pb-4 border-b border-slate-200 bg-white shadow-sm shrink-0">
		<div class="max-w-6xl mx-auto flex gap-6">
			<button
				onclick={() => (activeTab = 'labels')}
				class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'labels'
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}"
			>
				Buch-Etiketten
			</button>
			<button
				onclick={() => (activeTab = 'nachdruck')}
				class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'nachdruck'
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}"
			>
				Fehlende Etiketten
			</button>
			<button
				onclick={() => (activeTab = 'ids')}
				class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'ids'
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}"
			>
				Schülerausweise
			</button>
		</div>
	</div>

	<div class="flex-1 overflow-y-auto">
		<!-- Alle drei Reiter tragen dasselbe px-8 py-6 wie die Tab-Leiste darüber — vorher
		     bekam nur "Fehlende Etiketten" dieses Padding, die anderen beiden Reiter waren
		     komplett randlos und nur zufällig ähnlich weit eingerückt wie die Tab-Leiste
		     (deren max-w-6xl mx-auto bei anderer Fensterbreite auseinandergelaufen wäre). -->
		{#if activeTab === 'labels'}
			<div class="animate-fade-in h-full px-8 py-6">
				<LabelPrinter />
			</div>
		{:else if activeTab === 'nachdruck'}
			<!-- Nach dem Übergeben direkt zum Etikettendruck: Die Auswahl liegt dort schon
			     bereit, und ein Hinweis "wechseln Sie jetzt nach nebenan" wäre eine Arbeit,
			     die das Programm selbst erledigen kann. -->
			<div class="animate-fade-in h-full px-8 py-6">
				<EtikettenNachdruck onUebergeben={() => (activeTab = 'labels')} />
			</div>
		{:else}
			<div class="animate-fade-in h-full px-8 py-6">
				<StudentIdDesigner />
			</div>
		{/if}
	</div>
</div>

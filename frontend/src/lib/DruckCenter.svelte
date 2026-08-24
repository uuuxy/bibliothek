<script>
	import StudentIdDesigner from './StudentIdDesigner.svelte';
	import KlassenDruckEinstieg from './components/students/KlassenDruckEinstieg.svelte';
	import LabelPrinter from './LabelPrinter.svelte';
	import EtikettenNachdruck from './components/labels/EtikettenNachdruck.svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';
	import Reiter from './components/ui/Reiter.svelte';

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

<PageShell>
	<!-- Reiter auf der Leinwand, nicht in einem eigenen weissen Balken — wie Mahnwesen,
	     Medienkatalog und Schuelerdatei. -->
	<!-- Ohne max-w-6xl mx-auto: Die Reiterzeile stand dadurch auf einer ANDEREN Kante als
	     der Inhalt darunter — gemessen bei 1600 px begannen die Reiter bei 352, die
	     Überschriften bei 320. Der Kommentar an den Reiter-Inhalten ahnte das bereits
	     („nur zufällig ähnlich weit eingerückt … wäre bei anderer Fensterbreite
	     auseinandergelaufen"). Die Inhaltsbreite ist mit a4133a2 ohnehin abgeschafft:
	     „Es gibt keine Inhaltsbreite mehr, nur noch volle." -->
	<Reiter
		etikett="Druck-Center"
		aktiv={activeTab}
		onwahl={(id) => (activeTab = id)}
		reiter={[
			{ id: 'labels', label: 'Buch-Etiketten' },
			// Das Badge steht hier UND an „Druck-Center" in der Seitenleiste: Der Zähler
			// führt erst zum Ziel, dann zum Reiter darin.
			{ id: 'nachdruck', label: 'Fehlende Etiketten', anzahl: uiStore.offeneEtiketten },
			{ id: 'ids', label: 'Schülerausweise' },
			// Eigener Reiter (Peters Entscheidung 24.08.2026): Als Block über dem Designer
			// wirkte der Einstieg „hingeklatscht" — er ist eine eigene Aufgabe, kein Teil
			// des Ausweis-Designs.
			{ id: 'klassen', label: 'Klassenweise drucken' }
		]}
	/>

	<div class="flex-1 overflow-y-auto">
		<!-- Alle drei Reiter tragen dieselbe Kante wie die Tab-Leiste darüber. Das px-8 ist
		     dabei entfallen: Die äußere Polsterung sitzt laut PageShell in App.svelte und gilt
		     für alle Routen — ein zusätzliches px-8 hier polsterte doppelt und schob den
		     Inhalt gegenüber den Reitern um 32 px nach innen. Vorher
		     bekam nur "Fehlende Etiketten" dieses Padding, die anderen beiden Reiter waren
		     komplett randlos und nur zufällig ähnlich weit eingerückt wie die Tab-Leiste
		     (deren max-w-6xl mx-auto bei anderer Fensterbreite auseinandergelaufen wäre). -->
		{#if activeTab === 'labels'}
			<div class="animate-fade-in h-full py-6">
				<LabelPrinter />
			</div>
		{:else if activeTab === 'nachdruck'}
			<!-- Nach dem Übergeben direkt zum Etikettendruck: Die Auswahl liegt dort schon
			     bereit, und ein Hinweis "wechseln Sie jetzt nach nebenan" wäre eine Arbeit,
			     die das Programm selbst erledigen kann. -->
			<div class="animate-fade-in h-full py-6">
				<EtikettenNachdruck onUebergeben={() => (activeTab = 'labels')} />
			</div>
		{:else if activeTab === 'klassen'}
			<div class="animate-fade-in h-full py-6">
				<KlassenDruckEinstieg />
			</div>
		{:else}
			<div class="animate-fade-in h-full py-6">
				<StudentIdDesigner />
			</div>
		{/if}
	</div>
</PageShell>

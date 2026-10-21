<script>
	import AusweisDesigner from './AusweisDesigner.svelte';
	import KlassenDruckEinstieg from './components/leser/KlassenDruckEinstieg.svelte';
	import Etikettendruck from './Etikettendruck.svelte';
	import EtikettenNachdruck from './components/etiketten/EtikettenNachdruck.svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menue.js';
	import { DRUCK_CENTER_REITER } from './druckCenterReiter.js';

	// Nur die Reiter, deren Routen die Rolle öffnen darf: Der Menüpunkt öffnet mit jedem
	// ihrer Rechte, und ein Reiter dahinter, der nur 403 erntet, wäre eine Tür ins Leere.
	const reiter = $derived(
		DRUCK_CENTER_REITER.filter((r) =>
			r.rechte.every((recht) => hatRecht(authStore.currentUser, recht))
		).map((r) =>
			// Das Badge steht hier UND an „Druck-Center" in der Seitenleiste: Der Zähler
			// führt erst zum Ziel, dann zum Reiter darin.
			r.id === 'nachdruck'
				? { id: r.id, label: r.label, anzahl: uiStore.offeneEtiketten }
				: { id: r.id, label: r.label }
		)
	);
	let gewaehlt = $state('labels');
	const aktiverReiter = $derived(
		reiter.some((r) => r.id === gewaehlt) ? gewaehlt : (reiter[0]?.id ?? '')
	);

	// Ein Verweis aus einer anderen Ansicht (z. B. der Hinweis auf offene Etiketten im
	// Bestellwesen) bestimmt den Reiter. Sofort zurücksetzen, sonst klebt die Wahl am
	// nächsten Aufruf des Druck-Centers, ohne dass jemand versteht, warum.
	$effect(() => {
		if (!uiStore.requestedDruckCenterTab) return;
		gewaehlt = uiStore.requestedDruckCenterTab;
		uiStore.requestedDruckCenterTab = null;
	});
</script>

<Seitenrahmen>
	<!-- Reiter auf der Leinwand, nicht in einem eigenen weissen Balken — wie Mahnwesen,
	     Medienkatalog und Schuelerdatei. -->
	<!-- Ohne max-w-6xl mx-auto: Die Reiterzeile stand dadurch auf einer ANDEREN Kante als
	     der Inhalt darunter — gemessen bei 1600 px begannen die Reiter bei 352, die
	     Überschriften bei 320. Der Kommentar an den Reiter-Inhalten ahnte das bereits
	     („nur zufällig ähnlich weit eingerückt … wäre bei anderer Fensterbreite
	     auseinandergelaufen"). Die Inhaltsbreite ist mit a4133a2 ohnehin abgeschafft:
	     „Es gibt keine Inhaltsbreite mehr, nur noch volle." -->
	<Reiter etikett="Druck-Center" aktiv={aktiverReiter} onwahl={(id) => (gewaehlt = id)} {reiter} />

	<div class="flex-1 overflow-y-auto">
		<!-- Alle drei Reiter tragen dieselbe Kante wie die Tab-Leiste darüber. Das px-8 ist
		     dabei entfallen: Die äußere Polsterung sitzt laut Seitenrahmen in App.svelte und gilt
		     für alle Routen — ein zusätzliches px-8 hier polsterte doppelt und schob den
		     Inhalt gegenüber den Reitern um 32 px nach innen. Vorher
		     bekam nur "Fehlende Etiketten" dieses Padding, die anderen beiden Reiter waren
		     komplett randlos und nur zufällig ähnlich weit eingerückt wie die Tab-Leiste
		     (deren max-w-6xl mx-auto bei anderer Fensterbreite auseinandergelaufen wäre). -->
		{#if aktiverReiter === 'labels'}
			<div class="animate-fade-in h-full py-6">
				<Etikettendruck />
			</div>
		{:else if aktiverReiter === 'nachdruck'}
			<!-- Nach dem Übergeben direkt zum Etikettendruck: Die Auswahl liegt dort schon
			     bereit, und ein Hinweis "wechseln Sie jetzt nach nebenan" wäre eine Arbeit,
			     die das Programm selbst erledigen kann. -->
			<div class="animate-fade-in h-full py-6">
				<EtikettenNachdruck onUebergeben={() => (gewaehlt = 'labels')} />
			</div>
		{:else if aktiverReiter === 'klassen'}
			<div class="animate-fade-in h-full py-6">
				<KlassenDruckEinstieg />
			</div>
		{:else if aktiverReiter === 'ids'}
			<div class="animate-fade-in h-full py-6">
				<AusweisDesigner />
			</div>
		{/if}
	</div>
</Seitenrahmen>

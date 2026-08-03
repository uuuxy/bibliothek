<script>
	import { onMount } from 'svelte';
	import { apiGet } from './apiFetch.js';
	import { orderStore } from './stores/orderStore.svelte.js';
	import { printQueue } from './stores/printQueue.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';

	import OrderCreationPanel from './components/bestellungen/OrderCreationPanel.svelte';
	import IncomingShipments from './components/bestellungen/IncomingShipments.svelte';
	import WareneingangView from './components/bestellungen/WareneingangView.svelte';
	import OrderRecommendations from './components/bestellungen/OrderRecommendations.svelte';
	import SupplierManager from './components/bestellungen/SupplierManager.svelte';
	import BestellHistorie from './components/bestellungen/BestellHistorie.svelte';
	import BestellBerichte from './components/bestellungen/BestellBerichte.svelte';
	import PrintSuggestion from './components/bestellungen/PrintSuggestion.svelte';
	import KlassensatzReservierungen from './components/bestellungen/KlassensatzReservierungen.svelte';

	let activeTab = $state('bestellungen');
	let showWareneingang = $state(false);
	let showGreenFade = $state(false);
	/** @type {any[] | null} Exemplare der letzten Einbuchung ohne gedrucktes Etikett */
	let printSuggestion = $state(null);

	/** Exemplare ohne Barcode-Etikett — speist den stehenden Hinweis in PrintSuggestion. */
	let offeneEtiketten = $state(0);

	async function ladeOffeneEtiketten() {
		try {
			const daten = await apiGet('/api/exemplare/etiketten-offen/anzahl');
			offeneEtiketten = daten?.anzahl ?? 0;
		} catch (err) {
			// Ein fehlender Hinweis darf das Bestellwesen nicht aufhalten.
			console.error('Offene Etiketten konnten nicht gezählt werden', err);
		}
	}

	onMount(() => {
		orderStore.init();
		ladeOffeneEtiketten();
	});

	/** @param {any[]} receivedItems */
	async function handleShipmentReceived(receivedItems) {
		showWareneingang = false;
		const needsPrinting = receivedItems.filter((item) => !item.etikett_gedruckt);
		printSuggestion = needsPrinting.length > 0 ? needsPrinting : null;
		showGreenFade = true;
		await orderStore.loadIncomingShipments();
		orderStore.loadRecommendations();
		ladeOffeneEtiketten();
		setTimeout(() => {
			showGreenFade = false;
		}, 1500);
	}

	function handlePrintSuggestion() {
		printQueue.copies = printSuggestion;
		printSuggestion = null;
	}

	/**
	 * Höhe der Bestellspalte, damit sie nie unter den Fensterrand wächst.
	 *
	 * Aus dem Betrieb gemeldet als „Bestellung abgeschnitten": Die Spalte klebt beim
	 * Scrollen (sticky). Füllt sich der Warenkorb, wird sie höher als das Fenster — bei
	 * 1366×700 gemessen 1427 px, also 979 px unter dem Rand. Der Absenden-Knopf war dann
	 * nur noch zu erreichen, indem man die 105 Titel der Bedarfsliste daneben
	 * durchscrollte, bis der klebende Rahmen endete.
	 *
	 * Warum gemessen statt einer festen CSS-Grenze: Ein calc(100vh - fester Abzug) kann
	 * nicht stimmen, weil der Anfang der Spalte wandert — mit eingeblendetem
	 * Backup-Banner beginnt sie rund 80 px tiefer, und beim Scrollen wandert sie nach
	 * oben, bis sie klebt. Ein Versuch mit 8rem Abzug liess immer noch 125 px überstehen.
	 * Deshalb rechnet die Spalte mit ihrem tatsächlichen Abstand zum oberen Rand.
	 */
	/** @type {HTMLElement|undefined} */
	let rail = $state();
	let railMaxHeight = $state('');

	$effect(() => {
		if (!rail) return;
		const messen = () => {
			if (!rail) return;
			const oben = rail.getBoundingClientRect().top;
			// 8 px Luft unten, damit die Spalte nicht am Rand klebt.
			railMaxHeight = `${Math.max(240, Math.round(window.innerHeight - oben - 8))}px`;
		};
		messen();
		// Beim Scrollen wandert der Anfang, bis die Spalte klebt — dann wird mehr Platz frei.
		window.addEventListener('scroll', messen, true);
		window.addEventListener('resize', messen);

		// Und wenn sich das Layout ÜBER der Spalte ändert. Ohne das rechnete der Effekt nur
		// einmal beim Einhängen — zu einem Zeitpunkt, an dem der Backup-Hinweis und der
		// Wareneingangs-Streifen noch nicht standen. Die Spalte begann danach rund 100 px
		// tiefer, behielt aber die zu grosszügige Höhe und ragte wieder unter den Rand.
		const beobachter = new ResizeObserver(messen);
		beobachter.observe(document.body);
		beobachter.observe(rail);

		return () => {
			window.removeEventListener('scroll', messen, true);
			window.removeEventListener('resize', messen);
			beobachter.disconnect();
		};
	});
</script>

<div class="w-full h-full text-slate-800 font-sans flex flex-col gap-6">
	<!-- Tab-Bar: reine Navigation, keine Aktionen -->
	<div class="flex items-end gap-6 border-b border-slate-200 shrink-0">
		{#snippet tab(id, label)}
			<button
				onclick={() => (activeTab = id)}
				class="pb-3 text-sm font-semibold border-b-2 transition-colors cursor-pointer {activeTab ===
				id
					? 'border-blue-600 text-blue-700'
					: 'border-transparent text-slate-500 hover:text-slate-800'}">{label}</button
			>
		{/snippet}
		{@render tab('bestellungen', 'Bestellungen')}
		{@render tab('lieferanten', 'Lieferanten verwalten')}
		{@render tab('historie', 'Bestellhistorie')}
		{@render tab('berichte', 'Berichte')}
		<button
			onclick={() => (activeTab = 'klassensaetze')}
			class="pb-3 text-sm font-semibold border-b-2 transition-colors cursor-pointer flex items-center gap-2 {activeTab ===
			'klassensaetze'
				? 'border-blue-600 text-blue-700'
				: 'border-transparent text-slate-500 hover:text-slate-800'}"
		>
			Klassensatz-Reservierungen
			{#if uiStore.pendingReservierungen > 0}
				<span
					class="min-w-5 h-5 flex items-center justify-center rounded-full bg-rose-500 text-white text-[10px] font-bold px-1"
					>{uiStore.pendingReservierungen}</span
				>
			{/if}
		</button>
	</div>

	{#if activeTab === 'bestellungen'}
		{#if showWareneingang}
			<WareneingangView
				incomingShipments={orderStore.incomingShipments}
				onBack={() => (showWareneingang = false)}
				onReceived={handleShipmentReceived}
			/>
		{:else}
			<div class="flex flex-col gap-5">
				<!-- Wareneingang als schlanker Statusstreifen über der Arbeitsfläche -->
				<IncomingShipments
					incomingShipments={orderStore.incomingShipments}
					{showGreenFade}
					onOpenWareneingang={() => (showWareneingang = true)}
				/>

				<div class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
					<!-- HERO: der Bestellbedarf ist die tägliche Arbeitsfläche -->
					<div class="lg:col-span-7 xl:col-span-8 min-w-0">
						<OrderRecommendations
							recommendations={orderStore.recommendations}
							onAddToCart={(book) => orderStore.addToCart(book)}
						/>
					</div>

					<!-- RAIL: die Bestellung wächst sichtbar mit, bleibt beim Scrollen stehen.
					     Aus dem Betrieb gemeldet als „Bestellung abgeschnitten". Gemessen bei
					     1366×700 mit fünf Positionen im Korb: Die Spalte wurde 1288 px hoch und
					     ragte 826 px unter den Fensterrand — der Absenden-Knopf war nur noch zu
					     erreichen, indem man die ganze Bestellbedarfs-Liste daneben (105 Titel)
					     nach unten scrollte, bis der klebende Rahmen endete. Erreichbar im
					     technischen Sinn, unbrauchbar im täglichen.
					     Höhengrenze plus eigener Scrollbalken: Wird die Spalte zu hoch, scrollt sie
					     in sich selbst. Der Warenkorb bleibt damit dort bedienbar, wo er steht.
					     Verankert im E2E-Gate e2e/bestellung-erreichbar.spec.js. -->
					<div
						bind:this={rail}
						style:--rail-max={railMaxHeight}
						class="lg:col-span-5 xl:col-span-4 space-y-4 lg:sticky lg:top-2 lg:max-h-(--rail-max) lg:overflow-y-auto"
					>
						<PrintSuggestion {printSuggestion} onPrint={handlePrintSuggestion} {offeneEtiketten} />
						<OrderCreationPanel />
					</div>
				</div>
			</div>
		{/if}
	{/if}

	{#if activeTab === 'lieferanten'}
		<SupplierManager
			suppliers={orderStore.suppliers}
			onAddSupplier={(name, email, customerNumber, liefertMitBarcode, istStandard) =>
				orderStore.addSupplier(name, email, customerNumber, liefertMitBarcode, istStandard)}
			onEditSupplier={(id, name, email, customerNumber, liefertMitBarcode, istStandard) =>
				orderStore.editSupplier(id, name, email, customerNumber, liefertMitBarcode, istStandard)}
			onRemoveSupplier={(id) => orderStore.removeSupplier(id)}
		/>
	{/if}

	{#if activeTab === 'historie'}
		<BestellHistorie />
	{/if}

	{#if activeTab === 'berichte'}
		<BestellBerichte suppliers={orderStore.suppliers} />
	{/if}

	{#if activeTab === 'klassensaetze'}
		<KlassensatzReservierungen />
	{/if}
</div>

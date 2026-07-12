<script>
	import { onMount } from 'svelte';
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

	onMount(() => {
		orderStore.init();
	});

	/** @param {any[]} receivedItems */
	async function handleShipmentReceived(receivedItems) {
		showWareneingang = false;
		const needsPrinting = receivedItems.filter((item) => !item.etikett_gedruckt);
		printSuggestion = needsPrinting.length > 0 ? needsPrinting : null;
		showGreenFade = true;
		await orderStore.loadIncomingShipments();
		orderStore.loadRecommendations();
		setTimeout(() => {
			showGreenFade = false;
		}, 1500);
	}

	function handlePrintSuggestion() {
		printQueue.copies = printSuggestion;
		printSuggestion = null;
	}
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
			<div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start overflow-y-auto">
				<OrderCreationPanel />

				<div class="lg:col-span-4 space-y-6">
					<PrintSuggestion {printSuggestion} onPrint={handlePrintSuggestion} />

					<IncomingShipments
						incomingShipments={orderStore.incomingShipments}
						{showGreenFade}
						onOpenWareneingang={() => (showWareneingang = true)}
					/>

					<OrderRecommendations
						recommendations={orderStore.recommendations}
						onAddToCart={(book) => orderStore.addToCart(book)}
					/>
				</div>
			</div>
		{/if}
	{/if}

	{#if activeTab === 'lieferanten'}
		<SupplierManager
			suppliers={orderStore.suppliers}
			onAddSupplier={(name, email, customerNumber) =>
				orderStore.addSupplier(name, email, customerNumber)}
			onEditSupplier={(id, name, email, customerNumber) =>
				orderStore.editSupplier(id, name, email, customerNumber)}
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

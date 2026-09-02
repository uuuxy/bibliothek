<script>
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import { useUnifiedInventory } from './useUnifiedInventory.svelte.js';
	import InventoryStartModal from './components/InventoryStartModal.svelte';
	import InventoryFinishModal from './components/InventoryFinishModal.svelte';
	import Button from './components/ui/Button.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import FehlbestandBericht from './components/inventur/FehlbestandBericht.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import { Check, ClipboardCheck, Plus, ScanBarcode, TriangleAlert, X } from '@lucide/svelte';

	const inventoryState = useUnifiedInventory();

	// $state, nicht `let`: Diese drei werden per bind:this gefüllt, und alle drei werden
	// unten in einem $effect gelesen. Ohne $state ist die Zuweisung durch bind:this nicht
	// reaktiv — der Effekt läuft dann genau einmal, nämlich bevor das Element existiert,
	// und danach nie wieder.
	//
	// Beim Barcode-Feld ist das der teure Fall: Es wird erst gerendert, wenn die Inventur
	// auf 'active' steht. Der Effekt darunter feuerte also mit barcodeInputEl === undefined,
	// tat nichts, und wurde nie erneut ausgelöst — das Feld blieb ohne Fokus und jeder Scan
	// lief ins Leere, ohne dass irgendwo ein Fehler erschien.
	let startDialog = $state();
	let finishDialog = $state();
	let barcodeInputEl = $state();

	$effect(() => {
		if (inventoryState.showStartModal && startDialog) {
			startDialog.showModal();
		} else if (!inventoryState.showStartModal && startDialog) {
			startDialog.close();
		}
	});

	$effect(() => {
		if (inventoryState.showFinishModal && finishDialog) {
			finishDialog.showModal();
		} else if (!inventoryState.showFinishModal && finishDialog) {
			finishDialog.close();
		}
	});

	$effect(() => {
		if (inventoryState.status === 'active' && barcodeInputEl && !inventoryState.isScanning) {
			barcodeInputEl.focus();
		}
	});

	onMount(async () => {
		await inventoryState.loadSignaturen();
		await inventoryState.loadFaecher();
		await inventoryState.loadOffeneSessions();
		await inventoryState.loadAbgeschlosseneInventuren();
	});

	/** @param {string} iso */
	function datumKurz(iso) {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return iso;
		return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
	}

	function focusInput() {
		if (barcodeInputEl) barcodeInputEl.focus();
	}
</script>

<PageShell>
	<!-- Steht GANZ OBEN und ueberlebt das Zuruecksetzen: Der Bericht ist das Ergebnis der
	     Arbeit, nicht eine Randnotiz. Vorher endete eine Inventur mit einer Zahl im Toast,
	     die drei Sekunden spaeter weg war. -->
	{#if inventoryState.fehlbestand.length > 0}
		<FehlbestandBericht
			eintraege={inventoryState.fehlbestand}
			label={inventoryState.fehlbestandLabel}
			onSchliessen={inventoryState.fehlbestandSchliessen}
			onGefunden={inventoryState.fehlbestandGefunden}
			onEndgueltigLoeschen={inventoryState.fehlbestandEndgueltigLoeschen}
		/>
	{/if}

	{#if inventoryState.status === 'idle'}
		<div class="p-12 text-center flex flex-col items-center justify-center space-y-6">
			<div class="w-20 h-20 bg-blue-50 text-blue-500 rounded-full flex items-center justify-center">
				<ClipboardCheck class="w-10 h-10" aria-hidden="true" />
			</div>
			<div>
				<h3 class="text-xl font-bold text-slate-900">Keine Inventur aktiv</h3>
				<p class="text-slate-500 mt-2 max-w-md mx-auto">
					Starte einen neuen Inventur-Lauf. Du kannst entweder die gesamte Bibliothek prüfen oder
					gezielt nach einer bestimmten Signatur / Kategorie scannen.
				</p>
			</div>
			<Button size="lg" onclick={() => (inventoryState.showStartModal = true)} class="px-6">
				<Plus class="w-5 h-5" aria-hidden="true" />
				<span>Neue Bestandsprüfung starten</span>
			</Button>

			{#if inventoryState.errorMessage}
				<div
					class="w-full max-w-lg mx-auto p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800"
				>
					{inventoryState.errorMessage}
				</div>
			{/if}

			{#if inventoryState.offeneSessions.length > 0}
				<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
					<h4 class="text-sm font-semibold text-slate-500">Laufende Inventuren</h4>
					{#each inventoryState.offeneSessions as session (session.session_id)}
						<div
							class="flex items-center justify-between gap-3 p-3 bg-amber-50 border border-amber-200 rounded-lg"
						>
							<div class="min-w-0">
								<div class="font-semibold text-slate-800 truncate">{session.label}</div>
								<div class="text-xs text-slate-500">
									{session.erfasst} / {session.erwartet} erfasst · seit {session.gestartet_am?.slice(
										0,
										16
									)}
								</div>
							</div>
							<div class="flex items-center gap-2 shrink-0">
								<Button size="sm" onclick={() => inventoryState.resumeSession(session)}>
									Fortsetzen
								</Button>
								<Button
									variant="secondary"
									size="sm"
									onclick={() => inventoryState.verwerfeSession(session)}
								>
									Verwerfen
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Frühere Inventuren.
			     Der Fehlbestandsbericht entstand bisher nur aus der Antwort des Abschlusses
			     und lebte im Arbeitsspeicher DIESES Browsers: Neu laden — weg. Der Kollege am
			     zweiten Arbeitsplatz, der mit der Liste ins Regal geht, sah ihn nie. Die Daten
			     liegen dauerhaft auf dem Server; hier ist der Weg zurück zu ihnen. -->
			{#if inventoryState.abgeschlosseneInventuren.length > 0}
				<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
					<h4 class="text-sm font-semibold text-slate-500">Frühere Inventuren</h4>
					{#each inventoryState.abgeschlosseneInventuren as inventur (inventur.session_id)}
						<div
							class="flex items-center justify-between gap-3 px-3 py-2.5 border-b border-slate-200"
						>
							<div class="min-w-0">
								<div class="font-semibold text-slate-800 truncate">{inventur.label}</div>
								<div class="text-xs text-slate-500">
									{datumKurz(inventur.abgeschlossen_am)} · {inventur.erfasst} erfasst ·
									{#if inventur.verluste > 0}
										<span class="text-rose-600 font-semibold">{inventur.verluste} fehlend</span>
									{:else}
										vollständig
									{/if}
								</div>
							</div>
							<Button
								variant="secondary"
								size="sm"
								class="shrink-0"
								disabled={inventoryState.ladeFruehereLaeuft}
								onclick={() => inventoryState.zeigeFrueherenFehlbestand(inventur)}
							>
								Fehlbestand
							</Button>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{:else}
		<div class="space-y-6">
			<!-- Progress & Stats -->
			<div class="p-6">
				<div class="flex justify-between items-end mb-4">
					<div>
						<span class="text-sm font-semibold text-slate-500">Aktueller Fortschritt</span>
						<div class="text-2xl font-bold text-slate-900 mt-1">
							{inventoryState.stats.erfasst} / {inventoryState.stats.erwartet}
							<span class="text-base font-medium text-slate-400">erfasst</span>
						</div>
					</div>
					<div class="text-3xl font-bold text-blue-600">{inventoryState.getProgressPercent()}%</div>
				</div>
				<div class="w-full bg-slate-100 rounded-full h-3 overflow-hidden">
					<div
						class="bg-blue-600 h-3 rounded-full transition-all duration-500 ease-out"
						style="width: {inventoryState.getProgressPercent()}%"
					></div>
				</div>
			</div>

			<!-- Dieselbe 48-px-Suchpille wie die Ausleihe-Omnibox: Werkzeug der Seite, kein
			     Datenfeld in einer Leiste (Peter, 25.08.: „deutlich kleiner als in Ausleihe"). -->
			<form
				onsubmit={(e) => {
					e.preventDefault();
					inventoryState.handleScan(inventoryState.barcodeInput, focusInput);
				}}
			>
				<Suchpille
					id="inventur-scan"
					bind:element={barcodeInputEl}
					bind:wert={inventoryState.barcodeInput}
					etikett="Barcode scannen"
					platzhalter="Barcode scannen..."
					disabled={inventoryState.isScanning}
				>
					{#snippet nachlaufend()}
						{#if inventoryState.isScanning}
							<div
								class="h-5 w-5 shrink-0 animate-spin rounded-full border-2 border-slate-300 border-t-blue-600"
							></div>
						{:else}
							<ScanBarcode class="h-5 w-5 shrink-0 text-slate-500" aria-hidden="true" />
						{/if}
					{/snippet}
				</Suchpille>
			</form>
			<!-- Feedback Area -->
			{#if inventoryState.lastScan}
				<div
					transition:slide
					class="rounded-2xl p-6 border {inventoryState.lastScan.success &&
					inventoryState.lastScan.warnings.length === 0
						? 'bg-emerald-50 border-emerald-200'
						: !inventoryState.lastScan.success
							? 'bg-red-50 border-red-200'
							: 'bg-amber-50 border-amber-200'}"
				>
					<div class="flex items-start space-x-4">
						{#if inventoryState.lastScan.success && inventoryState.lastScan.warnings.length === 0}
							<div class="p-2 bg-emerald-100 rounded-full text-emerald-600 shrink-0">
								<Check class="w-6 h-6" aria-hidden="true" />
							</div>
						{:else if !inventoryState.lastScan.success}
							<div class="p-2 bg-red-100 rounded-full text-red-600 shrink-0">
								<X class="w-6 h-6" aria-hidden="true" />
							</div>
						{:else}
							<div class="p-2 bg-amber-100 rounded-full text-amber-600 shrink-0">
								<TriangleAlert class="w-6 h-6" aria-hidden="true" />
							</div>
						{/if}

						<div class="flex-1">
							<h4
								class="text-lg font-bold {inventoryState.lastScan.success &&
								inventoryState.lastScan.warnings.length === 0
									? 'text-emerald-900'
									: !inventoryState.lastScan.success
										? 'text-red-900'
										: 'text-amber-900'}"
							>
								{inventoryState.lastScan.title}
							</h4>
							<p
								class="text-sm font-medium mt-1 {inventoryState.lastScan.success &&
								inventoryState.lastScan.warnings.length === 0
									? 'text-emerald-700'
									: !inventoryState.lastScan.success
										? 'text-red-700'
										: 'text-amber-700'}"
							>
								Barcode: {inventoryState.lastScan.barcode}
							</p>

							{#if inventoryState.lastScan.warnings.length > 0}
								<ul class="mt-3 space-y-1">
									{#each inventoryState.lastScan.warnings as warn, _i (_i)}
										<li
											class="flex items-start text-sm {!inventoryState.lastScan.success
												? 'text-red-800'
												: 'text-amber-800'}"
										>
											<span class="mr-2 mt-0.5">•</span>
											<span>{warn}</span>
										</li>
									{/each}
								</ul>
							{/if}
						</div>
					</div>
				</div>
			{/if}

			<div class="pt-8 border-t border-slate-200 flex justify-end">
				<Button
					variant="danger"
					size="lg"
					onclick={() => (inventoryState.showFinishModal = true)}
					class="px-6"
				>
					Inventur abschließen
				</Button>
			</div>
		</div>
	{/if}
</PageShell>

<!-- Start Modal -->
<InventoryStartModal
	bind:dialogEl={startDialog}
	state={inventoryState}
	onClose={() => {
		inventoryState.showStartModal = false;
		inventoryState.clearError();
	}}
	onStart={inventoryState.startInventory}
/>

<!-- Finish Modal -->
<InventoryFinishModal
	bind:dialogEl={finishDialog}
	state={inventoryState}
	onClose={() => (inventoryState.showFinishModal = false)}
	onFinish={inventoryState.finishInventory}
/>

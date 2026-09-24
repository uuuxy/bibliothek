<script>
	import { onMount } from 'svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { slide } from 'svelte/transition';
	import { useUnifiedInventory } from './useUnifiedInventory.svelte.js';
	import InventoryStartModal from './components/InventoryStartModal.svelte';
	import InventoryFinishModal from './components/InventoryFinishModal.svelte';
	import Button from './components/ui/Button.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import FehlbestandBericht from './components/inventur/FehlbestandBericht.svelte';
	import ScanRueckmeldung from './components/inventur/ScanRueckmeldung.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import { ClipboardCheck, Plus, ScanBarcode } from '@lucide/svelte';

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
			<!-- Neutral wie der Leerzustand der Abgängerliste (AbgaengerTabelle.svelte): Die
			     Hauptfarbe bleibt dem einen gefüllten Knopf darunter. -->
			<div
				class="w-20 h-20 rounded-full bg-surface-container-low border border-outline-variant text-on-surface-variant flex items-center justify-center"
			>
				<ClipboardCheck class="w-10 h-10" aria-hidden="true" />
			</div>
			<div>
				<h3 class="text-xl font-bold text-on-surface">Keine Inventur aktiv</h3>
				<p class="text-on-surface-variant mt-2 max-w-md mx-auto">
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
					class="w-full max-w-lg mx-auto p-3 bg-warning-container rounded-lg text-sm text-on-warning-container"
				>
					{inventoryState.errorMessage}
				</div>
			{/if}

			{#if inventoryState.offeneSessions.length > 0}
				<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
					<h4 class="text-sm font-semibold text-on-surface-variant">Laufende Inventuren</h4>
					{#each inventoryState.offeneSessions as session (session.session_id)}
						<div
							class="flex items-center justify-between gap-3 p-3 bg-warning-container text-on-warning-container rounded-lg"
						>
							<div class="min-w-0">
								<div class="font-semibold truncate">{session.label}</div>
								<div class="text-xs">
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
					<h4 class="text-sm font-semibold text-on-surface-variant">Frühere Inventuren</h4>
					{#each inventoryState.abgeschlosseneInventuren as inventur (inventur.session_id)}
						<div
							class="flex items-center justify-between gap-3 px-3 py-2.5 border-b border-outline-variant"
						>
							<div class="min-w-0">
								<div class="font-semibold text-on-surface truncate">{inventur.label}</div>
								<div class="text-xs text-on-surface-variant">
									{datumKurz(inventur.abgeschlossen_am)} · {inventur.erfasst} erfasst ·
									{#if inventur.verluste > 0}
										<span class="text-error font-semibold">{inventur.verluste} fehlend</span>
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
						<span class="text-sm font-semibold text-on-surface-variant">Aktueller Fortschritt</span>
						<div class="text-2xl font-bold text-on-surface mt-1">
							{inventoryState.stats.erfasst} / {inventoryState.stats.erwartet}
							<span class="text-base font-medium text-on-surface-variant">erfasst</span>
						</div>
					</div>
					<div class="text-3xl font-bold text-primary">{inventoryState.getProgressPercent()}%</div>
				</div>
				<!-- Spur und Anzeige wie der lineare Fortschritt in M3 (material-web.dev, Tokens
				     track-color: surface-container-highest, active-indicator-color: primary). -->
				<div class="w-full bg-surface-container-highest rounded-full h-3 overflow-hidden">
					<div
						class="bg-primary h-3 rounded-full transition-all duration-500 ease-out"
						style="width: {inventoryState.getProgressPercent()}%"
					></div>
				</div>
			</div>

			<!-- Dieselbe 48-px-Suchpille wie die Ausleihe-Omnibox: Werkzeug der Seite, kein
			     Datenfeld in einer Leiste (Absprache vom 25.08.: „deutlich kleiner als in Ausleihe"). -->
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
							<Ladekreis size="md" />
						{:else}
							<ScanBarcode class="h-5 w-5 shrink-0 text-on-surface-variant" aria-hidden="true" />
						{/if}
					{/snippet}
				</Suchpille>
			</form>
			<!-- Die Rückmeldung zum letzten Scan (ScanRueckmeldung.svelte: Erfolg, Warnung,
			     Fehler auf eigenen Farbrollen). Das Einblenden bleibt hier, am {#if}. -->
			{#if inventoryState.lastScan}
				<div transition:slide>
					<ScanRueckmeldung scan={inventoryState.lastScan} />
				</div>
			{/if}

			<div class="pt-8 border-t border-outline-variant flex justify-end">
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

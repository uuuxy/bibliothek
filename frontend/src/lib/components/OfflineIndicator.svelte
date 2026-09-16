<!-- @component OfflineIndicator — ein Band, kein Vollbild.
     Stufe 3 des Offline-Baus, Punkt (a): An der Theke darf der Verbindungsverlust das
     Scannen nicht anhalten. Bis zum 16.09.2026 stand hier ein `fixed`-Block mit
     Riesenschrift und grossen Knoepfen; der Stufe-1-Nachweis am echten Chrome ergab:
     "der balken ist so gross, dass ich nichts buchen kann".

     DREI Lautstaerken statt einer, weil die drei Lagen nicht gleich schlimm sind:
       1. Warteschlange nicht lesbar — Offline-Scans gehen VERLOREN. Laut (Fehlerrolle).
       2. Vorgaenge liegen auf diesem Rechner — sichtbar, mit Sicherung. Mittel.
       3. Nur keine Verbindung, nichts offen — eine Zeile, leise. Bis zum 16.09.2026 rief
          dieser Fall "bitte Sicherung speichern", obwohl es nichts zu sichern gab
          ("0 Vorgaenge nur auf diesem Rechner").

     Farben aus den M3-Rollen statt aus der Palette (Farb-Ratsche zaehlt die Summe). -->
<script>
	import { offlineSync } from '../stores/offlineSync.svelte.js';
	import { CloudOff, Download, TriangleAlert, Upload } from '@lucide/svelte';
	import { toastStore } from '../stores/toastStore.svelte.js';
	import Button from './ui/Button.svelte';

	// Der Herzschlag kommt von aussen (App.svelte): Bis zum 16.09.2026 legte SEIN Ausfall
	// ein eigenes Vollbild ueber die Seite. Dieselbe Lage, dieselbe Zeile — nicht zwei.
	/** @type {{ verbindungVerloren?: boolean }} */
	let { verbindungVerloren = false } = $props();

	let ohneNetz = $derived(offlineSync.isOffline || verbindungVerloren);
	let sichtbar = $derived(
		offlineSync.pendingCount > 0 || ohneNetz || offlineSync.warteschlangeFehler
	);

	async function handleBackup() {
		try {
			await offlineSync.exportQueueAsJSON();
		} catch (err) {
			console.error('Sicherung nicht möglich:', err);
			toastStore.addToast(
				'Sicherung nicht möglich: Die Warteschlange auf diesem Rechner ist nicht lesbar.',
				'error'
			);
		}
	}

	/** @type {HTMLInputElement | null} */
	let fileInput = $state(null);

	// Mehrere Dateien auf einmal: Bei zehn Kiosk-Rechnern liegen im Sicherungsordner
	// zehn Dateien. Sie einzeln auszuwählen lädt dazu ein, eine zu übersehen.
	async function handleFileSelect(e) {
		const input = /** @type {HTMLInputElement} */ (e.target);
		const files = [...(input.files ?? [])];
		if (files.length === 0) return;

		let gesamt = 0;
		const fehler = [];
		for (const file of files) {
			try {
				gesamt += await offlineSync.importQueueFromJSON(file);
			} catch (err) {
				fehler.push(`${file.name}: ${err instanceof Error ? err.message : String(err)}`);
			}
		}

		if (gesamt > 0) {
			toastStore.addToast(
				`${gesamt} Vorgang/Vorgänge aus ${files.length - fehler.length} Datei(en) übernommen — werden jetzt übertragen.`,
				'success'
			);
		}
		if (fehler.length > 0) {
			toastStore.addToast(fehler.join(' · '), 'error');
		}
		input.value = ''; // reset
	}
</script>

{#if sichtbar}
	<div
		class="fixed top-0 right-0 left-0 z-9999 border-b {offlineSync.warteschlangeFehler
			? 'bg-error text-on-error border-error'
			: offlineSync.pendingCount > 0
				? 'bg-error-container text-on-error-container border-outline-variant'
				: 'bg-surface-container-low text-on-surface-variant border-outline-variant'}"
		role="status"
		aria-live="polite"
	>
		<div class="mx-auto flex max-w-7xl items-center gap-3 px-4 py-1.5">
			{#if offlineSync.warteschlangeFehler}
				<TriangleAlert size={18} strokeWidth={2.5} class="shrink-0" />
			{:else}
				<CloudOff size={18} strokeWidth={2.5} class="shrink-0" />
			{/if}

			<p class="min-w-0 flex-1 truncate text-sm font-semibold">
				{#if offlineSync.warteschlangeFehler}
					Warteschlange nicht lesbar — Offline-Scans werden auf diesem Rechner NICHT gespeichert.
					Bitte an einem anderen Rechner weiterarbeiten.
				{:else if offlineSync.pendingCount > 0}
					<!-- „Vorgänge", nicht „Vorgange": Die Mehrzahl wurde bis zum 16.09.2026 durch
					     Anhaengen eines „e" an „Vorgang" gebildet, der Umlaut fiel weg. Auf dem
					     Bildschirmfoto des Stufe-1-Nachweises steht „0 Vorgange nur auf diesem
					     Rechner" — gefunden hat es der erste Test dieser Datei. -->
					{offlineSync.pendingCount}
					{offlineSync.pendingCount === 1 ? 'Vorgang' : 'Vorgänge'} nur auf diesem Rechner — noch nicht
					im System.
					{#if offlineSync.isSyncing}
						Wird übertragen …
					{:else if ohneNetz}
						Sobald die Verbindung zurück ist, geschieht das von selbst. Vorher bitte sichern.
					{/if}
				{:else}
					Keine Verbindung — Scannen geht weiter, die Buchungen folgen von selbst.
				{/if}
			</p>

			{#if offlineSync.pendingCount > 0}
				<Button variant="secondary" size="sm" onclick={handleBackup} class="shrink-0">
					<Download size={16} strokeWidth={2.5} />
					Sicherung speichern
				</Button>
			{/if}

			<!-- Verstecktes File Input für Import -->
			<input
				type="file"
				accept=".json"
				multiple
				bind:this={fileInput}
				onchange={handleFileSelect}
				class="hidden"
			/>

			<Button
				variant="ghost"
				size="sm"
				onclick={() => fileInput?.click()}
				class="shrink-0"
				title="Sicherung eines anderen Rechners übernehmen"
			>
				<Upload size={16} strokeWidth={2.5} />
				Sicherung einspielen
			</Button>
		</div>
	</div>
{/if}

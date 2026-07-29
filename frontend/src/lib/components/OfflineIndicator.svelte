<script>
	import { offlineSync } from '../stores/offlineSync.svelte.js';
	import { CloudOff, Download, Upload } from '@lucide/svelte';
	import { toastStore } from '../stores/toastStore.svelte.js';
	import Button from './ui/Button.svelte';

	// isOffline and global events are now handled centrally in offlineSync.svelte.js

	async function handleBackup() {
		await offlineSync.exportQueueAsJSON();
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

{#if offlineSync.pendingCount > 0 || offlineSync.isOffline}
	<div
		class="fixed top-0 left-0 right-0 z-9999 bg-rose-600 text-white shadow-2xl border-b-4 border-rose-800 animate-slide-down"
	>
		<div
			class="max-w-7xl mx-auto px-6 py-4 flex flex-col md:flex-row items-center justify-between gap-4"
		>
			<div class="flex items-center gap-4">
				<div class="bg-rose-500/50 p-3 rounded-2xl shrink-0">
					<CloudOff size={32} strokeWidth={2.5} class="text-white" />
				</div>
				<div>
					<!-- Anweisung statt Angst: "Nicht ausschalten" hält niemand bis Feierabend
					     durch, und an einem Rechner, der beim Herunterfahren zurückgesetzt
					     wird, ist die Sicherung der einzige Weg, der wirklich hilft. -->
					<h1 class="text-xl md:text-2xl font-black tracking-tight drop-shadow-md">
						Offline — bitte Sicherung speichern, bevor dieser Rechner ausgeschaltet wird
					</h1>
					<p class="text-rose-100 font-semibold mt-1">
						{offlineSync.pendingCount} Vorgang{offlineSync.pendingCount === 1 ? '' : 'e'} nur auf diesem
						Rechner — noch nicht im System, für die anderen Arbeitsplätze unsichtbar.
						{#if offlineSync.isSyncing}
							<span class="ml-2">Wird übertragen …</span>
						{:else if offlineSync.isOffline}
							Sobald die Verbindung zurück ist, geschieht das von selbst.
						{/if}
					</p>
				</div>
			</div>

			<div class="flex items-center gap-3 shrink-0">
				{#if offlineSync.pendingCount > 0}
					<Button
						variant="secondary"
						size="lg"
						onclick={handleBackup}
						class="px-5 border-rose-200 text-rose-700 shadow-lg hover:bg-rose-50"
					>
						<Download size={18} strokeWidth={3} />
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
					variant="danger-solid"
					size="lg"
					onclick={() => fileInput?.click()}
					class="bg-rose-700 border-rose-800 text-rose-50 shadow-inner hover:bg-rose-800"
					title="Backup einspielen (falls du an einem anderen PC den Stand nachträgst)"
				>
					<Upload size={18} strokeWidth={2.5} />
					Offline-Backup einspielen
				</Button>
			</div>
		</div>
	</div>
{/if}

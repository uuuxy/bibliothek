<script>
	import Modal from './Modal.svelte';
	import { apiClient } from './apiFetch.js';
	import Button from './components/ui/Button.svelte';
	import StudentFormFelder from './components/StudentFormFelder.svelte';
	import { TriangleAlert } from '@lucide/svelte';

	let { open = false, readerGroups = [], onclose, onsuccess } = $props();

	let newVorname = $state('');
	let newNachname = $state('');
	let newKlasse = $state('');
	let customKlasseInput = $state(false);
	let newBarcode = $state('');
	let newGeburtsdatum = $state('');
	let createError = $state('');
	let duplicateConflict = $state(false);
	let isSaving = $state(false);

	// Watch for open state changes to reset form
	$effect(() => {
		if (open) {
			newVorname = '';
			newNachname = '';
			newKlasse = '';
			newBarcode = '';
			newGeburtsdatum = '';
			createError = '';
			duplicateConflict = false;
			customKlasseInput = false;
		}
	});

	async function createStudent() {
		createError = '';
		duplicateConflict = false;
		if (!newVorname.trim() || !newNachname.trim() || !newKlasse.trim()) {
			createError = 'Vorname, Nachname und Klasse sind Pflichtfelder.';
			return;
		}
		isSaving = true;
		try {
			const res = await apiClient.post('/api/schueler', {
				vorname: newVorname.trim(),
				nachname: newNachname.trim(),
				klasse: newKlasse.trim(),
				barcode_id: newBarcode.trim(),
				geburtsdatum: newGeburtsdatum.trim() || null
			});
			if (res.ok) {
				onsuccess?.();
			} else {
				if (res.status === 409) {
					duplicateConflict = true;
				} else {
					const errText = await res.text();
					try {
						const errObj = JSON.parse(errText);
						createError = errObj.error || 'Fehler beim Anlegen des Schülers.';
					} catch {
						createError = errText || 'Fehler beim Anlegen des Schülers.';
					}
				}
			}
		} catch (err) {
			createError = 'Netzwerkfehler beim Anlegen des Schülers.';
			console.error(err);
		} finally {
			isSaving = false;
		}
	}

	function handleClose() {
		onclose?.();
	}
</script>

<Modal {open} onclose={handleClose} size="md">
	{#snippet header()}
		<h3 class="text-sm font-bold text-slate-800">Neuen Schüler anlegen</h3>
	{/snippet}
	<div class="p-6 space-y-4">
		{#if duplicateConflict}
			<div
				class="p-4 bg-amber-50 border border-amber-200 rounded-xl flex items-start gap-3 text-sm font-semibold text-amber-800"
			>
				<TriangleAlert class="h-5 w-5 text-amber-500 shrink-0 mt-0.5" aria-hidden="true" />
				<div>
					<p>Achtung: Ein Schüler mit diesem Namen und Geburtsdatum existiert bereits im System.</p>
					<p class="text-xs font-normal mt-1 opacity-80">
						Bitte überprüfe die Daten, um Duplikate zu vermeiden. Wurde der Schüler eventuell
						bereits angelegt oder importiert?
					</p>
				</div>
			</div>
		{/if}

		{#if createError}
			<div
				class="p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600"
			>
				{createError}
			</div>
		{/if}

		<StudentFormFelder
			bind:vorname={newVorname}
			bind:nachname={newNachname}
			bind:geburtsdatum={newGeburtsdatum}
			bind:klasse={newKlasse}
			bind:barcode={newBarcode}
			bind:freieKlasse={customKlasseInput}
			{readerGroups}
		/>

		<div class="flex justify-end gap-3 pt-2 border-t border-slate-100">
			<Button variant="secondary" onclick={handleClose} disabled={isSaving}>Abbrechen</Button>
			<Button onclick={createStudent} disabled={isSaving}>
				{isSaving ? 'Speichern...' : 'Speichern'}
			</Button>
		</div>
	</div>
</Modal>

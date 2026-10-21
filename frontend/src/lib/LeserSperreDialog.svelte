<!-- @component LeserSperreDialog — Ausleihe eines Schülers sperren oder freigeben.
     Kopfzeile und Schließen-Knopf stellt Dialog.svelte. -->
<script>
	import { apiClient } from './apiFetch.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { LockOpen, Lock, CircleAlert } from '@lucide/svelte';
	import Dialog from './Dialog.svelte';
	import Button from './components/ui/Button.svelte';
	import { ausleiheGesperrt } from './sperrStatus.js';
	import { erzeugeDialogAnfrage } from './components/ui/dialogAnfrage.svelte.js';

	/** @type {{ open: boolean, leser: any, onsuccess: (updatedProfile: any) => void }} */
	let { open = $bindable(false), leser, onsuccess } = $props();

	let laeuft = $state(false);
	let fehler = $state('');
	let reason = $state('');

	// Aufheben, sobald eine Sperre am Leser besteht — von Hand oder die der Ehemaligen; die Tür
	// nimmt beide weg (api/leser/leser_sperre.go). Dasselbe Prädikat wie der
	// Knopf, der dieses Fenster öffnet (LeserKontoStatus).
	let gesperrt = $derived(ausleiheGesperrt(leser));
	// Beim Sperren (nicht beim Entsperren) ist ein Grund Pflicht — deckt sich mit dem
	// Backend-Check und dem DB-Constraint chk_schueler_block_reason.
	let wirdGesperrt = $derived(leser && !gesperrt);

	// Das Modal bleibt (via bind:open) dauerhaft gemountet, der State überlebt also das
	// Schließen. Ohne diesen Reset blitzte beim nächsten Öffnen — ggf. für einen anderen
	// Schüler — die alte Fehlermeldung und der alte Grund kurz auf.
	$effect(() => {
		if (!open) {
			fehler = '';
			reason = '';
		}
	});

	const anfrage = erzeugeDialogAnfrage(() => open);
	const zeigeFehler = (/** @type {string} */ text) => (fehler = text);

	async function bestaetige() {
		if (!leser) return;
		if (wirdGesperrt && reason.trim() === '') {
			fehler = 'Bitte einen Grund für die Sperre angeben.';
			return;
		}
		const name = `${leser.vorname} ${leser.nachname}`;
		const sperren = !gesperrt;
		laeuft = true;
		fehler = '';
		const lauf = anfrage.beginne();
		try {
			const res = await apiClient.patch(`/api/admin/students/${leser.id}/lock`, {
				is_locked: sperren,
				reason: reason.trim()
			});

			if (res.ok) {
				const updated = await res.json();
				lauf.erfolg(sperren ? `Ausleihe für ${name} gesperrt.` : `Sperre für ${name} aufgehoben.`);
				onsuccess(updated);
				if (lauf.nochOffen()) open = false;
			} else {
				const err = await res.json().catch(() => ({}));
				lauf.fehler(err.error || 'Fehler beim Aktualisieren der Sperre.', zeigeFehler);
			}
		} catch {
			lauf.fehler('Netzwerkfehler beim Aktualisieren der Sperre.', zeigeFehler);
		} finally {
			laeuft = false;
		}
	}
</script>

<Dialog {open} onclose={() => (open = false)} beschriftetDurch="sperre-titel">
	{#snippet header()}
		<h3
			id="sperre-titel"
			class="text-lg font-bold {gesperrt ? 'text-success' : 'text-error'} flex items-center gap-2"
		>
			{#if gesperrt}
				<LockOpen class="w-5 h-5" aria-hidden="true" />
				Sperre aufheben
			{:else}
				<Lock class="w-5 h-5" aria-hidden="true" />
				Ausleihe sperren
			{/if}
		</h3>
	{/snippet}

	<div class="px-6 py-6 text-on-surface-variant space-y-4">
		<p class="text-sm font-medium leading-relaxed">
			{#if gesperrt}
				Möchten Sie die Ausleihe für <span class="font-bold text-on-surface"
					>{leser.vorname} {leser.nachname}</span
				> wirklich freigeben?
			{:else}
				Möchten Sie die Ausleihe für <span class="font-bold text-on-surface"
					>{leser?.vorname} {leser?.nachname}</span
				> wirklich sperren?
			{/if}
		</p>

		{#if wirdGesperrt}
			<label class="block space-y-1.5">
				<span class="text-xs font-bold text-on-surface"
					>Grund der Sperre <span class="text-error">*</span></span
				>
				<textarea
					bind:value={reason}
					rows="2"
					disabled={laeuft}
					placeholder="z. B. wiederholt Bücher nicht zurückgegeben"
					class="w-full px-3 py-2 text-sm border border-outline-variant rounded-xl focus:outline-none focus:ring-2 focus:ring-error/30 disabled:opacity-50"
				></textarea>
			</label>
		{/if}

		{#if fehler}
			<div
				role="alert"
				class="p-3 bg-error-container text-on-error-container rounded-xl flex gap-2 items-start"
			>
				<CircleAlert class="w-4 h-4 mt-0.5 shrink-0" aria-hidden="true" />
				<p class="text-xs font-bold leading-tight">{fehler}</p>
			</div>
		{/if}
	</div>

	<div class="px-6 py-4 border-t border-outline-variant flex justify-end gap-3">
		<Button variant="secondary" onclick={() => (open = false)}>Abbrechen</Button>
		<Button variant={gesperrt ? 'success' : 'danger-solid'} onclick={bestaetige} disabled={laeuft}>
			{#if laeuft}
				<Ladekreis size="sm" farbe="aktuell" />
				Wird verarbeitet...
			{:else}
				Bestätigen
			{/if}
		</Button>
	</div>
</Dialog>

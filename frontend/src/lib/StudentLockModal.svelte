<!-- @component StudentLockModal — Ausleihe eines Schülers sperren oder freigeben.

     Seit 07.09.2026 auf Modal.svelte (Register 05.09.); Kopfzeile und Schließen-Knopf
     stellt das Bauteil. -->
<script>
	import { apiClient } from './apiFetch.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { Unlock, Lock, AlertCircle } from '@lucide/svelte';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';

	/** @type {{ open: boolean, profile: any, onsuccess: (updatedProfile: any) => void }} */
	let { open = $bindable(false), profile, onsuccess } = $props();

	let isSubmitting = $state(false);
	let errorMsg = $state('');
	let reason = $state('');

	// Beim Sperren (nicht beim Entsperren) ist ein Grund Pflicht — deckt sich mit dem
	// Backend-Check und dem DB-Constraint chk_schueler_block_reason.
	let willLock = $derived(profile && !profile.is_manually_blocked);

	// Das Modal bleibt (via bind:open) dauerhaft gemountet, der State überlebt also das
	// Schließen. Ohne diesen Reset blitzte beim nächsten Öffnen — ggf. für einen anderen
	// Schüler — die alte Fehlermeldung und der alte Grund kurz auf.
	$effect(() => {
		if (!open) {
			errorMsg = '';
			reason = '';
		}
	});

	async function handleConfirm() {
		if (!profile) return;
		if (willLock && reason.trim() === '') {
			errorMsg = 'Bitte einen Grund für die Sperre angeben.';
			return;
		}
		isSubmitting = true;
		errorMsg = '';
		try {
			const res = await apiClient.patch(`/api/admin/students/${profile.id}/lock`, {
				is_locked: !profile.is_manually_blocked,
				reason: reason.trim()
			});

			if (res.ok) {
				const updated = await res.json();
				onsuccess(updated);
				reason = '';
				open = false;
			} else {
				const err = await res.json().catch(() => ({}));
				errorMsg = err.error || 'Fehler beim Aktualisieren der Sperre.';
			}
		} catch {
			errorMsg = 'Netzwerkfehler.';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<Modal {open} onclose={() => (open = false)} beschriftetDurch="sperre-titel">
	{#snippet header()}
		<h3
			id="sperre-titel"
			class="text-lg font-bold {profile?.is_manually_blocked
				? 'text-emerald-700'
				: 'text-error'} flex items-center gap-2"
		>
			{#if profile?.is_manually_blocked}
				<Unlock class="w-5 h-5" aria-hidden="true" />
				Sperre aufheben
			{:else}
				<Lock class="w-5 h-5" aria-hidden="true" />
				Ausleihe sperren
			{/if}
		</h3>
	{/snippet}

	<div class="px-6 py-6 text-on-surface-variant space-y-4">
		<p class="text-sm font-medium leading-relaxed">
			{#if profile?.is_manually_blocked}
				Möchten Sie die Ausleihe für <span class="font-bold text-on-surface"
					>{profile.vorname} {profile.nachname}</span
				> wirklich freigeben?
			{:else}
				Möchten Sie die Ausleihe für <span class="font-bold text-on-surface"
					>{profile?.vorname} {profile?.nachname}</span
				> wirklich sperren?
			{/if}
		</p>

		{#if willLock}
			<label class="block space-y-1.5">
				<span class="text-xs font-bold text-on-surface"
					>Grund der Sperre <span class="text-error">*</span></span
				>
				<textarea
					bind:value={reason}
					rows="2"
					disabled={isSubmitting}
					placeholder="z. B. wiederholt Bücher nicht zurückgegeben"
					class="w-full px-3 py-2 text-sm border border-outline-variant rounded-xl focus:outline-none focus:ring-2 focus:ring-error/30 disabled:opacity-50"
				></textarea>
			</label>
		{/if}

		{#if errorMsg}
			<div
				role="alert"
				class="p-3 bg-error-container text-on-error-container rounded-xl flex gap-2 items-start"
			>
				<AlertCircle class="w-4 h-4 mt-0.5 shrink-0" aria-hidden="true" />
				<p class="text-xs font-bold leading-tight">{errorMsg}</p>
			</div>
		{/if}
	</div>

	<div class="px-6 py-4 border-t border-outline-variant flex justify-end gap-3">
		<Button variant="secondary" onclick={() => (open = false)} disabled={isSubmitting}>
			Abbrechen
		</Button>
		<Button
			variant={profile?.is_manually_blocked ? 'success' : 'danger-solid'}
			onclick={handleConfirm}
			disabled={isSubmitting}
		>
			{#if isSubmitting}
				<Ladekreis size="sm" farbe="aktuell" />
				Wird verarbeitet...
			{:else}
				Bestätigen
			{/if}
		</Button>
	</div>
</Modal>

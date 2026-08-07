<script>
	import { apiFetch } from './apiFetch.js';
	import Button from './components/ui/Button.svelte';
	import { TriangleAlert } from '@lucide/svelte';

	let { open = false, profile, onclose, onsuccess } = $props();

	let deleteError = $state('');
	let isDeleting = $state(false);
	let confirmText = $state('');

	let expectedConfirmText = $derived(profile ? `${profile.vorname} ${profile.nachname}` : '');
	let isConfirmed = $derived(confirmText === expectedConfirmText);

	$effect(() => {
		if (open) {
			deleteError = '';
			isDeleting = false;
			confirmText = '';
		}
	});

	async function deleteStudent() {
		if (profile?.entliehene_buecher && profile.entliehene_buecher.length > 0) {
			deleteError = 'Löschen nicht möglich: Schüler hat noch entliehene Bücher';
			return;
		}
		deleteError = '';
		isDeleting = true;
		try {
			const res = await apiFetch(`/api/schueler/${profile.id}`, { method: 'DELETE' });
			if (res.ok) {
				onsuccess?.();
			} else {
				const errText = await res.text();
				try {
					const errObj = JSON.parse(errText);
					deleteError = errObj.error || 'Fehler beim Löschen des Schülers.';
				} catch {
					deleteError = errText || 'Fehler beim Löschen des Schülers.';
				}
			}
		} catch (err) {
			deleteError = 'Netzwerkfehler beim Löschen des Schülers.';
			console.error(err);
		} finally {
			isDeleting = false;
		}
	}

	function handleClose() {
		onclose?.();
	}
</script>

{#if open && profile}
	<div
		class="fixed inset-0 z-50 grid place-items-center bg-slate-900/40 backdrop-blur-xs p-4 animate-fade-in"
		role="dialog"
		aria-modal="true"
	>
		<div
			class="w-full max-w-md rounded-3xl border border-slate-200 bg-white p-6 shadow-2xl text-slate-800 text-left"
		>
			<h3 class="text-lg font-bold text-rose-600 flex items-center gap-2">
				<TriangleAlert class="h-6 w-6 text-rose-600" aria-hidden="true" />
				<span>Schüler löschen</span>
			</h3>
			{#if profile.entliehene_buecher && profile.entliehene_buecher.length > 0}
				<div
					class="mt-4 p-4 bg-rose-50 border border-rose-100 rounded-2xl text-sm font-semibold text-rose-700"
				>
					Löschen nicht möglich: Schüler hat noch entliehene Bücher
				</div>
				<div class="mt-6 flex justify-end">
					<Button variant="secondary" onclick={handleClose}>Schließen</Button>
				</div>
			{:else}
				<p class="mt-4 text-sm text-slate-600 leading-relaxed font-sans">
					Sind Sie sicher, dass Sie das Profil von <strong
						>{profile.vorname} {profile.nachname}</strong
					> löschen/archivieren möchten? Alle historischen Ausleihen werden anonymisiert. Dieser Vorgang
					kann in der regulären Oberfläche nicht rückgängig gemacht werden.
				</p>

				<div class="mt-5">
					<label class="block text-xs font-bold text-slate-700 mb-1.5" for="confirm-name">
						Bitte tippen Sie den Namen zur Bestätigung ein: <span
							class="font-mono text-rose-600 select-none bg-rose-50 px-1 py-0.5 rounded"
							>{expectedConfirmText}</span
						>
					</label>
					<input
						id="confirm-name"
						type="text"
						bind:value={confirmText}
						placeholder={expectedConfirmText}
						autocomplete="off"
						class="w-full border border-slate-300 rounded-xl px-4 py-3 text-sm focus:ring-2 focus:ring-rose-200 focus:border-rose-400 focus:outline-none transition-all"
					/>
				</div>

				{#if deleteError}
					<div
						class="mt-4 p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600"
					>
						{deleteError}
					</div>
				{/if}
				<div class="mt-6 flex flex-col-reverse sm:flex-row justify-end gap-3">
					<Button
						variant="secondary"
						size="lg"
						onclick={handleClose}
						disabled={isDeleting}
						class="w-full sm:w-auto">Abbrechen</Button
					>
					<Button
						variant="danger-solid"
						size="lg"
						onclick={deleteStudent}
						disabled={isDeleting || !isConfirmed}
						class="w-full sm:w-auto"
					>
						{#if isDeleting}Wird verarbeitet...{:else}Endgültig archivieren/löschen{/if}
					</Button>
				</div>
			{/if}
		</div>
	</div>
{/if}

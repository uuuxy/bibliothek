<script>
	import { apiClient } from './apiFetch.js';
	import { Unlock, Lock, X, AlertCircle } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';

	/** @type {{ open: boolean, profile: any, onsuccess: (updatedProfile: any) => void }} */
	let { open = $bindable(false), profile, onsuccess } = $props();

	let isSubmitting = $state(false);
	let errorMsg = $state('');

	async function handleConfirm() {
		if (!profile) return;
		isSubmitting = true;
		errorMsg = '';
		try {
			const res = await apiClient.patch(`/api/admin/students/${profile.id}/lock`, {
				is_locked: !profile.is_manually_blocked
			});

			if (res.ok) {
				const updated = await res.json();
				onsuccess(updated);
				open = false;
			} else {
				const err = await res.json().catch(() => ({}));
				errorMsg = err.error || 'Fehler beim Aktualisieren der Sperre.';
			}
		} catch (e) {
			errorMsg = 'Netzwerkfehler.';
		} finally {
			isSubmitting = false;
		}
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm animate-fade-in"
	>
		<div
			class="bg-white rounded-3xl shadow-2xl w-full max-w-md overflow-hidden transform transition-all border border-slate-100"
		>
			<div
				class="px-6 py-5 border-b border-slate-100 flex items-center justify-between bg-slate-50/50"
			>
				<h3
					class="text-lg font-bold {profile.is_manually_blocked
						? 'text-emerald-700'
						: 'text-rose-700'} flex items-center gap-2"
				>
					{#if profile.is_manually_blocked}
						<Unlock class="w-5 h-5" />
						Sperre aufheben
					{:else}
						<Lock class="w-5 h-5" />
						Ausleihe sperren
					{/if}
				</h3>
				<button
					onclick={() => (open = false)}
					disabled={isSubmitting}
					class="p-1 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 transition-colors disabled:opacity-50"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<div class="px-6 py-6 text-slate-600 space-y-4">
				<p class="text-sm font-medium leading-relaxed">
					{#if profile.is_manually_blocked}
						Möchten Sie die Ausleihe für <span class="font-bold text-slate-900"
							>{profile.vorname} {profile.nachname}</span
						> wirklich freigeben?
					{:else}
						Möchten Sie die Ausleihe für <span class="font-bold text-slate-900"
							>{profile.vorname} {profile.nachname}</span
						> wirklich sperren?
					{/if}
				</p>

				{#if errorMsg}
					<div
						class="p-3 bg-rose-50 border border-rose-100 rounded-xl flex gap-2 items-start text-rose-700 animate-fade-in"
					>
						<AlertCircle class="w-4 h-4 mt-0.5 shrink-0" />
						<p class="text-xs font-bold leading-tight">{errorMsg}</p>
					</div>
				{/if}
			</div>

			<div class="px-6 py-4 bg-slate-50 border-t border-slate-100 flex justify-end gap-3">
				<Button variant="secondary" onclick={() => (open = false)} disabled={isSubmitting}>
					Abbrechen
				</Button>
				<Button
					variant={profile.is_manually_blocked ? 'success' : 'danger-solid'}
					onclick={handleConfirm}
					disabled={isSubmitting}
				>
					{#if isSubmitting}
						<div
							class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
						></div>
						Wird verarbeitet...
					{:else}
						Bestätigen
					{/if}
				</Button>
			</div>
		</div>
	</div>
{/if}

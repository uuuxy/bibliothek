<script>
	import { AlertTriangle } from '@lucide/svelte';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';

	/**
	 * @typedef {Object} Props
	 * @property {boolean} open
	 * @property {() => void} onclose
	 * @property {any} userToDelete
	 * @property {boolean} deletingUser
	 * @property {string | null} error
	 * @property {() => void} confirmDeleteUser
	 */
	/** @type {Props} */
	let { open, onclose, userToDelete, deletingUser, error, confirmDeleteUser } = $props();
</script>

<Modal {open} {onclose} size="sm">
	<div class="p-6 space-y-4">
		{#if error}
			<!-- Fachlicher Konflikt (z. B. 409: offene Handapparat-Ausleihen). Bleibt im Modal
				     stehen, damit die handlungsleitende Meldung im Kontext sichtbar ist. -->
			<div
				class="p-3.5 rounded-xl bg-rose-50 border border-rose-100 text-rose-600 text-xs font-semibold leading-relaxed animate-slide-up"
			>
				<AlertTriangle class="h-4 w-4" aria-hidden="true" />
				{error}
			</div>
		{/if}
		<div
			class="w-12 h-12 rounded-full bg-rose-50 border border-rose-100 text-rose-600 flex items-center justify-center text-xl mx-auto"
		>
			<AlertTriangle class="h-4 w-4" aria-hidden="true" />
		</div>
		<div class="text-center space-y-1.5">
			<h3 class="font-bold text-slate-800 text-base">Benutzer unwiderruflich löschen?</h3>
			<p class="text-xs text-slate-500 leading-relaxed font-medium">
				Sind Sie sicher, dass Sie den Benutzer <strong
					>{userToDelete?.vorname} {userToDelete?.nachname}</strong
				> löschen möchten? Diese Aktion wird im Logbuch vermerkt.
			</p>
		</div>
		<div class="flex items-center justify-center gap-3 pt-3 border-t border-slate-100">
			<Button variant="secondary" onclick={onclose} disabled={deletingUser}>Abbrechen</Button>
			<Button variant="danger-solid" onclick={confirmDeleteUser} disabled={deletingUser}>
				{#if deletingUser}<div
						class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"
					></div>{/if}
				Löschen
			</Button>
		</div>
	</div>
</Modal>

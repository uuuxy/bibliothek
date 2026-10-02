<script>
	import { AlertTriangle } from '@lucide/svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
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

<Modal {open} {onclose} size="sm" beschriftetDurch="benutzer-loeschen-titel">
	<div class="p-6 space-y-4">
		{#if error}
			<!-- Etwa offene Ausleihen im Handapparat: Der Satz des Servers sagt, was zu tun ist. -->
			<div
				role="alert"
				class="flex animate-slide-up items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
			>
				<AlertTriangle class="h-4 w-4 shrink-0" aria-hidden="true" />
				<span>{error}</span>
			</div>
		{/if}
		<div
			class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-error-container text-on-error-container"
		>
			<AlertTriangle class="h-4 w-4" aria-hidden="true" />
		</div>
		<div class="text-center space-y-1.5">
			<h3 id="benutzer-loeschen-titel" class="text-base font-bold text-on-surface">
				Benutzer unwiderruflich löschen?
			</h3>
			<p class="text-xs leading-relaxed font-medium text-on-surface-variant">
				Sind Sie sicher, dass Sie den Benutzer <strong
					>{userToDelete?.vorname} {userToDelete?.nachname}</strong
				> löschen möchten? Diese Aktion wird im Logbuch vermerkt.
			</p>
		</div>
		<div class="flex items-center justify-center gap-3 border-t border-outline-variant pt-3">
			<Button variant="secondary" onclick={onclose} disabled={deletingUser}>Abbrechen</Button>
			<Button variant="danger-solid" onclick={confirmDeleteUser} disabled={deletingUser}>
				{#if deletingUser}<Ladekreis size="sm" farbe="aktuell" />{/if}
				Löschen
			</Button>
		</div>
	</div>
</Modal>

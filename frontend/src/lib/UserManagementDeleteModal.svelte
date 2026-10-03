<!-- @component UserManagementDeleteModal — Rückfrage vor dem Löschen eines Kontos.

     Aufbau wie die Rückfrage des Hauses (ui/BestaetigungsDialog): Überschrift, Text, Aktionen
     rechts, die Aktion am Rand. Ein eigener Dialog bleibt es, weil er offen bleibt, solange
     gelöscht wird, und den Grund eines gescheiterten Löschens selbst zeigt. -->
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
	<div class="space-y-4 p-6">
		<h2 id="benutzer-loeschen-titel" class="text-lg font-bold text-on-surface">
			Benutzer unwiderruflich löschen?
		</h2>
		<p class="text-sm leading-relaxed text-on-surface-variant">
			Sind Sie sicher, dass Sie den Benutzer <strong
				>{userToDelete?.vorname} {userToDelete?.nachname}</strong
			> löschen möchten? Diese Aktion wird im Logbuch vermerkt.
		</p>
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
		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={onclose} disabled={deletingUser}>Abbrechen</Button>
			<Button variant="danger-solid" onclick={confirmDeleteUser} disabled={deletingUser}>
				{#if deletingUser}<Ladekreis size="sm" farbe="aktuell" />{/if}
				Löschen
			</Button>
		</div>
	</div>
</Modal>

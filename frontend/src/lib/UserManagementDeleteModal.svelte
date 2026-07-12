<script>
	import Modal from './Modal.svelte';

	/**
	 * @typedef {Object} Props
	 * @property {boolean} open
	 * @property {() => void} onclose
	 * @property {any} userToDelete
	 * @property {boolean} deletingUser
	 * @property {() => void} confirmDeleteUser
	 */
	/** @type {Props} */
	let { open, onclose, userToDelete, deletingUser, confirmDeleteUser } = $props();
</script>

<Modal {open} {onclose} size="sm">
		<div class="p-6 space-y-4">
			<div
				class="w-12 h-12 rounded-full bg-rose-50 border border-rose-150 text-rose-600 flex items-center justify-center text-xl mx-auto"
			>
				⚠️
			</div>
			<div class="text-center space-y-1.5">
				<h3 class="font-bold text-slate-800 text-sm">Benutzer unwiderruflich löschen?</h3>
				<p class="text-xs text-slate-450 leading-relaxed font-medium">
					Sind Sie sicher, dass Sie den Benutzer <strong
						>{userToDelete?.vorname} {userToDelete?.nachname}</strong
					> löschen möchten? Diese Aktion wird im Logbuch vermerkt.
				</p>
			</div>
			<div class="flex items-center justify-center gap-3 pt-3 border-t border-slate-100">
				<button
					onclick={onclose}
					disabled={deletingUser}
					class="px-4 py-2 text-xs font-semibold border border-slate-200 rounded-xl hover:bg-slate-50 cursor-pointer"
					>Abbrechen</button
				>
				<button
					onclick={confirmDeleteUser}
					disabled={deletingUser}
					class="px-4 py-2 text-xs font-bold text-white bg-rose-600 hover:bg-rose-700 rounded-xl shadow-xs transition-colors flex items-center justify-center cursor-pointer"
				>
					{#if deletingUser}<div
							class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin mr-1"
						></div>{/if}
					Löschen
				</button>
			</div>
		</div>
</Modal>

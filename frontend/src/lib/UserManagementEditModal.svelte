<script>
	import Modal from './Modal.svelte';

	/**
	 * @typedef {Object} Props
	 * @property {boolean} open
	 * @property {() => void} onclose
	 * @property {boolean} isEditingUser
	 * @property {any} userForm
	 * @property {boolean} submittingUser
	 * @property {string | null} error
	 * @property {(e: SubmitEvent) => void} handleSaveUser
	 */
	/** @type {Props} */
	let {
		open,
		onclose,
		isEditingUser,
		userForm = $bindable(),
		submittingUser,
		error,
		handleSaveUser
	} = $props();
</script>

<Modal {open} {onclose} size="md">
	{#snippet header()}
		<h3 class="font-bold text-slate-800 text-sm">
			{isEditingUser ? 'Benutzer bearbeiten' : 'Neuen Benutzer anlegen'}
		</h3>
	{/snippet}
		<form onsubmit={handleSaveUser} class="p-6 space-y-4">
			{#if error}
				<div
					class="p-3.5 rounded-xl bg-rose-50 border border-rose-100 text-rose-600 text-xs font-semibold leading-relaxed animate-slide-up"
				>
					⚠️ {error}
				</div>
			{/if}
			<div class="grid grid-cols-2 gap-4">
				{@render inputField(
					'vorname',
					'Vorname',
					'text',
					userForm.vorname,
					(/** @type {any} */ v) => (userForm.vorname = v),
					true
				)}
				{@render inputField(
					'nachname',
					'Nachname',
					'text',
					userForm.nachname,
					(/** @type {any} */ v) => (userForm.nachname = v),
					true
				)}
			</div>
			{@render inputField(
				'email',
				'E-Mail Adresse',
				'email',
				userForm.email,
				(/** @type {any} */ v) => (userForm.email = v),
				true
			)}
			{@render inputField(
				'barcode_id',
				'Barcode (Anmelde-ID)',
				'text',
				userForm.barcode_id,
				(/** @type {any} */ v) => (userForm.barcode_id = v),
				false,
				'Z. B. L-001, MA-04 (optional)'
			)}
			<div class="space-y-1.5">
				<label for="rolle" class="block text-xs font-bold text-slate-400 uppercase tracking-wider"
					>Benutzer-Rolle</label
				>
				<select
					id="rolle"
					bind:value={userForm.rolle}
					class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
				>
					<option value="mitarbeiter">Mitarbeiter</option>
					<option value="lehrer">Lehrer</option>
					<option value="admin">Administrator</option>
				</select>
			</div>
			{#if isEditingUser}
				<div class="flex items-center gap-3 py-1.5">
					<label class="relative inline-flex items-center cursor-pointer">
						<input type="checkbox" bind:checked={userForm.aktiv} class="sr-only peer" />
						<div
							class="w-10 h-6 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-350 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"
						></div>
					</label>
					<span class="text-xs font-bold text-slate-650">Benutzerkonto ist aktiv</span>
				</div>
			{/if}
			<div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-100">
				<button
					type="button"
					onclick={onclose}
					class="px-4 py-2 text-xs font-semibold border border-slate-200 rounded-xl hover:bg-slate-50 cursor-pointer"
					>Abbrechen</button
				>
				<button
					type="submit"
					disabled={submittingUser}
					class="px-4 py-2 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-xs transition-colors flex items-center justify-center cursor-pointer disabled:opacity-60"
				>
					{#if submittingUser}<div
							class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin mr-1"
						></div>{/if}
					Speichern
				</button>
			</div>
		</form>
</Modal>

{#snippet inputField(id, label, type, value, onInput, required, placeholder)}
	<div class="space-y-1.5">
		<label for={id} class="block text-xs font-bold text-slate-400 uppercase tracking-wider"
			>{label}</label
		>
		<input
			{id}
			{type}
			{value}
			oninput={(e) => onInput(e.currentTarget.value)}
			{required}
			placeholder={placeholder ?? ''}
			class="w-full bg-slate-50 border border-slate-200 rounded-xl py-2.5 px-3 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/10 focus:border-blue-300 transition-all font-medium text-slate-800"
		/>
	</div>
{/snippet}

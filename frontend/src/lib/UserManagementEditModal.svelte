<script>
	import { AlertTriangle } from '@lucide/svelte';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Switch from './components/ui/Switch.svelte';
	import Select from './components/ui/Select.svelte';

	const ROLLEN = [
		{ value: 'helfer', label: 'Helfer' },
		{ value: 'mitarbeiter', label: 'Mitarbeiter' },
		{ value: 'lehrer', label: 'Lehrer' },
		{ value: 'admin', label: 'Administrator' }
	];

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
				<AlertTriangle class="h-4 w-4" aria-hidden="true" />
				{error}
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
			<label for="rolle" class="block text-xs font-medium text-slate-400">Benutzer-Rolle</label>
			<Select id="rolle" bind:value={userForm.rolle} options={ROLLEN} />
		</div>
		{#if isEditingUser}
			<!-- Vorher ein peer-checked-Nachbau OHNE zugänglichen Namen: Der Screenreader las
			     „Kontrollkästchen", die danebenstehende Erklärung gehörte niemandem. -->
			<div class="flex items-center gap-3 py-1.5">
				<Switch id="benutzer-aktiv" bind:checked={userForm.aktiv} label="Benutzerkonto ist aktiv" />
				<label for="benutzer-aktiv" class="cursor-pointer text-xs font-bold text-slate-600">
					Benutzerkonto ist aktiv
				</label>
			</div>
		{/if}
		<div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-100">
			<Button variant="secondary" type="button" onclick={onclose}>Abbrechen</Button>
			<Button type="submit" disabled={submittingUser}>
				{#if submittingUser}<div
						class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"
					></div>{/if}
				Speichern
			</Button>
		</div>
	</form>
</Modal>

{#snippet inputField(id, label, type, value, onInput, required, placeholder)}
	<div class="space-y-1.5">
		<label for={id} class="block text-xs font-medium text-slate-400">{label}</label>
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

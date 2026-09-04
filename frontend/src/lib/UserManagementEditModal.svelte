<script>
	import { AlertTriangle } from '@lucide/svelte';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Switch from './components/ui/Switch.svelte';
	import Select from './components/ui/Select.svelte';
	import Feld from './components/ui/Feld.svelte';

	const ROLLEN = [
		{ value: 'helfer', label: 'Helfer' },
		{ value: 'mitarbeiter', label: 'Mitarbeiter' },
		{ value: 'kollegium', label: 'Kollegium (nur Portal)' },
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
		<h3 class="font-bold text-slate-800 text-base">
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
			<Feld id="vorname" label="Vorname" bind:value={userForm.vorname} required />
			<Feld id="nachname" label="Nachname" bind:value={userForm.nachname} required />
		</div>
		<Feld id="email" label="E-Mail Adresse" type="email" bind:value={userForm.email} required />
		<Feld
			id="barcode_id"
			label="Barcode (Anmelde-ID)"
			bind:value={userForm.barcode_id}
			placeholder="Z. B. L-001, MA-04 (optional)"
		/>
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

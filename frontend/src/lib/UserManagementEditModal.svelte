<script>
	import { TriangleAlert } from '@lucide/svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Switch from './components/ui/Switch.svelte';
	import Select from './components/ui/Select.svelte';
	import Feld from './components/ui/Feld.svelte';
	import { ROLLEN_AUSWAHL } from './benutzerRollen.js';

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

	// Die Rolle sagt, was jemand darf — mehr entscheidet dieses Formular nicht. Das Feld
	// „Personenart" ist mit Migration 125 weggefallen: Wer jemand ist, steht an seiner
	// Leserzeile und gehört in die Leserdatei.
	const AUSWAHLEN = [{ id: 'rolle', label: 'Benutzer-Rolle', options: ROLLEN_AUSWAHL }];
</script>

<Modal {open} {onclose} size="md" beschriftetDurch="benutzer-formular-titel">
	{#snippet header()}
		<h3 id="benutzer-formular-titel" class="text-base font-bold text-on-surface">
			{isEditingUser ? 'Benutzer bearbeiten' : 'Neuen Benutzer anlegen'}
		</h3>
	{/snippet}
	<form onsubmit={handleSaveUser} class="p-6 space-y-4">
		{#if error}
			<div
				role="alert"
				class="flex animate-slide-up items-center gap-2 rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
			>
				<TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
				<span>{error}</span>
			</div>
		{/if}
		<div class="grid grid-cols-2 gap-4">
			<Feld id="vorname" label="Vorname" bind:value={userForm.vorname} required />
			<Feld id="nachname" label="Nachname" bind:value={userForm.nachname} required />
		</div>
		<Feld id="email" label="E-Mail Adresse" type="email" bind:value={userForm.email} required />
		<Feld
			id="barcode_id"
			label="Ausweisnummer"
			bind:value={userForm.barcode_id}
			hint="Leeres Feld: Das Programm vergibt die nächste freie Nummer."
		/>
		{#each AUSWAHLEN as auswahl (auswahl.id)}
			<div class="space-y-1.5">
				<label for={auswahl.id} class="block text-xs font-medium text-on-surface-variant"
					>{auswahl.label}</label
				>
				<Select id={auswahl.id} bind:value={userForm[auswahl.id]} options={auswahl.options} />
			</div>
		{/each}
		{#if isEditingUser}
			<div class="flex items-center gap-3 py-1.5">
				<Switch id="benutzer-aktiv" bind:checked={userForm.aktiv} label="Benutzerkonto ist aktiv" />
				<label
					for="benutzer-aktiv"
					class="cursor-pointer text-xs font-bold text-on-surface-variant"
				>
					Benutzerkonto ist aktiv
				</label>
			</div>
		{/if}
		<div class="flex items-center justify-end gap-3 border-t border-outline-variant pt-3">
			<Button variant="secondary" type="button" onclick={onclose}>Abbrechen</Button>
			<Button type="submit" disabled={submittingUser}>
				{#if submittingUser}<Ladekreis size="sm" farbe="aktuell" />{/if}
				Speichern
			</Button>
		</div>
	</form>
</Modal>

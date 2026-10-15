<!-- @component Start einer Inventur: welcher Teil des Bestands geprüft wird.

     Bauform wie BestellMittelDialog: Modal, Radio-Gruppe, die Felder zur gewählten Art, die
     Aktion erst mit vollständiger Auswahl (M3 Dialogs: „Disable confirming actions until a
     choice is made. Dismissive actions are never disabled."). Eine Meldung des Servers steht
     im Dialog (M3 Dialogs: „Errors about the dialog fields should always appear inline"). -->
<script>
	import { slide } from 'svelte/transition';
	import Modal from '../Modal.svelte';
	import Button from './ui/Button.svelte';
	import Select from './ui/Select.svelte';
	import Feld from './ui/Feld.svelte';
	import Radio from './ui/Radio.svelte';

	const KLASSEN = [
		{ value: '', label: 'Alle Klassen' },
		...[5, 6, 7, 8, 9, 10, 11, 12, 13].map((g) => ({ value: g, label: `Klasse ${g}` }))
	];

	const BEREICHE = [
		{ wert: 'global', name: 'Komplette Bibliothek', text: 'Prüfe den gesamten Bestand ab.' },
		{
			wert: 'signature',
			name: 'Nur bestimmte Signatur',
			text: 'Grenze die Inventur auf eine Kategorie ein.'
		},
		{
			wert: 'filter',
			name: 'Nach Fach / Klasse',
			text: 'Gezielte Teil-Inventur, z. B. „Mathe, Klasse 5“.'
		}
	];

	/**
	 * @type {{
	 *   open: boolean,
	 *   state: any,
	 *   onClose: () => void,
	 *   onStart: () => void
	 * }}
	 */
	let { open, state, onClose, onStart } = $props();

	const bereit = $derived(
		state.scopeType === 'global' ||
			(state.scopeType === 'signature' && String(state.selectedSignatur).trim() !== '') ||
			(state.scopeType === 'filter' && (state.selectedFach !== '' || state.selectedGrade !== ''))
	);
</script>

<Modal {open} onclose={onClose} size="md" beschriftetDurch="inventur-start-titel">
	<div class="space-y-4 p-6">
		<h2 id="inventur-start-titel" class="text-lg font-bold text-on-surface">
			Bereich der Inventur wählen
		</h2>
		<p class="text-sm leading-relaxed text-on-surface-variant">
			Welcher Teil der Bibliothek soll geprüft werden?
		</p>

		<fieldset class="space-y-3">
			<legend class="sr-only">Bereich der Inventur</legend>
			{#each BEREICHE as b (b.wert)}
				<label class="flex cursor-pointer items-start gap-3 text-sm text-on-surface">
					<Radio bind:group={state.scopeType} value={b.wert} aria-label={b.name} />
					<span>
						<span class="block font-medium">{b.name}</span>
						<span class="block text-xs text-on-surface-variant">{b.text}</span>
					</span>
				</label>
			{/each}
		</fieldset>

		{#if state.scopeType === 'signature'}
			<div transition:slide>
				<!-- Freie Eingabe mit Vorschlagsliste: Die Signatur wird als Präfix gelesen,
				     „LMF Deu 7" erfasst also auch „LMF Deu 7 / Bie". Eine reine Auswahlliste
				     könnte nur vorhandene Werte anbieten und damit kein ganzes Regal. -->
				<Feld
					id="inv-signatur"
					label="Signatur auswählen"
					list="inv-signatur-vorschlaege"
					bind:value={state.selectedSignatur}
					placeholder="z. B. LMF Deu 7"
					hint="Erfasst alles, was mit dieser Signatur beginnt — „LMF Deu 7“ also auch „LMF Deu 7 / Bie“."
				/>
				<datalist id="inv-signatur-vorschlaege">
					{#each state.signaturen as sig (sig.signatur)}
						<option value={sig.signatur}>{sig.signatur} — {sig.exemplare} Exemplare</option>
					{/each}
				</datalist>
			</div>
		{/if}

		{#if state.scopeType === 'filter'}
			<div transition:slide class="grid grid-cols-2 gap-3">
				<div class="space-y-1.5">
					<label for="inventur-fach" class="block text-xs font-medium text-on-surface-variant"
						>Fach</label
					>
					<Select
						id="inventur-fach"
						bind:value={state.selectedFach}
						options={[
							{ value: '', label: 'Alle Fächer' },
							...state.faecher.map((/** @type {string} */ f) => ({ value: f, label: f }))
						]}
					/>
				</div>
				<div class="space-y-1.5">
					<label for="inventur-klasse" class="block text-xs font-medium text-on-surface-variant"
						>Klasse</label
					>
					<Select id="inventur-klasse" bind:value={state.selectedGrade} options={KLASSEN} />
				</div>
			</div>
		{/if}

		{#if state.errorMessage}
			<p class="text-sm text-error" role="alert">{state.errorMessage}</p>
		{/if}

		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={onClose}>Abbrechen</Button>
			<Button onclick={onStart} disabled={!bereit}>Inventur Starten</Button>
		</div>
	</div>
</Modal>

<!-- @component Start einer Inventur: welcher Teil des Bestands geprüft wird.

     Bauform wie BestellMittelDialog: Modal, Radio-Gruppe, die Felder zur gewählten Art, die
     Aktion erst mit einer Auswahl (M3 Dialogs: „Disable confirming actions until a choice is
     made. Dismissive actions are never disabled."). Eine fehlende Signatur ist ein leeres Feld:
     Der Klick nennt sie dort (ARCHITEKTUR, „Felder und Eingabe“). Eine Meldung des Servers steht
     im Dialog (M3 Dialogs: „Errors about the dialog fields should always appear inline"). -->
<script>
	import { tick } from 'svelte';
	import { slide } from 'svelte/transition';
	import { standardAccelerate, standardDecelerate } from '../../utils/bewegung.js';
	import Dialog from '../ui/Dialog.svelte';
	import Button from '../ui/Button.svelte';
	import Select from '../ui/Select.svelte';
	import VorschlagFeld from '../ui/VorschlagFeld.svelte';
	import SignaturCoverVorschau from '../signaturen/SignaturCoverVorschau.svelte';
	import Radio from '../ui/Radio.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Pflichthinweis from '../ui/Pflichthinweis.svelte';

	const KLASSEN = [
		{ value: '', label: 'Alle Klassen' },
		...[5, 6, 7, 8, 9, 10, 11, 12, 13].map((g) => ({ value: g, label: `Klasse ${g}` }))
	];

	const BEREICHE = [
		{ wert: 'global', name: 'Komplette Bibliothek', text: 'Prüfe den gesamten Bestand ab' },
		{
			wert: 'signature',
			name: 'Nur bestimmte Signatur',
			text: 'Grenze die Inventur auf eine Kategorie ein'
		},
		{
			wert: 'filter',
			name: 'Nach Fach oder Klasse',
			text: 'Gezielte Teil-Inventur, z. B. „Mathe, Klasse 5“'
		}
	];

	/**
	 * @type {{
	 *   open: boolean,
	 *   ablauf: any,
	 *   onClose: () => void,
	 *   onStart: () => Promise<void>
	 * }}
	 */
	let { open, ablauf, onClose, onStart } = $props();

	// Solange der Start am Server ist, bleibt „Starten" gesperrt. Ein zweiter Klick
	// schickte einen zweiten Start, den der Server ablehnt; nach dem Abschluss stand dann „Für
	// diesen Bereich läuft bereits eine Inventur", obwohl keine lief.
	let startet = $state(false);
	async function starte() {
		if (startet || ohneAuswahl) return;
		verlangt = true;
		if (signaturFehlt) {
			await tick();
			document.getElementById('inv-signatur')?.focus();
			return;
		}
		startet = true;
		try {
			await onStart();
		} finally {
			startet = false;
		}
	}

	// Fach und Klasse sind eine Auswahl: Ohne eins von beiden ist beim Filter keine Wahl
	// getroffen, „Starten" bleibt gesperrt.
	const ohneAuswahl = $derived(
		ablauf.bereichArt === 'filter' &&
			ablauf.gewaehltesFach === '' &&
			ablauf.gewaehlterJahrgang === ''
	);
	const signaturFehlt = $derived(
		ablauf.bereichArt === 'signature' && String(ablauf.gewaehlteSignatur).trim() === ''
	);
	let verlangt = $state(false);
	$effect(() => {
		if (open) verlangt = false;
	});
</script>

<Dialog {open} onclose={onClose} size="md" beschriftetDurch="inventur-start-titel">
	<div class="space-y-4 p-6">
		<h2 id="inventur-start-titel" class="dialog-titel">Bereich der Inventur wählen</h2>
		<p class="text-sm leading-relaxed text-on-surface-variant">
			Welcher Teil der Bibliothek soll geprüft werden?
		</p>

		<fieldset class="space-y-3">
			<legend class="sr-only">Bereich der Inventur</legend>
			{#each BEREICHE as b (b.wert)}
				<label class="flex cursor-pointer items-start gap-3 text-sm text-on-surface">
					<Radio bind:group={ablauf.bereichArt} value={b.wert} aria-label={b.name} />
					<span>
						<span class="block font-medium">{b.name}</span>
						<span class="block text-xs text-on-surface-variant">{b.text}</span>
					</span>
				</label>
			{/each}
		</fieldset>

		{#if ablauf.bereichArt === 'signature'}
			<div
				in:slide={{ duration: 250, easing: standardDecelerate }}
				out:slide={{ duration: 200, easing: standardAccelerate }}
			>
				<!-- Freie Eingabe mit Vorschlagsliste: Die Signatur wird als Präfix gelesen,
				     „LMF Deu 7" erfasst also auch „LMF Deu 7 / Bie". Eine reine Auswahlliste
				     könnte nur vorhandene Werte anbieten und damit kein ganzes Regal. -->
				<Pflichthinweis />
				<VorschlagFeld
					id="inv-signatur"
					etikett="Signatur auswählen"
					aria-required="true"
					ungueltig={verlangt && signaturFehlt}
					bind:value={ablauf.gewaehlteSignatur}
					vorschlaege={ablauf.signaturen.map((s) => ({
						wert: s.signatur,
						zusatz: `${s.exemplare} ${s.exemplare === 1 ? 'Exemplar' : 'Exemplare'}`
					}))}
					placeholder="z. B. LMF Deu 7"
					hint={verlangt && signaturFehlt
						? 'Bitte die Signatur eintragen'
						: 'Erfasst alles, was mit dieser Signatur beginnt, „LMF Deu 7“ also auch „LMF Deu 7 / Bie“'}
				>
					{#snippet vorschau(/** @type {string} */ wert)}
						<SignaturCoverVorschau signatur={wert} />
					{/snippet}
				</VorschlagFeld>
			</div>
		{/if}

		{#if ablauf.bereichArt === 'filter'}
			<div
				in:slide={{ duration: 250, easing: standardDecelerate }}
				out:slide={{ duration: 200, easing: standardAccelerate }}
				class="grid grid-cols-2 gap-3"
			>
				<div class="space-y-1.5">
					<label for="inventur-fach" class="block text-sm font-medium text-on-surface-variant"
						>Fach</label
					>
					<Select
						id="inventur-fach"
						bind:value={ablauf.gewaehltesFach}
						options={[
							{ value: '', label: 'Alle Fächer' },
							...ablauf.faecher.map((/** @type {string} */ f) => ({ value: f, label: f }))
						]}
					/>
				</div>
				<div class="space-y-1.5">
					<label for="inventur-klasse" class="block text-sm font-medium text-on-surface-variant"
						>Klasse</label
					>
					<Select id="inventur-klasse" bind:value={ablauf.gewaehlterJahrgang} options={KLASSEN} />
				</div>
			</div>
		{/if}

		{#if ablauf.fehler}
			<p class="text-sm text-error" role="alert">{ablauf.fehler}</p>
		{/if}

		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={onClose}>Abbrechen</Button>
			<Button onclick={starte} disabled={ohneAuswahl || startet}>
				{#if startet}
					<Ladekreis size="sm" farbe="aktuell" />
				{/if}
				Starten
			</Button>
		</div>
	</div>
</Dialog>

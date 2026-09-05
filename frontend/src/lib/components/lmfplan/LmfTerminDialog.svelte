<!-- @component LmfTerminDialog — eine Zeile des Plans anlegen oder ändern: Datum, Stunde,
     Art, 0..n Klassen aus dem Vokabular, Vermerk. Eine Zeile ohne Klasse („Bücher
     setzen") braucht einen Vermerk — sonst stünde eine leere Zeile im Plan. -->
<script>
	import { untrack } from 'svelte';
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import Segmente from '../ui/Segmente.svelte';
	import { ARTEN, STUNDEN } from '../../lmfplanDienst.js';

	/** @type {{ open: boolean, termin: any, klassen: string[], onclose: () => void, onspeichern: (t: any) => void }} */
	let { open, termin, klassen, onclose, onspeichern } = $props();

	let datum = $state('');
	let stunde = $state(1);
	let art = $state('rueckgabe');
	let gewaehlt = $state(/** @type {string[]} */ ([]));
	let weitereKlasse = $state('');
	let vermerk = $state('');

	// Beim Öffnen aus dem übergebenen Termin füllen (Bearbeiten) oder leeren (Neu).
	$effect(() => {
		if (!open) return;
		untrack(() => {
			datum = termin?.datum ?? '';
			stunde = termin?.stunde ?? 1;
			art = termin?.art ?? 'rueckgabe';
			gewaehlt = [...(termin?.klassen ?? [])];
			weitereKlasse = '';
			vermerk = termin?.vermerk ?? '';
		});
	});

	// Klassen des Vokabulars plus die, die der Termin schon trägt (auch wenn sie gerade
	// keine Schüler haben — die Ausgabe im August plant für Klassen, die es erst nach dem
	// LUSD-Import gibt).
	const auswahl = $derived(
		[...new Set([...klassen, ...gewaehlt])].sort((a, b) =>
			a.localeCompare(b, 'de', { numeric: true })
		)
	);

	/** @param {string} k */
	function toggle(k) {
		gewaehlt = gewaehlt.includes(k) ? gewaehlt.filter((x) => x !== k) : [...gewaehlt, k];
	}

	function weitereHinzufuegen() {
		const k = weitereKlasse.trim();
		if (k && !gewaehlt.includes(k)) gewaehlt = [...gewaehlt, k];
		weitereKlasse = '';
	}

	const gueltig = $derived(Boolean(datum) && (gewaehlt.length > 0 || vermerk.trim() !== ''));

	function speichern() {
		if (!gueltig) return;
		onspeichern({
			id: termin?.id,
			datum,
			stunde: Number(stunde),
			art,
			klassen: gewaehlt,
			vermerk: vermerk.trim()
		});
	}
</script>

<Modal {open} {onclose} size="lg">
	{#snippet header()}
		<h2 class="text-title-large font-normal text-on-surface">
			{termin?.id ? 'Termin bearbeiten' : 'Termin hinzufügen'}
		</h2>
	{/snippet}

	<div class="space-y-4">
		<Segmente
			etikett="Art des Termins"
			optionen={ARTEN.map((a) => ({ wert: a.wert, text: a.label }))}
			wert={art}
			onwahl={(/** @type {string} */ w) => (art = w)}
		/>

		<div class="grid grid-cols-2 gap-4">
			<Feld id="lmf-termin-datum" label="Datum" type="date" bind:value={datum} />
			<div>
				<label
					class="block text-xs font-medium text-on-surface-variant mb-1"
					for="lmf-termin-stunde">Stunde</label
				>
				<Select
					id="lmf-termin-stunde"
					bind:value={stunde}
					options={STUNDEN.map((s) => ({ value: s, label: `${s}. Stunde` }))}
				/>
			</div>
		</div>

		<fieldset>
			<legend class="text-xs font-medium text-on-surface-variant mb-1"
				>Klassen (mehrere möglich, keine bei „Bücher setzen")</legend
			>
			<div
				class="max-h-48 overflow-y-auto rounded-lg border border-outline-variant p-2 grid grid-cols-3 gap-x-3 gap-y-1"
			>
				{#each auswahl as k (k)}
					<label class="flex items-center gap-2 text-sm text-on-surface cursor-pointer">
						<input
							type="checkbox"
							checked={gewaehlt.includes(k)}
							onchange={() => toggle(k)}
							class="accent-primary"
						/>
						{k}
					</label>
				{/each}
				{#if auswahl.length === 0}
					<p class="col-span-3 text-sm text-on-surface-variant">Noch keine Klassen bekannt.</p>
				{/if}
			</div>
			<div class="mt-2 flex items-end gap-2">
				<Feld
					id="lmf-termin-weitere-klasse"
					label="Weitere Klasse"
					bind:value={weitereKlasse}
					placeholder="z. B. 05G6"
					class="flex-1"
					onkeydown={(/** @type {KeyboardEvent} */ e) => {
						if (e.key === 'Enter') {
							e.preventDefault();
							weitereHinzufuegen();
						}
					}}
				/>
				<Button variant="secondary" onclick={weitereHinzufuegen} disabled={!weitereKlasse.trim()}
					>Hinzufügen</Button
				>
			</div>
		</fieldset>

		<Feld
			id="lmf-termin-vermerk"
			label="Besonderheiten"
			bind:value={vermerk}
			placeholder="z. B. Bücher setzen, Nachzügler, erst zur 2. Hälfte"
		/>

		<div class="flex justify-end gap-2 pt-2">
			<Button variant="secondary" onclick={onclose}>Abbrechen</Button>
			<Button onclick={speichern} disabled={!gueltig}>Speichern</Button>
		</div>
	</div>
</Modal>

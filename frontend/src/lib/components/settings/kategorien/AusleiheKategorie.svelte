<script>
	/**
	 * @component AusleiheKategorie
	 * Wie lange ausgeliehen wird und wie viel gleichzeitig — dazu der Ferien-Leseclub,
	 * der genau diese Fristen für die Ferien pauschal ersetzt. Er steht deshalb hier
	 * und nicht in einer eigenen Kategorie: Er ist eine Ausnahme von den Fristen
	 * darüber, keine Sache für sich.
	 */
	import SettingField from '../SettingField.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import Switch from '../../ui/Switch.svelte';
	import { untrack } from 'svelte';
	import { speichereKategorie } from '../../../einstellungenSpeichern.js';

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	// Der Anfangswert ist eine MOMENTAUFNAHME: Das Formular gehört ab hier dem
	// Benutzer, nicht dem Server. Frische Werte kommen nach dem Speichern über den
	// {#key}-Block in SystemSettings.svelte, der die Kategorie neu aufbaut — ohne
	// untrack würde Svelte hier eine Ableitung erwarten und beim Neuladen die
	// halb getippte Eingabe überschreiben.
	const start = untrack(() => daten);

	let fristBuch = $state(start.frist_buch_tage ?? 21);
	let fristMedien = $state(start.frist_medien_tage ?? 7);
	let maxAusleihen = $state(start.max_ausleihen_schueler ?? 5);
	let lmfStichtag = $state(start.lmf_stichtag ?? '07-31');
	let leseclubAktiv = $state(start.ferien_leseclub_aktiv ?? false);
	let leseclubZieldatum = $state(start.ferien_leseclub_zieldatum ?? '');

	const speichern = () =>
		speichereKategorie({
			felder: {
				lmf_stichtag: lmfStichtag,
				ferien_leseclub_aktiv: leseclubAktiv,
				ferien_leseclub_zieldatum: leseclubZieldatum
			},
			zahlen: [
				{ schluessel: 'frist_buch_tage', label: 'Tage / Buch', wert: fristBuch },
				{ schluessel: 'frist_medien_tage', label: 'Tage / Medien', wert: fristMedien },
				{ schluessel: 'max_ausleihen_schueler', label: 'Max. Ausleihen', wert: maxAusleihen }
			],
			onSaved
		});
</script>

<KategorieRahmen
	titel="Ausleihe & Fristen"
	kurz="Rückgabefristen, Ausleih-Obergrenze und der Stichtag der Lernmittelfreiheit."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Bücher und Medien haben getrennte Fristen, weil eine DVD schneller zurückkommt als ein
			Lesebuch. Lernmittel (LMF) laufen nicht auf Tage, sondern bis zum Stichtag am Schuljahresende.
		</p>
		<p>
			Der Ferien-Leseclub überschreibt beide Fristen: Solange er aktiv ist, bekommt JEDE neue
			Ausleihe das eingetragene Ferienende als Rückgabetermin.
		</p>
	{/snippet}

	<div class="grid grid-cols-2 gap-x-8 gap-y-6 md:grid-cols-4">
		<SettingField bind:value={fristBuch} label="Tage / Buch" min={1} max={365} />
		<SettingField bind:value={fristMedien} label="Tage / Medien" min={1} max={365} />
		<SettingField bind:value={maxAusleihen} label="Max. Ausleihen" min={1} max={50} />
		<SettingField
			bind:value={lmfStichtag}
			label="LMF-Stichtag (MM-TT)"
			type="text"
			placeholder="07-31"
			pattern={'\\d{2}-\\d{2}'}
			maxlength={5}
		/>
	</div>

	<div class="flex flex-col gap-4 border-t border-outline-variant pt-6">
		<div class="flex items-start justify-between gap-4">
			<div class="flex flex-col gap-1">
				<span class="text-sm font-medium text-on-surface">Ferien-Leseclub</span>
				<span class="text-sm text-on-surface-variant"
					>Alle neuen Ausleihen laufen bis zum Ferienende statt bis zur Regelfrist.</span
				>
			</div>
			<Switch bind:checked={leseclubAktiv} label="Ferien-Leseclub umschalten" />
		</div>
		{#if leseclubAktiv}
			<div class="max-w-xs">
				<SettingField bind:value={leseclubZieldatum} label="Ferienende" type="date" />
			</div>
		{/if}
	</div>
</KategorieRahmen>

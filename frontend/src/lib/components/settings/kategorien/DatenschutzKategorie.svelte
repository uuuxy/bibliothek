<script>
	/**
	 * @component DatenschutzKategorie
	 * Zwei Löschfristen und zwei Sitzungsfristen (docs/datenschutz_offene_punkte.md
	 * A1/A4).
	 *
	 * Zwei Lesehistorie-Fristen, weil es zwei Verarbeitungstätigkeiten sind:
	 * Schülerbücherei kurz (HBDI-Muster: löschen, sobald nicht mehr notwendig),
	 * Lernmittel lang (die Bestandskartei muss Ausleihe UND Rücklauf nachweisen,
	 * Schadensersatz läuft über die Schulaufsicht).
	 *
	 * Die 0 ist hier ein echter Wert und heißt „aus". Ein LEER geräumtes Feld ist
	 * dagegen keine Angabe und wird gemeldet, statt still als 0 durchzugehen — genau
	 * dieser stille Weg schaltete am 22.08.2026 die Befristung ab (Prüfung A4).
	 */
	import SettingField from '../SettingField.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
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

	let lesehistorie = $state(start.lesehistorie_tage ?? 90);
	let lesehistorieLmf = $state(start.lesehistorie_lernmittel_tage ?? 730);
	let thekeLeeren = $state(start.theke_leeren_minuten ?? 5);
	let sperre = $state(start.sperre_minuten ?? 15);

	const speichern = () =>
		speichereKategorie({
			zahlen: [
				{
					schluessel: 'lesehistorie_tage',
					label: 'Lesehistorie Schülerbücherei',
					wert: lesehistorie,
					min: 0
				},
				{
					schluessel: 'lesehistorie_lernmittel_tage',
					label: 'Lesehistorie Lernmittel',
					wert: lesehistorieLmf,
					min: 0
				},
				{
					schluessel: 'theke_leeren_minuten',
					label: 'Theke leeren nach',
					wert: thekeLeeren,
					min: 0
				},
				{ schluessel: 'sperre_minuten', label: 'Sperrbildschirm nach', wert: sperre, min: 0 }
			],
			onSaved
		});
</script>

<KategorieRahmen
	titel="Datenschutz & Sitzung"
	kurz="Wann eine Ausleihe ihren Namen verliert und wann der Arbeitsplatz sich selbst schließt."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Lesehistorie: Tage nach der Rückgabe, nach denen eine Ausleihe nicht mehr dem Schüler
			zugeordnet ist. Sie zählt weiter in der Statistik, nur ohne Namen. Offene Schadensfälle halten
			die Zuordnung.
		</p>
		<p>
			Sitzung: Minuten ohne Bedienung, bis die Theke den geladenen Schüler fallen lässt bzw. der
			Sperrbildschirm kommt. Entsperrt wird mit dem eigenen Passwort.
		</p>
		<p>Eine getippte 0 schaltet die jeweilige Frist ab.</p>
	{/snippet}

	<div class="grid grid-cols-2 gap-x-8 gap-y-6 md:grid-cols-4">
		<SettingField
			bind:value={lesehistorie}
			label="Lesehistorie Schülerbücherei (Tage)"
			min={0}
			max={3650}
			hint="Vorgabe 90."
		/>
		<SettingField
			bind:value={lesehistorieLmf}
			label="Lesehistorie Lernmittel (Tage)"
			min={0}
			max={3650}
			hint="Vorgabe 730 (zwei Schuljahre)."
		/>
		<SettingField
			bind:value={thekeLeeren}
			label="Theke leeren nach (Min.)"
			min={0}
			max={1440}
			hint="Vorgabe 5."
		/>
		<SettingField
			bind:value={sperre}
			label="Sperrbildschirm nach (Min.)"
			min={0}
			max={1440}
			hint="Vorgabe 15."
		/>
	</div>
</KategorieRahmen>

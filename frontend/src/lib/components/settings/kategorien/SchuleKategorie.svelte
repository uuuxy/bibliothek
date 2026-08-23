<script>
	/**
	 * @component SchuleKategorie
	 * Die Stammdaten der Schule. Sie stehen auf jedem Buchetikett, auf dem
	 * Schülerausweis und im Briefkopf von Mahnung, Bestellung und allen Berichten —
	 * bis zum 04.08.2026 ließen sie sich nirgends eintragen, und auf dem Etikett
	 * stand ersatzweise fest „Schulbibliothek".
	 *
	 * Die Platzhalter sind MUSTERDATEN, nicht die echten Schuldaten. Bis zum
	 * 06.08.2026 stand hier die richtige Adresse als Platzhalter; zusammen mit dem
	 * Hinweis „leer lassen ändert nichts" las sich das Formular wie ausgefüllt, und
	 * niemand trug etwas ein. In der Datenbank standen derweil leere Strings.
	 *
	 * Neu seit dem Speichern je Kategorie: Ein geleertes Feld wird auch geleert.
	 * Vorher hieß leer „nicht anfassen" — eine Notbremse dagegen, dass das Speichern
	 * einer fremden Sektion den Briefkopf löscht. Diese Gefahr besteht nicht mehr,
	 * und ein falscher Eigentumsvermerk ließ sich damit nie wieder entfernen.
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

	let name = $state(start.schule_name ?? '');
	let strasse = $state(start.schule_strasse ?? '');
	let plz = $state(start.schule_plz ?? '');
	let ort = $state(start.schule_ort ?? '');
	let eigentumsvermerk = $state(start.etikett_eigentumsvermerk ?? '');

	const speichern = () =>
		speichereKategorie({
			felder: {
				schule_name: name,
				schule_strasse: strasse,
				schule_plz: plz,
				schule_ort: ort,
				etikett_eigentumsvermerk: eigentumsvermerk
			},
			onSaved
		});
</script>

<KategorieRahmen
	titel="Schule"
	kurz="Name und Anschrift der Schule — sie erscheinen auf jedem Etikett, Ausweis und Briefkopf."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Der Name ist die erste Zeile des Buchetiketts und die Kopfzeile des Schülerausweises; die
			Anschrift bildet den Briefkopf von Mahnungen, Bestellungen und Berichten.
		</p>
		<p>
			Grauer Text im Feld ist ein Beispiel, kein gespeicherter Wert. Ein Feld, das Sie leeren, wird
			auch gespeichert geleert — der Ausweis fällt dann auf seinen Musterkopf zurück.
		</p>
	{/snippet}

	<div class="grid grid-cols-1 gap-x-8 gap-y-5 md:grid-cols-2">
		<SettingField
			bind:value={name}
			label="Name der Schule"
			type="text"
			maxlength={120}
			placeholder="z. B. Städtisches Gymnasium Musterstadt"
		/>
		<SettingField
			bind:value={eigentumsvermerk}
			label="Eigentumsvermerk"
			type="text"
			maxlength={80}
			placeholder="z. B. Eigentum des Landes Hessen"
			hint="Letzte Zeile auf dem Buchetikett. Leer = Vorgabe."
		/>
		<SettingField
			bind:value={strasse}
			label="Straße und Hausnummer"
			type="text"
			maxlength={120}
			placeholder="z. B. Musterstraße 12"
		/>
		<!-- Reicht die drei Zeilen des Aussenrasters durch (siehe SettingField). -->
		<div class="row-span-3 grid grid-cols-3 grid-rows-subgrid gap-x-4">
			<SettingField bind:value={plz} label="PLZ" type="text" maxlength={10} placeholder="12345" />
			<SettingField
				bind:value={ort}
				label="Ort"
				type="text"
				maxlength={80}
				placeholder="Musterstadt"
				class="col-span-2"
			/>
		</div>
	</div>
</KategorieRahmen>

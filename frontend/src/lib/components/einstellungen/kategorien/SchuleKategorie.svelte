<script>
	/**
	 * @component SchuleKategorie
	 * Die Stammdaten der Schule. Sie stehen auf jedem Buchetikett, auf dem Schülerausweis
	 * und im Briefkopf von Mahnung, Bestellung und allen Berichten.
	 *
	 * Die Platzhalter sind Musterdaten, nicht die echten Schuldaten: Mit der richtigen
	 * Adresse als Platzhalter läse sich das Formular wie ausgefüllt, und niemand trüge etwas
	 * ein.
	 *
	 * Ein geleertes Feld wird auch geleert: Mit dem Speichern je Kategorie löscht keine
	 * fremde Sektion den Briefkopf, und „leer heißt nicht anfassen" ließe einen falschen
	 * Eigentumsvermerk nie wieder entfernen.
	 */
	import Feld from '../../ui/Feld.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import { untrack } from 'svelte';

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	// Der Anfangswert ist eine MOMENTAUFNAHME: Das Formular gehört ab hier dem
	// Benutzer, nicht dem Server. Frische Werte kommen nach dem Speichern über den
	// {#key}-Block in KategorieDetail.svelte, der die Kategorie neu aufbaut — ohne
	// untrack würde Svelte hier eine Ableitung erwarten und beim Neuladen die
	// halb getippte Eingabe überschreiben.
	const start = untrack(() => daten);

	// Der Stand beim Öffnen unter den Namen der Anfrage, mit der Vorgabe für eine nie gesetzte
	// Einstellung. Gespeichert wird nur, was davon abweicht (einstellungenSpeichern.js).
	const geladen = {
		schule_name: start.schule_name ?? '',
		schule_strasse: start.schule_strasse ?? '',
		schule_plz: start.schule_plz ?? '',
		schule_ort: start.schule_ort ?? '',
		etikett_eigentumsvermerk: start.etikett_eigentumsvermerk ?? '',
		etikett_eigentumsvermerk_schuelerbuecherei:
			start.etikett_eigentumsvermerk_schuelerbuecherei ?? ''
	};
	let name = $state(geladen.schule_name);
	let strasse = $state(geladen.schule_strasse);
	let plz = $state(geladen.schule_plz);
	let ort = $state(geladen.schule_ort);
	let eigentumsvermerk = $state(geladen.etikett_eigentumsvermerk);
	let eigentumsvermerkSchuelerbuecherei = $state(
		geladen.etikett_eigentumsvermerk_schuelerbuecherei
	);

	const eingabe = () => ({
		geladen,
		felder: {
			schule_name: name,
			schule_strasse: strasse,
			schule_plz: plz,
			schule_ort: ort,
			etikett_eigentumsvermerk: eigentumsvermerk,
			etikett_eigentumsvermerk_schuelerbuecherei: eigentumsvermerkSchuelerbuecherei
		}
	});
</script>

<KategorieRahmen
	titel="Schule"
	kurz="Name und Anschrift der Schule — sie erscheinen auf jedem Etikett, Ausweis und Briefkopf."
	{eingabe}
	{onSaved}
>
	{#snippet mehr()}
		<p>
			Der Name ist die erste Zeile des Buchetiketts und die Kopfzeile des Schülerausweises; die
			Anschrift bildet den Briefkopf von Mahnungen, Bestellungen und Berichten.
		</p>
		<p>
			Welcher Eigentumsvermerk auf ein Etikett kommt, richtet sich danach, woraus das Buch bezahlt
			wurde: nach der Zuordnung seiner Bestellung, und bei Büchern ohne Bestellung danach, ob der
			Titel ein Lernmittel ist. Für die Schülerbücherei gibt es keine Vorgabe — bleibt das Feld
			leer, tragen diese Bücher keinen Vermerk.
		</p>
		<p>
			Grauer Text im Feld ist ein Beispiel, kein gespeicherter Wert. Ein Feld, das Sie leeren, wird
			auch gespeichert geleert — der Ausweis fällt dann auf seinen Musterkopf zurück.
		</p>
	{/snippet}

	<div class="grid grid-cols-1 gap-x-8 gap-y-5 md:grid-cols-2">
		<Feld
			bind:value={name}
			etikett="Name der Schule"
			type="text"
			maxlength={120}
			placeholder="z. B. Städtisches Gymnasium Musterstadt"
		/>
		<Feld
			bind:value={strasse}
			etikett="Straße und Hausnummer"
			type="text"
			maxlength={120}
			placeholder="z. B. Musterstraße 12"
		/>
		<!-- Reicht die drei Zeilen des Aussenrasters durch (siehe Feld). -->
		<div class="row-span-3 grid grid-cols-3 grid-rows-subgrid gap-x-4">
			<Feld bind:value={plz} etikett="PLZ" type="text" maxlength={10} placeholder="12345" />
			<Feld
				bind:value={ort}
				etikett="Ort"
				type="text"
				maxlength={80}
				placeholder="Musterstadt"
				class="col-span-2"
			/>
		</div>
		<!-- md:col-start-1: Die zwei Vermerke gehören zusammen und stehen in einer Zeile. Ohne
		     den Zeilenanfang rutschte der erste neben PLZ/Ort, und der zweite stünde allein
		     darunter. -->
		<Feld
			bind:value={eigentumsvermerk}
			etikett="Eigentumsvermerk"
			class="md:col-start-1"
			type="text"
			maxlength={80}
			placeholder="z. B. Eigentum des Landes Hessen"
			hint="Letzte Zeile auf dem Etikett der Lernmittel. Leer = Vorgabe."
		/>
		<Feld
			bind:value={eigentumsvermerkSchuelerbuecherei}
			etikett="Eigentumsvermerk Schülerbücherei"
			type="text"
			maxlength={80}
			placeholder="z. B. Eigentum der Stadt Musterstadt"
			hint="Für Bücher aus Mitteln des Schulträgers. Leer = kein Vermerk."
		/>
	</div>
</KategorieRahmen>

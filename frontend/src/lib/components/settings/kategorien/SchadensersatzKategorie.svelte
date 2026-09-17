<script>
	/**
	 * @component SchadensersatzKategorie
	 * Die Angaben, die auf den Schadensersatz-Bescheid gehören und die nur die Schule
	 * kennt. Zwei Nummern und die Aufsichtsbehörde sind Pflicht — ein Bescheid ohne
	 * Referenznummer lässt sich nicht bezahlen, weil niemand die Zahlung zuordnen kann.
	 *
	 * Vorbelegt ist, was aus dem Musterschreiben bekannt ist (Frist, Zahlstelle,
	 * Bankverbindung). Was leer bleiben darf, bleibt im Brief einfach weg.
	 */
	import Feld from '../../ui/Feld.svelte';
	import Switch from '../../ui/Switch.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import { untrack } from 'svelte';
	import { speichereKategorie } from '../../../einstellungenSpeichern.js';

	/** @type {{ daten: Record<string, any>, onSaved?: () => void | Promise<void> }} */
	let { daten, onSaved } = $props();

	// Momentaufnahme wie in den übrigen Kategorien: Das Formular gehört ab hier dem
	// Benutzer; frische Werte kommen über den {#key}-Block nach dem Speichern.
	const start = untrack(() => daten);

	let bereichNr = $state(start.bescheid_bereich_nr ?? '');
	let schulnummer = $state(start.bescheid_schulnummer ?? '');
	let aufsicht = $state(start.bescheid_aufsicht ?? '');
	let schulleitung = $state(start.bescheid_schulleitung ?? '');
	let geschaeftszeichen = $state(start.bescheid_geschaeftszeichen ?? '');
	let bearbeiter = $state(start.bescheid_bearbeiter ?? '');
	let durchwahl = $state(start.bescheid_durchwahl ?? '');
	let zahlstelle = $state(start.bescheid_zahlstelle ?? '');
	let bankverbindung = $state(start.bescheid_bankverbindung ?? '');
	let fristTage = $state(start.bescheid_frist_tage ?? 28);
	// Welcher Preis die Grundlage ist (Anforderungsliste des Medienzentrums Nr. 3).
	// Die Frage steht als „immer Einkaufspreis", damit NICHT angehakt die Regel der
	// Arbeitshilfe ist — eine Anlage, in der niemand etwas einstellt, rechnet mit dem
	// heutigen Listenpreis, so wie es der Erlass verlangt.
	let immerKaufpreis = $state(start.ersatzwert_immer_kaufpreis === true);

	const speichern = () =>
		speichereKategorie({
			felder: {
				ersatzwert_immer_kaufpreis: immerKaufpreis,
				bescheid_bereich_nr: bereichNr,
				bescheid_schulnummer: schulnummer,
				bescheid_aufsicht: aufsicht,
				bescheid_schulleitung: schulleitung,
				bescheid_geschaeftszeichen: geschaeftszeichen,
				bescheid_bearbeiter: bearbeiter,
				bescheid_durchwahl: durchwahl,
				bescheid_zahlstelle: zahlstelle,
				bescheid_bankverbindung: bankverbindung
			},
			zahlen: [
				{
					schluessel: 'bescheid_frist_tage',
					label: 'Zahlungsfrist (Tage)',
					wert: fristTage,
					min: 1
				}
			],
			onSaved
		});
</script>

<KategorieRahmen
	titel="Schadensersatz"
	kurz="Grundlage der Berechnung und die Angaben für den Bescheid."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Welcher Preis die Grundlage ist, entscheidet über den Betrag: Der Listenpreis ist, was ein
			Ersatz heute kostet, der Einkaufspreis, was die Schule damals bezahlt hat. Im ersten
			Verleihjahr gilt ohnehin immer der Einkaufspreis — dort ändert der Schalter nichts.
		</p>
		<p>
			Die Referenznummer setzt sich aus vier vierstelligen Blöcken zusammen: Nummer des
			Schulamtsbereichs, Kassenjahr, Schulnummer und einer laufenden Nummer, die das Programm je
			Kassenjahr weiterzählt. Sie wird pro Brief vergeben und nie zweimal — über sie wird eine
			eingehende Zahlung zugeordnet. Ohne die beiden Nummern lässt sich kein Bescheid erstellen.
		</p>
		<p>
			Die Zahlungsfrist rechnet ab dem Briefdatum und steht im Brief als Datum. Zahlstelle und
			Bankverbindung sind mit den Angaben aus dem Musterschreiben vorbelegt.
		</p>
	{/snippet}

	<!-- Die Grundlage der Rechnung steht VOR den Briefangaben: Sie entscheidet über den
	     BETRAG, die Felder darunter nur über die Form des Schreibens. Bauform wie in
	     „Bestellwesen": Titel, ein Satz, Schalter rechts. -->
	<div class="flex items-start justify-between gap-4 border-b border-outline-variant pb-6">
		<div class="flex flex-col gap-1">
			<span class="text-sm font-medium text-on-surface">Immer mit dem Einkaufspreis rechnen</span>
			<span class="text-sm text-on-surface-variant"
				>Aus: Ab dem zweiten Verleihjahr gilt der Listenpreis, wenn einer erfasst ist — so verlangt
				es die Arbeitshilfe des Landes. An: Es gilt immer der Preis, den die Schule bezahlt hat; die
				Begründung im Bescheid sagt das.</span
			>
		</div>
		<Switch bind:checked={immerKaufpreis} label="Berechnungsgrundlage umschalten" />
	</div>

	<div class="grid gap-5 sm:grid-cols-2">
		<Feld
			id="bescheid-bereich"
			label="Nummer des Schulamtsbereichs"
			bind:value={bereichNr}
			placeholder="z. B. 5830"
		/>
		<Feld
			id="bescheid-schulnummer"
			label="Schulnummer"
			bind:value={schulnummer}
			placeholder="z. B. 1234"
		/>
	</div>

	<Feld
		id="bescheid-aufsicht"
		label="Aufsichtsbehörde (Name und Anschrift)"
		bind:value={aufsicht}
		placeholder="Name, Straße, PLZ Ort"
	/>
	<Feld
		id="bescheid-schulleitung"
		label="Name der Schulleitung (Unterschriftszeile)"
		bind:value={schulleitung}
	/>

	<div class="grid gap-5 sm:grid-cols-3">
		<Feld id="bescheid-gz" label="Geschäftszeichen" bind:value={geschaeftszeichen} />
		<Feld id="bescheid-bearbeiter" label="Bearbeiter" bind:value={bearbeiter} />
		<Feld id="bescheid-durchwahl" label="Durchwahl" bind:value={durchwahl} />
	</div>

	<Feld id="bescheid-zahlstelle" label="Zahlstelle" bind:value={zahlstelle} />
	<Feld
		id="bescheid-bank"
		label="Bankverbindung (eine Angabe je Zeile)"
		bind:value={bankverbindung}
		mehrzeilig
		zeilen={4}
	/>
	<Feld
		id="bescheid-frist"
		label="Zahlungsfrist (Tage ab Briefdatum)"
		type="number"
		min="1"
		bind:value={fristTage}
		feld="w-32"
	/>
</KategorieRahmen>

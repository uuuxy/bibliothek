<script>
	/**
	 * @component BestellwesenKategorie
	 * Zwei Schalter, die entscheiden, womit das Bestellwesen überhaupt arbeitet: ob
	 * es Bedarf meldet und ob es mit Geld rechnet.
	 */
	import Feld from '../../ui/Feld.svelte';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import Switch from '../../ui/Switch.svelte';
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
		bestellbedarf_warnung_aktiv: start.bestellbedarf_warnung_aktiv ?? true,
		preise_erfassen: start.preise_erfassen ?? true,
		bestelllink_gueltigkeit_tage: start.bestelllink_gueltigkeit_tage ?? 21
	};
	let warnungAktiv = $state(geladen.bestellbedarf_warnung_aktiv);
	let preiseErfassen = $state(geladen.preise_erfassen);
	let linkTage = $state(geladen.bestelllink_gueltigkeit_tage);

	const eingabe = () => ({
		geladen,
		felder: {
			bestellbedarf_warnung_aktiv: warnungAktiv,
			preise_erfassen: preiseErfassen
		},
		zahlen: [
			{
				schluessel: 'bestelllink_gueltigkeit_tage',
				label: 'Bestätigungs-Link gültig (Tage)',
				wert: linkTage,
				min: 1
			}
		]
	});
</script>

<KategorieRahmen
	titel="Bestellwesen"
	kurz="Ob fehlende Lernmittel gemeldet werden und ob mit Preisen gearbeitet wird"
	{eingabe}
	{onSaved}
>
	{#snippet mehr()}
		<p>
			Die Bestellbedarf-Warnung meldet Schulbücher, deren Bestand nicht für die Kinder reicht, die
			sie im nächsten Schuljahr brauchen. Wie viele das sind, zeigen die Ausleihen dieses
			Schuljahres. Aus bedeutet: keine Bestellbedarf-Liste.
		</p>
		<p>
			Preise an = Preisfeld im Warenkorb, Betragsspalten in der Historie, Berichte mit Summen. Aus =
			dieselben Listen zählen Exemplare. Bereits erfasste Beträge bleiben gespeichert und erscheinen
			wieder, sobald der Schalter zurückgelegt wird.
		</p>
	{/snippet}

	<div class="flex flex-col gap-6">
		<div class="flex items-start justify-between gap-4">
			<div class="flex flex-col gap-1">
				<span class="text-sm font-medium text-on-surface">Bestellbedarf-Warnung</span>
				<span class="text-sm text-on-surface-variant"
					>Meldet Schulbücher, deren Bestand nicht für die Kinder im nächsten Schuljahr reicht</span
				>
			</div>
			<Switch bind:checked={warnungAktiv} etikett="Bestellbedarf-Warnung umschalten" />
		</div>

		<div class="flex items-start justify-between gap-4 border-t border-outline-variant pt-6">
			<div class="flex flex-col gap-1">
				<span class="text-sm font-medium text-on-surface">Preise erfassen</span>
				<span class="text-sm text-on-surface-variant"
					>Aus: Bestellhistorie und Berichte zählen Exemplare statt Euro</span
				>
			</div>
			<Switch bind:checked={preiseErfassen} etikett="Preise im Bestellwesen umschalten" />
		</div>

		<div class="border-t border-outline-variant pt-6">
			<Feld
				type="number"
				min={1}
				bind:value={linkTage}
				etikett="Bestätigungs-Link gültig (Tage)"
				hint="Solange kann der Händler über den Link aus der Bestellmail bestätigen und Etiketten drucken. Gilt für neu erzeugte Links; laufende behalten ihr Datum."
				feld="w-32"
			/>
		</div>
	</div>
</KategorieRahmen>

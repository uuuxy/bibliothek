<script>
	/**
	 * @component ErreichbarkeitKategorie
	 * Zwei Adressen, die nach AUSSEN zeigen — und die beide dieselbe Regel tragen:
	 * leer heißt aus.
	 *
	 * Sie stehen deshalb zusammen und nicht bei den Schul-Stammdaten, wo genau diese
	 * Regel der Nachbarschaft widerspräche (dort heißt leer schlicht leer). Drei
	 * Leer-Regeln in einem Formular waren der eigentliche Überladungs-Befund vom
	 * 23.08.2026; eine Kategorie hat jetzt genau eine.
	 *
	 * Die öffentliche Adresse ist keine Kosmetik: Aus ihr entsteht der
	 * Bestätigungs-Link in der Bestellmail. Der Server kann sie nicht erraten — hinter
	 * dem Reverse-Proxy sieht er nur seinen internen Namen, und ein Link auf
	 * „localhost" wäre beim Lieferanten wertlos (Fund vom 17.08.2026: Die Mails gingen
	 * monatelang ohne den Link raus, um dessentwillen es sie gibt).
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

	let adresse = $state(start.oeffentliche_adresse ?? '');
	let alarmEmpfaenger = $state(start.alarm_empfaenger ?? '');

	const speichern = () =>
		speichereKategorie({
			felder: { oeffentliche_adresse: adresse, alarm_empfaenger: alarmEmpfaenger },
			onSaved
		});
</script>

<KategorieRahmen
	titel="Erreichbarkeit & Alarme"
	kurz="Unter welcher Adresse Dritte das System erreichen und wer die Alarm-Mails bekommt — leer schaltet beides ab."
	{speichern}
>
	{#snippet mehr()}
		<p>
			Aus der öffentlichen Adresse baut das System den Bestätigungs-Link, den ein Lieferant mit der
			Bestellmail bekommt. Ohne sie geht die Bestellung ohne Link raus.
		</p>
		<p>
			Alarm-Empfänger bekommen die Kritisch-Meldungen der Betriebsbereitschaft (mehrere Adressen mit
			Komma). Bleibt das Feld leer, gehen sie an alle aktiven Admin-Konten — ein Alarm, der
			niemanden erreicht, ist keiner.
		</p>
	{/snippet}

	<div class="grid grid-cols-1 gap-x-8 gap-y-5 md:grid-cols-2">
		<SettingField
			bind:value={adresse}
			label="Öffentliche Adresse"
			type="text"
			maxlength={200}
			placeholder="https://bibliothek.schule.de"
			hint="Leer = keine Bestätigungs-Links verschicken."
		/>
		<SettingField
			bind:value={alarmEmpfaenger}
			label="Alarm-Empfänger"
			type="text"
			maxlength={300}
			placeholder="it@schule.de, leitung@schule.de"
			hint="Leer = alle aktiven Admin-Konten."
		/>
	</div>
</KategorieRahmen>

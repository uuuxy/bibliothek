<!-- @component SignaturFeld — die Signatur steht auf dem Rückenetikett des Buchs und ist bei
     der Neuanlage eines Bibliotheksbuchs Pflicht; Lernmittel tragen kein Etikett. Vorgeschlagen
     werden die Signaturen, die im Bestand vorkommen: Ein erfundenes Wort stünde in keinem Regal.
     Die Pflicht zeigt der Stern an der Beschriftung, das leere Feld der Fehlerzustand. -->
<script>
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	import { ladeSignaturen } from '../../../../lib/utils/signaturen.js';
	import { signaturFehlt, signaturPflicht } from './buch_form_optionen.js';

	/** @type {{ formular: any }} */
	let { formular = $bindable() } = $props();

	/** @type {{ signatur: string, titel: number }[]} */
	let vorhandene = $state([]);
	$effect(() => {
		let abgebrochen = false;
		ladeSignaturen().then((liste) => {
			if (!abgebrochen) vorhandene = liste;
		});
		return () => {
			abgebrochen = true;
		};
	});
	// Platzhalter aus dem Bestand: die drei größten Regale. Ohne Bestand steht kein Beispiel da.
	const beispiele = $derived(
		[...vorhandene]
			.sort((a, b) => b.titel - a.titel)
			.slice(0, 3)
			.map((s) => s.signatur)
			.join(', ')
	);
	const pflicht = $derived(signaturPflicht(formular));
	const fehlt = $derived(signaturFehlt(formular));
</script>

<Feld
	id="buch-signatur"
	label={pflicht ? 'Signatur (Buchrücken) *' : 'Signatur (Buchrücken)'}
	bind:value={formular.signatur}
	list="signatur-vorschlaege"
	placeholder={beispiele ? `z. B. ${beispiele}` : 'Signatur des Regals'}
	required={pflicht}
	ungueltig={fehlt}
	hint={fehlt
		? 'Ohne Signatur kein Etikett — bitte Systematik-Kürzel eintragen (Speichern ist bis dahin gesperrt).'
		: formular.istLernmittel
			? 'Lernmittel tragen kein Rückenetikett — die Signatur ist hier nur eine Notiz.'
			: 'Wird 1:1 auf das Rücken-Etikett gedruckt — am besten eine vorhandene Regaladresse.'}
/>
<datalist id="signatur-vorschlaege">
	{#each vorhandene as s (s.signatur)}
		<option value={s.signatur}>{s.signatur} — {s.titel} Titel</option>
	{/each}
</datalist>

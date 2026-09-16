<!-- @component LeserKontaktFelder — Anschrift und die beiden E-Mail-Adressen der Akte.

     Eigene Datei, weil LeserEditFelder.svelte sonst über die 200-Zeilen-Grenze wächst
     (frontend-hygiene-dateigroesse.test.js).

     Hier stehen ZWEI Adressen, und sie gehören verschiedenen Personen:

       - Eltern E-Mail: die Adresse der Erziehungsberechtigten an der Leserzeile
         (leser.eltern_email). Ein Kollege hat keine Eltern — bei ihm ist das Feld
         verschlossen, wie Klasse und Abgangsjahr.
       - Schul-E-Mail: die dienstliche Adresse AM KONTO (benutzer.email). Sie ist keine
         Kontaktangabe, sondern der Schlüssel: An ihr erkennt die Anmeldung (IMAP) die
         Person, und an ihr hängt der Schutz vor dem Doppeleintrag.

     Bis zum 16.09.2026 gab es hier nur die Eltern-Adresse — auch in der Akte einer
     Lehrkraft, mit dem Platzhalter „eltern@schule.de". Die Adresse, an der seit demselben
     Tag ihre Identität hängt, kam in der Maske nicht vor: Beim Anlegen Pflicht, beim
     Bearbeiten weder sichtbar noch änderbar. Wer sie nachtragen wollte, trug sie ins
     Eltern-Feld.

     Nachtragen, nicht ändern: Steht am Konto bereits eine Adresse, ist das Feld eine
     Anzeige (readonly, damit man sie lesen und kopieren kann). Geändert wird sie in der
     Benutzerverwaltung — sie ist die Identität des Kontos, und zwei Türen zu demselben
     Zustand kennt nur eine die Regeln (api/student_schul_email.go). -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Abschnitt from '../ui/Abschnitt.svelte';
	import { istKollegium } from '../../leserArt.js';

	/** @type {{ formData: any, kontoVorhanden?: boolean }} */
	let { formData, kontoVorhanden = false } = $props();

	const kollege = $derived(istKollegium({ art: formData.art }));
</script>

<section>
	<Abschnitt titel="Kontaktdaten" />

	<!-- EIN flaches Raster, keine Hülle: Eine <div>-Hülle spannt eine Rasterzeile, das
	     Feld braucht drei (Subgrid) — es rutscht aus der Reihe (feld-huellen.test.js). -->
	<div class="grid grid-cols-4 gap-4 md:grid-cols-8">
		<Feld
			id="strasse"
			label="Straße"
			class="col-span-3"
			bind:value={formData.strasse}
			placeholder="Musterstraße"
		/>
		<Feld id="hausnummer" label="Nr." bind:value={formData.hausnummer} placeholder="12a" />
		<Feld
			id="plz"
			label="PLZ"
			class="col-span-2"
			bind:value={formData.plz}
			placeholder="12345"
			maxlength={5}
			feld="font-mono"
		/>
		<Feld
			id="ort"
			label="Ort"
			class="col-span-2"
			bind:value={formData.ort}
			placeholder="Musterstadt"
		/>

		<Feld
			id="schul_email"
			label={kollege && !kontoVorhanden ? 'Schul-E-Mail *' : 'Schul-E-Mail'}
			class="col-span-4"
			type="email"
			bind:value={formData.email}
			placeholder={kollege ? 'vorname.nachname@schule.de' : ''}
			disabled={!kollege}
			readonly={kollege && kontoVorhanden}
			hint={!kollege
				? 'Ein Schüler hat keinen Zugang.'
				: kontoVorhanden
					? 'Der Zugang besteht. Die Adresse ist die Anmeldung selbst — ändern in der Benutzerverwaltung.'
					: 'Damit entsteht der Zugang zu „Mein Portal“. Meldet sich die Person später selbst an, findet die Anmeldung diesen Eintrag statt einen zweiten anzulegen.'}
		/>
		<Feld
			id="eltern_email"
			label="Eltern E-Mail"
			class="col-span-4"
			type="email"
			bind:value={formData.eltern_email}
			placeholder={kollege ? '' : 'eltern@schule.de'}
			disabled={kollege}
			hint={kollege ? 'Gehört zum Schüler.' : undefined}
		/>
	</div>
</section>

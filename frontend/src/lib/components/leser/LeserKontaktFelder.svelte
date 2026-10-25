<!-- @component LeserKontaktFelder — Anschrift und die beiden E-Mail-Adressen der Akte.

     Eigene Datei, weil LeserakteFelder.svelte sonst über die 200-Zeilen-Grenze wächst
     (frontend-hygiene-dateigroesse.test.js).

     Hier stehen zwei Adressen, und sie gehören verschiedenen Personen:

       - Eltern E-Mail: die Adresse der Erziehungsberechtigten an der Leserzeile
         (leser.eltern_email). Ein Kollege hat keine Eltern — bei ihm ist das Feld
         verschlossen, wie Klasse und Abgangsjahr.
       - Schul-E-Mail: die dienstliche Adresse am Konto (benutzer.email). Sie ist keine
         Kontaktangabe, sondern der Schlüssel: An ihr erkennt die Anmeldung (IMAP) die
         Person, und an ihr hängt der Schutz vor dem Doppeleintrag.

     Nachtragen, nicht ändern: Steht am Konto bereits eine Adresse, ist das Feld eine
     Anzeige (readonly, damit man sie lesen und kopieren kann). Geändert wird sie in der
     Benutzerverwaltung: Sie ist die Identität des Kontos, und von zwei Türen zu demselben
     Zustand kennt nur eine die Regeln (api/leser/leser_schul_email.go).

     Praktikum und Fachbereich bekommen keinen Zugang (artMitKonto): Bei ihnen
     ist das Feld verschlossen wie beim Schüler — außer ein Konto besteht schon, dann bleibt
     es die Anzeige. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Abschnitt from '../ui/Abschnitt.svelte';
	import { istKollegium, artMitKonto } from './leserArt.js';

	/** @type {{ eingaben: any, kontoVorhanden?: boolean }} */
	let { eingaben, kontoVorhanden = false } = $props();

	const kollege = $derived(istKollegium({ art: eingaben.art }));
	// Nachtragbar ist die Adresse nur, wo ein Zugang zur Art gehört und noch keiner besteht.
	const nachtragbar = $derived(artMitKonto(eingaben.art) && !kontoVorhanden);
</script>

<section>
	<Abschnitt titel="Kontaktdaten" />

	<!-- EIN flaches Raster, keine Hülle: Eine <div>-Hülle spannt eine Rasterzeile, das
	     Feld braucht drei (Subgrid) — es rutscht aus der Reihe (feld-huellen.test.js). -->
	<div class="grid grid-cols-4 gap-4 md:grid-cols-8">
		<Feld
			id="strasse"
			etikett="Straße"
			class="col-span-3"
			bind:value={eingaben.strasse}
			placeholder="Musterstraße"
		/>
		<Feld id="hausnummer" etikett="Nr." bind:value={eingaben.hausnummer} placeholder="12a" />
		<Feld
			id="plz"
			etikett="PLZ"
			class="col-span-2"
			bind:value={eingaben.plz}
			placeholder="12345"
			maxlength={5}
			feld="font-mono"
		/>
		<Feld
			id="ort"
			etikett="Ort"
			class="col-span-2"
			bind:value={eingaben.ort}
			placeholder="Musterstadt"
		/>

		<Feld
			id="schul_email"
			etikett={nachtragbar ? 'Schul-E-Mail *' : 'Schul-E-Mail'}
			class="col-span-4"
			type="email"
			bind:value={eingaben.email}
			placeholder={nachtragbar ? 'vorname.nachname@schule.de' : ''}
			disabled={!nachtragbar && !(kollege && kontoVorhanden)}
			readonly={kollege && kontoVorhanden}
			hint={!kollege
				? 'Ein Schüler hat keinen Zugang.'
				: kontoVorhanden
					? 'Der Zugang besteht. Die Adresse ist die Anmeldung selbst — ändern in der Benutzerverwaltung.'
					: nachtragbar
						? 'Damit entsteht der Zugang zu „Mein Portal“. Meldet sich die Person später selbst an, findet die Anmeldung diesen Eintrag statt einen zweiten anzulegen.'
						: 'Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal“.'}
		/>
		<Feld
			id="eltern_email"
			etikett="Eltern E-Mail"
			class="col-span-4"
			type="email"
			bind:value={eingaben.eltern_email}
			placeholder={kollege ? '' : 'eltern@schule.de'}
			disabled={kollege}
			hint={kollege ? 'Gehört zum Schüler.' : undefined}
		/>
	</div>
</section>

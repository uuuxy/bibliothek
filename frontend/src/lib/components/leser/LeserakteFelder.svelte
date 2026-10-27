<!-- @component LeserakteFelder — die Eingabefelder der Leserakte, eine Maske für jeden.

     Für jeden steht dasselbe an derselben Stelle. Abgeschaltet statt versteckt ist nur, was
     wirklich nicht gehen darf:

       - LUSD-ID: Die Datenbank verbietet sie einem Kollegen
         (chk_leser_nur_schueler_werden_abgaenger). Durch die LUSD kommen nur Schüler.
       - Klasse und Abgangsjahr: Einer Lehrkraft oder LiV ist keine Klasse zugeordnet. Das
         Abgangsjahr hängt an der Klasse, der Server leitet es aus ihr ab
         (repository.AbgaengerJahr), also gehören beide zusammen.
       - Die Art über die Schüler-Grenze: Ein Schüler bleibt Schüler.

     Die Art steht zuerst: An ihr hängt, welche Felder überhaupt leben, und der Anlege-Dialog
     fragt ebenfalls zuerst nach ihr (LeserAnlegenDialog.svelte). Die LUSD-ID steht bei den
     Schuldaten, nicht bei den persönlichen Daten: Sie ist keine Angabe über die Person,
     sondern die Kennung der Schulverwaltung. Die Ausweisnummer steht als Letztes im
     Abschnitt, denn bei einem Kollegen ist sie das einzige bediente Feld. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import Select from '../ui/Select.svelte';
	import Abschnitt from '../ui/Abschnitt.svelte';
	import LeserArtWahl from './LeserArtWahl.svelte';
	import LeserKontaktFelder from './LeserKontaktFelder.svelte';
	import { istKollegium, LESER_ARTEN } from './leserArt.js';
	import { klassenLadefehler } from './klassenVorschlaege.svelte.js';

	/**
	 * `eingaben` ist der $state-Proxy aus useLeserakteFormular und wird hier direkt an den
	 * Feldern gebunden — kein $bindable: Gebunden werden die EIGENSCHAFTEN des Objekts,
	 * das Objekt selbst wird nie ersetzt. Ein $bindable verlangte vom Aufrufer ein
	 * `bind:`, und dort ist eingaben ein `const` aus der Hook-Destrukturierung.
	 * @type {{ eingaben: any, klassen?: string[], klassenFehler?: string, lusdVerknuepft?: boolean, kontoVorhanden?: boolean }}
	 */
	let {
		eingaben,
		klassen = [],
		klassenFehler = '',
		lusdVerknuepft = false,
		kontoVorhanden = false
	} = $props();

	// Auswählen statt tippen: Ein Tippfehler legte sonst eine Klasse an, die es an der Schule
	// nicht gibt. Die eigene Klasse steht immer mit in der Liste, auch wenn sie dort fehlt
	// (etwa bei einem Leser im Papierkorb).
	const klassenOptionen = $derived(
		[...new Set([...klassen, eingaben.klasse].filter(Boolean))].map((k) => ({
			value: k,
			label: k
		}))
	);

	const kollege = $derived(istKollegium({ art: eingaben.art }));

	// Die Grenze verläuft in BEIDE Richtungen: Wer Schüler ist, kann nur Schüler bleiben;
	// wer keiner ist, wird keiner. Gesperrt ist deshalb immer die jeweils andere Seite.
	const gesperrteArten = $derived(
		kollege ? ['schueler'] : LESER_ARTEN.filter((a) => a !== 'schueler')
	);

	// Der Stern markiert, was der Server verlangt — und er hängt an der ART, wie die
	// Pflicht selbst (chk_leser_schueler_pflichtfelder). Ohne ihn erfährt man erst beim
	// Speichern, dass ein Feld nicht leer bleiben darf.
	const schuelerPflicht = $derived(kollege ? '' : ' *');
</script>

<!-- ── Persönliche Daten ──────────────────────────────── -->
<section>
	<Abschnitt titel="Persönliche Daten" />

	<!-- Die Art, wie bei Littera in den Stammdaten: eine Auswahlliste (sieben Arten), im
	     selben Raster wie die Felder darunter und so breit wie eines davon. -->
	<div class="mb-4 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
		<LeserArtWahl
			id="art"
			bind:art={eingaben.art}
			gesperrt={gesperrteArten}
			hint={kollege
				? 'Innerhalb des Kollegiums wechselbar. Ein Schüler kommt aus der LUSD — dorthin führt kein Weg.'
				: 'Ein Schüler kommt aus der LUSD und bleibt Schüler.'}
		/>
	</div>

	<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
		<Feld id="vorname" etikett="Vorname *" bind:value={eingaben.vorname} feld="font-semibold" />
		<Feld id="nachname" etikett="Nachname *" bind:value={eingaben.nachname} feld="font-semibold" />
		<Feld
			id="geburtsdatum"
			etikett="Geburtsdatum{schuelerPflicht}"
			type="date"
			bind:value={eingaben.geburtsdatum}
			hint={kollege ? undefined : 'Der Schlüssel, an dem der LUSD-Import wiedererkennt.'}
		/>
	</div>
</section>

<!-- ── Schuldaten ─────────────────────────────────────── -->
<section>
	<Abschnitt titel="Schuldaten" />

	<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
		<!-- Wie ui/Feld mit Beschriftung: drei Zeilen im Subgrid, damit die Nachbarn fluchten. -->
		<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
			<label for="klasse" class="text-sm font-medium text-on-surface-variant"
				>Klasse{schuelerPflicht}</label
			>
			<Select
				id="klasse"
				bind:value={eingaben.klasse}
				options={klassenOptionen}
				placeholder="Klasse wählen"
				disabled={kollege}
			/>
			{#if kollege}
				<span class="text-xs text-on-surface-variant">Nur ein Schüler hat eine Klasse.</span>
			{:else if klassenFehler}
				<!-- Zur Wahl steht dann nur die eigene Klasse; der Fehlertext sagt, warum. -->
				<span class="text-xs text-error">{klassenLadefehler(klassenFehler)}</span>
			{/if}
		</div>
		<Feld
			id="abgangsjahr"
			etikett="Abgangsjahr"
			type="number"
			bind:value={eingaben.abgaenger_jahr}
			feld="font-semibold"
			disabled={kollege}
			hint={kollege ? 'Gehört zur Klasse.' : undefined}
		/>
		<!-- Die LUSD-ID ist beim Schüler kontrolliert nachtragbar: nur setzbar, solange sie
		     leer ist (Waise adoptieren). Ist sie gesetzt, bleibt sie schreibgeschützt — der
		     Server lehnt Änderung und Leerung ohnehin ab. -->
		<Feld
			id="lusd_id"
			etikett="LUSD-ID"
			bind:value={eingaben.lusd_id}
			feld="font-mono"
			disabled={kollege || lusdVerknuepft}
			hint={kollege
				? 'Durch die LUSD kommen nur Schüler.'
				: lusdVerknuepft
					? 'Bereits mit der LUSD verknüpft — Änderung nur über den Import.'
					: 'Nachtragbar: verknüpft diesen Schüler mit der LUSD.'}
		/>
		<Feld
			id="barcode"
			etikett="Ausweisnummer{schuelerPflicht}"
			bind:value={eingaben.barcode_id}
			feld="font-mono"
		/>

		<!-- Kein Status-Dropdown: „status" ist ein abgeleiteter Lesewert
		     (aktiv/gesperrt/abgaenger aus ist_gesperrt/ist_abgaenger) ohne eigene
		     DB-Spalte. Sperren läuft übers Lock-Modal, Abgänger über das Abgangsjahr —
		     ein editierbares Feld hier wurde vom Backend still verworfen. -->
	</div>
</section>

<LeserKontaktFelder {eingaben} {kontoVorhanden} />

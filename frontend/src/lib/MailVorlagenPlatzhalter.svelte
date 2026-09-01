<script>
	/**
	 * @component MailVorlagenPlatzhalter
	 * Verwendung + erlaubte Platzhalter JE VORLAGEN-TYP, unter dem Vorlagen-Editor.
	 *
	 * Bis zum 01.09.2026 zeigte der Editor für jede Vorlage dieselben vier
	 * Platzhalter an — für die Händler-Mail ersetzte der Renderer keinen einzigen
	 * davon: Wer die Vorlage laut Anleitung umformulierte, schickte dem Buchhändler
	 * wörtlich „{{.BuchListe}}". Diese Liste hält api/mail_vorlagen_platzhalter_test.go
	 * deckungsgleich mit den Go-Renderern (reports_pdf.go, bestellmail_text.go) —
	 * wer dort einen Platzhalter ändert, muss hier nachziehen, sonst wird das Gate rot.
	 *
	 * @prop {string} typ - Vorlagen-Typ (mail_vorlagen.typ), z. B. MAHNUNG_ELTERN.
	 */
	let { typ } = $props();

	const vorlagenInfo = {
		MAHNUNG_ELTERN: {
			verwendung:
				'Gedruckter Eltern-Mahnbrief (Fensterkuvert) — es geht keine Mail an Eltern. Die Mahn-Mails an Klassenleitungen haben eigene, feste Texte.',
			platzhalter: ['{{.Vorname}}', '{{.Nachname}}', '{{.BuchListe}}', '{{.Frist}}']
		},
		BESTELLUNG_HAENDLER: {
			verwendung:
				'Bestellmail an den Buchhändler. Fehlt {{.BestaetigungsLink}} im Text, hängt das System den Bestätigungs-Link automatisch als eigenen Absatz an.',
			platzhalter: [
				'{{.Datum}}',
				'{{.Kundennummer}}',
				'{{.AnzahlTitel}}',
				'{{.AnzahlExemplare}}',
				'{{.BestaetigungsLink}}'
			]
		}
	};
	const info = $derived(vorlagenInfo[typ] ?? null);
</script>

<!-- Flacher Akzent statt Kachel. Eine Vorlage ohne Versandweg bekommt eine
     Warnung statt einer Anleitung. -->
{#if info}
	{#if info.platzhalter.length === 0}
		<div class="border-l-2 border-error py-1 pl-4">
			<h4 class="mb-1 text-sm font-bold text-on-surface">Ohne Wirkung</h4>
			<p class="text-xs leading-relaxed text-on-surface-variant">{info.verwendung}</p>
		</div>
	{:else}
		<div class="border-l-2 border-primary py-1 pl-4">
			<h4 class="mb-1 text-sm font-bold text-on-surface">Erlaubte Platzhalter</h4>
			<p class="text-xs leading-relaxed text-on-surface-variant">
				{info.verwendung}
				<br />
				{#each info.platzhalter as p (p)}
					<code
						class="bg-surface-container mt-1 mr-1 inline-block rounded px-1.5 py-0.5 text-on-surface"
						>{p}</code
					>
				{/each}
			</p>
		</div>
	{/if}
{/if}

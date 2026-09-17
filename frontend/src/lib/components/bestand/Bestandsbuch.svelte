<!-- @component Bestandsbuch — Zugangs- und Abgangsbuch, ein Bauteil.

     Punkt 1 des Protokolls vom 16.09.2026 („Zugangs- und Abgangsbuch fehlen"). Beide Bücher
     sind derselbe Nachweis in zwei Richtungen: Zeitraum wählen, Liste je Topf, Blatt zum
     Abheften. Zwei getrennte Bauteile wären zwei Orte für dieselben vier Entscheidungen —
     und beim ersten Unterschied (Wortlaut, Spaltenhöhe, Fehlermeldung) sähen sie aus wie
     zwei Programme.

     Abschnitte und Überschriften kommen FERTIG vom Server. Hier selbst zu gruppieren war
     der erste Entwurf des Abgangsbuchs — und ließ Blatt und Bildschirm sofort
     auseinanderlaufen („Lernmittelfreiheit (Land)" gegen „Lernmittel (Land)").

     Der Zeitraum ebenso: Beim ersten Laden fragt das Bauteil ohne Datumsangabe, und der
     Server antwortet mit dem laufenden Schulhalbjahr (Stichtage 15.3./15.9.). Die Felder
     zeigen danach, was der Server tatsächlich gelesen hat. -->
<script>
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { formatDatum } from '../../utils/format.js';
	import { Printer } from '@lucide/svelte';

	/**
	 * @type {{
	 *   pfad: string,
	 *   buchname: string,
	 *   wortSingular: string,
	 *   spalten: {kopf: string, feld: string, klasse?: string}[]
	 * }}
	 */
	let { pfad, buchname, wortSingular, spalten } = $props();

	let von = $state('');
	let bis = $state('');
	let buch = $state(/** @type {any} */ (null));
	let laeuft = $state(true);
	let fehler = $state('');

	const abschnitte = $derived(buch?.abschnitte ?? []);

	// Der Ausdruck nimmt den GELADENEN Zeitraum, nicht den in den Eingabefeldern.
	//
	// Bis zum 17.09.2026 las er `von`/`bis` — und damit das, was gerade im Feld stand. Wer
	// ein Datum änderte und direkt auf „Ausdrucken" klickte, ohne „Anzeigen" zu drücken,
	// heftete ein Blatt ab, dessen Zeitraum er nie geprüft hatte: auf dem Bildschirm das
	// eine Halbjahr, auf dem Papier ein anderes. Bei einem Nachweis, der unterschrieben
	// wird, ist das der teuerste Unterschied von allen (Rasterdurchgang 17.09.2026, Frage 3).
	//
	// Vor der ersten Antwort bleibt die Adresse ohne Zeitraum — dann antwortet der Server
	// mit dem laufenden Halbjahr, und genau das steht gleich darauf auch auf dem Schirm.
	const druckZeitraum = $derived(
		buch
			? `?von=${encodeURIComponent(String(buch.von).slice(0, 10))}&bis=${encodeURIComponent(String(buch.bis).slice(0, 10))}`
			: ''
	);
	const druckAdresse = $derived(`/api/bestand/${pfad}/pdf${druckZeitraum}`);

	async function laden() {
		laeuft = true;
		fehler = '';
		try {
			const adresse =
				von && bis
					? `/api/bestand/${pfad}?von=${encodeURIComponent(von)}&bis=${encodeURIComponent(bis)}`
					: `/api/bestand/${pfad}`;
			const res = await apiFetch(adresse);
			if (!res.ok) {
				const daten = await res.json().catch(() => ({}));
				fehler = daten.error || `Das ${buchname} konnte nicht geladen werden.`;
				return;
			}
			buch = await res.json();
			von = String(buch.von).slice(0, 10);
			bis = String(buch.bis).slice(0, 10);
		} catch (e) {
			fehler = `Das ${buchname} konnte nicht geladen werden.`;
			console.error(`${buchname}:`, e);
		} finally {
			laeuft = false;
		}
	}

	onMount(laden);
</script>

<div class="py-6 space-y-6">
	<div class="flex flex-wrap items-end gap-4">
		<Feld bind:value={von} label="Von" type="date" class="w-44" />
		<Feld bind:value={bis} label="Bis" type="date" class="w-44" />
		<Button variant="secondary" onclick={laden} disabled={laeuft}>Anzeigen</Button>
		<a
			href={druckAdresse}
			class="ml-auto inline-flex items-center gap-2 h-9 px-4 rounded-xl bg-primary text-on-primary text-sm font-semibold"
		>
			<Printer class="h-4 w-4 shrink-0" aria-hidden="true" />
			Ausdrucken
		</a>
	</div>

	{#if laeuft}
		<Ladekreis />
	{:else if fehler}
		<LadeFehler titel="{buchname} nicht geladen" text={fehler} onerneut={laden} />
	{:else if buch}
		{#each abschnitte as abschnitt (abschnitt.topf)}
			<section class="space-y-2">
				<h3 class="text-sm font-semibold text-on-surface">
					{abschnitt.titel} · {abschnitt.zeilen.length}
					{abschnitt.zeilen.length === 1 ? 'Exemplar' : 'Exemplare'}
				</h3>
				{#if abschnitt.zeilen.length === 0}
					<p class="text-sm text-on-surface-variant">
						Kein {wortSingular} in diesem Zeitraum.
					</p>
				{:else}
					<Tabelle beschriftung="{buchname} — {abschnitt.titel}">
						<thead>
							<tr>
								{#each spalten as spalte (spalte.feld)}
									<th>{spalte.kopf}</th>
								{/each}
							</tr>
						</thead>
						<tbody>
							{#each abschnitt.zeilen as zeile (zeile.barcode)}
								<tr>
									{#each spalten as spalte (spalte.feld)}
										<td class={spalte.klasse ?? ''}>
											{spalte.feld === 'datum' ? formatDatum(zeile.datum) : zeile[spalte.feld]}
										</td>
									{/each}
								</tr>
							{/each}
						</tbody>
					</Tabelle>
				{/if}
			</section>
		{/each}

		<!-- Was NICHT auf der Liste steht, gehört darunter — sonst behauptet ein Nachweis
		     Vollständigkeit, die er nicht hat. Beim Abgangsbuch sind das die Exemplare ohne
		     Abgangsdatum UND die körperlich gelöschten (Rasterdurchgang 17.09.2026), beim
		     Zugangsbuch die ohne hinterlegte Bestellung. Das jeweils fremde Feld fehlt in der
		     Antwort schlicht, und `undefined > 0` ist falsch — der Block bleibt dann weg. -->
		{#if buch.ohne_zeitpunkt > 0}
			<p class="text-sm text-on-surface-variant border-t border-outline-variant pt-3">
				{buch.ohne_zeitpunkt} weitere Exemplare sind ausgesondert, ohne dass ein Abgangsdatum bekannt
				ist. Sie wurden vor der Einführung des Abgangsbuchs ausgebucht und lassen sich keinem Zeitraum
				zuordnen.
			</p>
		{/if}
		{#if buch.aus_katalog_geloescht > 0}
			<p class="text-sm text-on-surface-variant border-t border-outline-variant pt-3">
				{buch.aus_katalog_geloescht} Exemplare wurden in diesem Zeitraum aus dem Katalog gelöscht, statt
				ausgesondert zu werden — mit ihrem Titel oder als endgültig entfernter Verlust. Titel, Signatur
				und Abgangsgrund sind mit ihnen gelöscht worden; sie stehen deshalb in keiner Liste oben. Ihre
				Nummern sind in den System-Logs nachschlagbar.
			</p>
		{/if}
		{#if abschnitte.some((/** @type {any} */ a) => a.topf === '' && a.zeilen.length > 0)}
			<p class="text-sm text-on-surface-variant border-t border-outline-variant pt-3">
				Zu den Exemplaren unter „ohne Zuordnung" ist keine Bestellung hinterlegt — Altbestand,
				Handanlage oder Bestandskorrektur. Aus welchen Mitteln sie bezahlt wurden, ist deshalb nicht
				belegt, und ihr Zugangsdatum ist der Tag, an dem sie im Programm angelegt wurden.
			</p>
		{/if}
	{/if}
</div>

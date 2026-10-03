<!-- @component Bestandsbuch — Zugangs- und Abgangsbuch, ein Bauteil.

     Beide Bücher sind derselbe Nachweis in zwei Richtungen: Zeitraum wählen, Liste je Topf,
     Blatt zum Abheften. Zwei Bauteile wären zwei Orte für dieselben Entscheidungen.

     Abschnitte und Überschriften kommen fertig vom Server, damit Blatt und Bildschirm
     denselben Topf mit demselben Wort nennen. Der Zeitraum ebenso: Das erste Laden fragt ohne
     Datum, der Server antwortet mit dem laufenden Schulhalbjahr (Stichtage 15.3./15.9.), und
     die Felder zeigen danach, was er gelesen hat. -->
<script>
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import Abschnitt from '../ui/Abschnitt.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import FilterChips from '../ui/FilterChips.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { formatDatum, formatZahl } from '../../utils/format.js';
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

	/** @typedef {{ topf: string, titel: string, zeilen: any[] }} Topf */
	const abschnitte = $derived(/** @type {Topf[]} */ (buch?.abschnitte ?? []));

	// Die Töpfe stehen als Felder mit ihrer Zahl unter dem Zeitraum. Ein leerer Topf belegt
	// unten keinen Abschnitt: Dass nichts kam oder ging, sagt die Null im Feld. Ein Klick zeigt
	// einen Topf allein; der Ausdruck bleibt das ganze Buch. Das Feld trägt den Namen des Topfs
	// wie Überschrift und Blatt, auch wo er länger ist als die 20 Zeichen, die M3 für Chips nennt.
	let nurTopf = $state(/** @type {string | null} */ (null));
	const felder = $derived(
		abschnitte.map((a) => ({ wert: a.topf, text: `${a.titel} · ${formatZahl(a.zeilen.length)}` }))
	);
	// „Ohne Zuordnung" fehlt in einem Zeitraum ohne solche Zeilen; die Wahl darauf gilt dann nicht.
	const wahl = $derived(abschnitte.some((a) => a.topf === nurTopf) ? nurTopf : null);
	const gezeigt = $derived(
		abschnitte.filter((a) => (wahl === null ? a.zeilen.length > 0 : a.topf === wahl))
	);
	// Eine Liste mit Zeilen bleibt im Dokument und wird nur ausgeblendet: Zehntausende Zeilen
	// neu aufzubauen dauert Sekunden, in denen der Klick ohne Antwort bliebe.
	const gebaut = $derived(abschnitte.filter((a) => a.zeilen.length > 0 || a.topf === wahl));

	// Der Ausdruck nimmt den geladenen Zeitraum, nicht den in den Eingabefeldern: Wer ein Datum
	// ändert und ohne „Anzeigen" druckt, heftete sonst ein Blatt ab, dessen Zeitraum nie auf dem
	// Bildschirm stand. Vor der ersten Antwort bleibt die Adresse ohne Zeitraum; der Server
	// nimmt dann das laufende Halbjahr.
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

{#snippet nichts()}
	<p class="text-sm text-on-surface-variant">Kein {wortSingular} in diesem Zeitraum.</p>
{/snippet}

<div class="py-6 space-y-6">
	<div class="space-y-4">
		<div class="flex flex-wrap items-end gap-4">
			<Feld bind:value={von} label="Von" type="date" class="w-44" />
			<Feld bind:value={bis} label="Bis" type="date" class="w-44" />
			<Button variant="secondary" onclick={laden} disabled={laeuft}>Anzeigen</Button>
			<!-- Ein Link, kein Knopf: Er führt zum Blatt und lässt sich in einem neuen Reiter öffnen.
			     Die Form ist die von ui/Button. -->
			<a
				href={druckAdresse}
				class="m3-state ml-auto inline-flex h-9 items-center gap-2 rounded-full bg-primary px-4 text-sm font-semibold text-on-primary"
			>
				<Printer class="h-4 w-4 shrink-0" aria-hidden="true" />
				Ausdrucken
			</a>
		</div>
		{#if !laeuft && !fehler && abschnitte.length > 0}
			<FilterChips
				etikett="Nach Mittelherkunft filtern"
				optionen={felder}
				wert={wahl}
				onwahl={(topf) => (nurTopf = topf)}
			/>
		{/if}
	</div>

	{#if laeuft}
		<Ladekreis />
	{:else if fehler}
		<LadeFehler titel="{buchname} nicht geladen" text={fehler} onerneut={laden} />
	{:else if buch}
		{#each gebaut as abschnitt (abschnitt.topf)}
			{@const anzahl = abschnitt.zeilen.length}
			<section hidden={!gezeigt.includes(abschnitt)}>
				<Abschnitt
					titel="{abschnitt.titel} · {formatZahl(anzahl)} {anzahl === 1 ? 'Exemplar' : 'Exemplare'}"
				/>
				{#if anzahl === 0}
					{@render nichts()}
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
		{#if gezeigt.length === 0}
			{@render nichts()}
		{/if}

		<!-- Was nicht auf der Liste steht, gehört darunter, sonst behauptet der Nachweis eine
		     Vollständigkeit, die er nicht hat. Das Feld des jeweils anderen Buchs fehlt in der
		     Antwort, und `undefined > 0` ist falsch: Der Hinweis bleibt dann weg. -->
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
		<!-- Der Satz erklärt die Liste „ohne Zuordnung"; ist sie nicht zu sehen, erklärt er nichts. -->
		{#if gezeigt.some((a) => a.topf === '' && a.zeilen.length > 0)}
			<p class="text-sm text-on-surface-variant border-t border-outline-variant pt-3">
				Zu den Exemplaren unter „ohne Zuordnung" ist keine Bestellung hinterlegt — Altbestand,
				Handanlage oder Bestandskorrektur. Aus welchen Mitteln sie bezahlt wurden, ist deshalb nicht
				belegt, und ihr Zugangsdatum ist der Tag, an dem sie im Programm angelegt wurden.
			</p>
		{/if}
	{/if}
</div>

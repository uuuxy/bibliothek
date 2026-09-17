<!-- @component Abgangsbuch — welche Exemplare in einem Zeitraum aus dem Bestand gegangen sind.

     Punkt 1 des Protokolls vom 16.09.2026 („Zugangs- und Abgangsbuch fehlen"). Das Blatt
     wird zum Stichtag ausgedruckt und abgeheftet; der Bildschirm ist die Vorschau darauf.

     Zwei Abschnitte, einer je Topf: Die Finanzen von Land und Schulträger werden getrennt
     geführt, und wer die Zahlen aus einer gemischten Liste von Hand zieht, zieht sie jedes
     Halbjahr neu — und anders.

     Der Zeitraum kommt beim ersten Laden VOM SERVER (laufendes Halbjahr, Stichtage 15.3.
     und 15.9.). Ihn hier zu rechnen wäre die zweite Auslegung von „laufendes Halbjahr" —
     und der Ausdruck deckte am Ende einen anderen Zeitraum ab als die Liste davor. -->
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

	let von = $state('');
	let bis = $state('');
	let buch = $state(/** @type {any} */ (null));
	let laeuft = $state(true);
	let fehler = $state('');

	// Die Abschnitte kommen FERTIG vom Server, samt Überschrift. Hier selbst zu gruppieren
	// war der erste Entwurf — und ließ Blatt und Bildschirm sofort auseinanderlaufen: Der
	// Ausdruck schrieb „Lernmittelfreiheit (Land)", die Oberfläche „Lernmittel (Land)".
	const abschnitte = $derived(buch?.abschnitte ?? []);

	const druckAdresse = $derived(
		`/api/bestand/abgangsbuch/pdf?von=${encodeURIComponent(von)}&bis=${encodeURIComponent(bis)}`
	);

	async function laden() {
		laeuft = true;
		fehler = '';
		try {
			const adresse =
				von && bis
					? `/api/bestand/abgangsbuch?von=${encodeURIComponent(von)}&bis=${encodeURIComponent(bis)}`
					: '/api/bestand/abgangsbuch';
			const res = await apiFetch(adresse);
			if (!res.ok) {
				const daten = await res.json().catch(() => ({}));
				fehler = daten.error || 'Das Abgangsbuch konnte nicht geladen werden.';
				return;
			}
			buch = await res.json();
			// Die Felder tragen danach, was der Server tatsächlich gelesen hat — nicht das,
			// was jemand eingetippt hat. Sonst behauptet die Überschrift einen Zeitraum, den
			// die Liste darunter nicht abdeckt.
			von = String(buch.von).slice(0, 10);
			bis = String(buch.bis).slice(0, 10);
		} catch (e) {
			fehler = 'Das Abgangsbuch konnte nicht geladen werden.';
			console.error('Abgangsbuch:', e);
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
		<LadeFehler titel="Abgangsbuch nicht geladen" text={fehler} onerneut={laden} />
	{:else if buch}
		{#each abschnitte as abschnitt (abschnitt.topf)}
			<section class="space-y-2">
				<h3 class="text-sm font-semibold text-on-surface">
					{abschnitt.titel} · {abschnitt.zeilen.length}
					{abschnitt.zeilen.length === 1 ? 'Exemplar' : 'Exemplare'}
				</h3>
				{#if abschnitt.zeilen.length === 0}
					<p class="text-sm text-on-surface-variant">Kein Abgang in diesem Zeitraum.</p>
				{:else}
					<Tabelle beschriftung="Abgänge — {abschnitt.titel}">
						<thead>
							<tr>
								<th>Abgang</th>
								<th>Nummer</th>
								<th>Titel</th>
								<th>Signatur</th>
								<th>Grund</th>
							</tr>
						</thead>
						<tbody>
							{#each abschnitt.zeilen as zeile (zeile.barcode)}
								<tr>
									<td class="whitespace-nowrap">{formatDatum(zeile.datum)}</td>
									<td class="whitespace-nowrap font-mono">{zeile.barcode}</td>
									<td>{zeile.titel}</td>
									<td class="whitespace-nowrap">{zeile.signatur}</td>
									<td class="whitespace-nowrap">{zeile.grund_text}</td>
								</tr>
							{/each}
						</tbody>
					</Tabelle>
				{/if}
			</section>
		{/each}

		<!-- Die Zahl gehört hierhin, nicht weggelassen: Abgänge ohne Datum stehen in KEINER
		     Halbjahresliste. Ein Nachweis, der das verschweigt, behauptet Vollständigkeit. -->
		{#if buch.ohne_zeitpunkt > 0}
			<p class="text-sm text-on-surface-variant border-t border-outline-variant pt-3">
				{buch.ohne_zeitpunkt} weitere Exemplare sind ausgesondert, ohne dass ein Abgangsdatum bekannt
				ist. Sie wurden vor der Einführung des Abgangsbuchs ausgebucht und lassen sich keinem Zeitraum
				zuordnen.
			</p>
		{/if}
	{/if}
</div>

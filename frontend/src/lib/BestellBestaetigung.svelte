<script>
	// Die Seite, die der Lieferant über den Link aus der Bestellmail öffnet.
	//
	// Sie ist die einzige Oberfläche des Systems ohne Anmeldung, die etwas verändert —
	// und sie zeigt deshalb bewusst wenig: eine Bestellung, ihre Etiketten, ein Knopf.
	// Der Token steht in der Adresse; einen Login gibt es hier nicht und soll es nicht
	// geben, sonst müsste der Lieferant ein Geheimnis verwalten.
	import { apiFetch } from './apiFetch.js';
	import Tabelle from './components/ui/Tabelle.svelte';
	import Button from './components/ui/Button.svelte';
	import BestaetigungEtiketten from './components/bestellungen/BestaetigungEtiketten.svelte';

	const token = window.location.pathname.replace(/^\/bestellung\//, '').replace(/\/+$/, '');

	/** @type {'laedt' | 'bereit' | 'ungueltig'} */
	let zustand = $state('laedt');
	/** @type {any} */
	let bestellung = $state(null);
	let fehler = $state('');
	let sendet = $state(false);
	// Welche Größe der Lieferant geöffnet hat. Reine Notiz für die Historie der Schule —
	// gebraucht wird sie dort nicht, weggeworfen wäre sie aber schade.
	let geoeffneteGroesse = $state('');
	// Das Bogenraster für die KLEINEN Etiketten. Der Lieferant druckt auf sein eigenes
	// Material, und davon gibt es verschiedene Rastergrößen — bis 06.08.2026 kam der
	// Bogen immer im Zweckform-Raster, und wer andere Bögen im Drucker hatte, bekam
	// einen Ausdruck, der danebenliegt.
	//
	// Die Auswahl kommt aus der Antwort des Servers (etiketten_formate), nicht aus einer
	// Liste hier: Zwei Listen über dieselben Etikettenbögen laufen auseinander, sobald
	// eine Seite ein Format ergänzt.
	let formatId = $state('');

	async function laden() {
		try {
			const res = await apiFetch(`/api/public/bestellung/${encodeURIComponent(token)}`);
			if (!res.ok) {
				zustand = 'ungueltig';
				return;
			}
			bestellung = await res.json();
			formatId ||= bestellung.etiketten_format_vorgabe ?? '';
			zustand = 'bereit';
		} catch {
			zustand = 'ungueltig';
		}
	}
	laden();

	async function bestaetigen() {
		sendet = true;
		fehler = '';
		try {
			const res = await apiFetch(
				`/api/public/bestellung/${encodeURIComponent(token)}/bestaetigen`,
				{
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						etiketten_groesse: geoeffneteGroesse,
						// Nur bei 'klein' aussagekräftig — sonst leer, damit in der Historie
						// der Schule kein Raster steht, das gar nicht gedruckt wurde.
						etiketten_format: geoeffneteGroesse === 'klein' ? formatId : ''
					})
				}
			);
			if (res.ok) {
				// Neu laden statt lokal umschalten: Bestätigt hier jemand zweimal (zwei
				// Tabs, zwei Personen im selben Postfach), zeigt die Seite danach den
				// echten Zustand aus der Datenbank und keine erfundene Quittung.
				await laden();
				return;
			}
			if (res.status === 409) {
				await laden();
				return;
			}
			const daten = await res.json().catch(() => ({}));
			fehler = daten.error || 'Die Bestätigung konnte nicht gespeichert werden.';
		} catch {
			fehler = 'Die Bestätigung konnte nicht gespeichert werden.';
		} finally {
			sendet = false;
		}
	}

	// Topf · Lieferant · Kundennummer · Anzahl — leere Angaben fallen samt Trenner weg.
	// Der Topf steht vorn: Der Händler bekommt am selben Tag zwei gleich aussehende Links.
	let kopfzeile = $derived(
		[
			bestellung?.mittel,
			bestellung?.lieferant_name,
			bestellung?.kundennummer ? `Kundennummer ${bestellung.kundennummer}` : null,
			`${bestellung?.anzahl_exemplare} Exemplare`
		]
			.filter(Boolean)
			.join(' · ')
	);

	// Die ISBN-Spalte erscheint nur, wenn wenigstens eine Position eine trägt. Sonst stand
	// dort eine leere Spalte, und auf dem Handy klebten die Überschriften als „ISBNMenge"
	// aneinander. Ein Lieferant öffnet so einen Link oft am Telefon.
	let zeigeISBN = $derived(bestellung?.positionen?.some((/** @type {any} */ p) => p.isbn) ?? false);

	/** @param {string} iso */
	function datum(iso) {
		return new Date(iso).toLocaleDateString('de-DE', {
			day: '2-digit',
			month: 'long',
			year: 'numeric'
		});
	}
</script>

<main class="bg-surface min-h-screen px-4 py-10">
	<div class="mx-auto max-w-2xl space-y-6">
		{#if zustand === 'laedt'}
			<p class="text-on-surface-variant text-center">Bestellung wird geladen …</p>
		{:else if zustand === 'ungueltig'}
			<div class="bg-surface-container-lowest rounded-xl p-8 text-center shadow-sm">
				<h1 class="text-on-surface text-lg">Dieser Link ist nicht mehr gültig</h1>
				<p class="text-on-surface-variant mt-2 text-sm">
					Bestätigungs-Links laufen nach einiger Zeit ab und gehören immer zu genau einer
					Bestellung. Bitte wenden Sie sich an die Schulbibliothek, wenn Sie einen neuen benötigen.
				</p>
			</div>
		{:else}
			<div class="bg-surface-container-lowest rounded-xl p-8 shadow-sm">
				<p class="text-on-surface-variant text-xs font-medium">
					{bestellung.schule_name || 'Schulbibliothek'}
				</p>
				{#if bestellung.schule_anschrift}
					<p class="text-on-surface-variant text-xs">{bestellung.schule_anschrift}</p>
				{/if}
				<h1 class="text-on-surface mt-2 text-xl">
					Bestellung vom {datum(bestellung.bestelldatum)}
				</h1>
				<!-- Eine Zeichenkette statt zusammengesetzter Markup-Schnipsel: Zwischen
				     {#if}-Blöcken verschluckt der Formatierer die Leerzeichen, und im Browser
				     stand „Naacher· Kundennummer". -->
				<p class="text-on-surface-variant mt-1 text-sm">{kopfzeile}</p>
				<Tabelle beschriftung="Bestellte Titel" class="mt-6">
					<thead>
						<tr>
							<th>Titel</th>
							{#if zeigeISBN}<th>ISBN</th>{/if}
							<th class="text-right">Menge</th>
						</tr>
					</thead>
					<tbody>
						{#each bestellung.positionen as p (p.titel_name + p.isbn)}
							<tr>
								<td class="font-medium">{p.titel_name}</td>
								{#if zeigeISBN}<td>{p.isbn}</td>{/if}
								<td class="text-right">{p.menge}</td>
							</tr>
						{/each}
					</tbody>
				</Tabelle>
			</div>

			{#if bestellung.etiketten_vorhanden}
				<BestaetigungEtiketten {bestellung} {token} bind:formatId bind:geoeffneteGroesse />
			{/if}

			<div class="bg-surface-container-lowest rounded-xl p-8 shadow-sm">
				{#if bestellung.bestaetigt_am}
					<h2 class="text-primary text-base font-medium">Bestellung bestätigt</h2>
					<p class="text-on-surface-variant mt-1 text-sm">
						Eingegangen am {datum(bestellung.bestaetigt_am)}. Die Schulbibliothek sieht die
						Bestätigung in ihrer Bestellhistorie — Sie müssen nichts weiter tun.{bestellung.link_gueltig_bis
							? ` Diese Seite und die Etiketten bleiben bis zum ${datum(bestellung.link_gueltig_bis)} erreichbar.`
							: ''}
					</p>
				{:else}
					<h2 class="text-on-surface text-base font-medium">Bestellung bestätigen</h2>
					<p class="text-on-surface-variant mt-1 text-sm">
						Damit meldet sich die Bestellung in der Schulbibliothek als von Ihnen bestätigt. Das ist
						einmal möglich.{bestellung.link_gueltig_bis
							? ` Dieser Link gilt bis zum ${datum(bestellung.link_gueltig_bis)}; danach hilft die Schulbibliothek mit einem neuen.`
							: ''}
					</p>
					{#if fehler}
						<p class="text-error mt-3 text-sm font-medium">{fehler}</p>
					{/if}
					<Button size="lg" class="mt-4" disabled={sendet} onclick={bestaetigen}>
						{sendet ? 'Wird gesendet …' : 'Bestellung jetzt bestätigen'}
					</Button>
				{/if}
			</div>
			<p class="text-on-surface-variant pb-4 text-center text-xs">
				Fragen zu dieser Bestellung? Antworten Sie einfach auf die Bestellmail der Schulbibliothek.
			</p>
		{/if}
	</div>
</main>

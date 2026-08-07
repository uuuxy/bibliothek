<script>
	// Die Seite, die der Lieferant über den Link aus der Bestellmail öffnet.
	//
	// Sie ist die einzige Oberfläche des Systems ohne Anmeldung, die etwas verändert —
	// und sie zeigt deshalb bewusst wenig: eine Bestellung, ihre Etiketten, ein Knopf.
	// Der Token steht in der Adresse; einen Login gibt es hier nicht und soll es nicht
	// geben, sonst müsste der Lieferant ein Geheimnis verwalten.
	import { apiFetch } from './apiFetch.js';
	import Button from './components/ui/Button.svelte';
	import Select from './components/ui/Select.svelte';
	import { Check } from '@lucide/svelte';

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
	$effect(() => {
		if (!formatId && bestellung?.etiketten_format_vorgabe) {
			formatId = bestellung.etiketten_format_vorgabe;
		}
	});

	async function laden() {
		try {
			const res = await apiFetch(`/api/public/bestellung/${encodeURIComponent(token)}`);
			if (!res.ok) {
				zustand = 'ungueltig';
				return;
			}
			bestellung = await res.json();
			zustand = 'bereit';
		} catch {
			zustand = 'ungueltig';
		}
	}
	laden();

	/** @param {'klein' | 'gross'} groesse */
	function etikettenOeffnen(groesse) {
		geoeffneteGroesse = groesse;
		// Das Raster gilt nur für die kleinen Etiketten. Das große Lernmittel-Etikett hat
		// ein festes Raster (4 Stück auf A4) und wird ausgeschnitten, nicht auf
		// vorgestanzte Bögen gedruckt.
		const query = groesse === 'klein' && formatId ? `?format=${encodeURIComponent(formatId)}` : '';
		window.open(
			`/api/public/bestellung/${encodeURIComponent(token)}/etiketten/${groesse}${query}`,
			'_blank',
			'noopener'
		);
	}

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

	// Lieferant · Kundennummer · Anzahl — leere Angaben fallen samt Trenner weg.
	let kopfzeile = $derived(
		[
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

<div class="min-h-screen bg-slate-50 px-4 py-10">
	<div class="mx-auto max-w-2xl space-y-6">
		{#if zustand === 'laedt'}
			<p class="text-center text-slate-400">Bestellung wird geladen …</p>
		{:else if zustand === 'ungueltig'}
			<div class="rounded-xl bg-white p-8 text-center shadow-sm">
				<h1 class="text-lg font-bold text-slate-800">Dieser Link ist nicht mehr gültig</h1>
				<p class="mt-2 text-sm text-slate-500">
					Bestätigungs-Links laufen nach einiger Zeit ab und gehören immer zu genau einer
					Bestellung. Bitte wenden Sie sich an die Schulbibliothek, wenn Sie einen neuen benötigen.
				</p>
			</div>
		{:else}
			<div class="rounded-xl bg-white p-8 shadow-sm">
				<p class="text-xs font-semibold tracking-wide text-slate-400 uppercase">
					{bestellung.schule_name || 'Schulbibliothek'}
				</p>
				{#if bestellung.schule_anschrift}
					<p class="text-xs text-slate-400">{bestellung.schule_anschrift}</p>
				{/if}
				<h1 class="mt-2 text-xl font-bold text-slate-800">
					Bestellung vom {datum(bestellung.bestelldatum)}
				</h1>
				<!-- Eine Zeichenkette statt zusammengesetzter Markup-Schnipsel: Zwischen
				     {#if}-Blöcken verschluckt der Formatierer die Leerzeichen, und im Browser
				     stand „Naacher· Kundennummer". -->
				<p class="mt-1 text-sm text-slate-500">{kopfzeile}</p>

				<table class="mt-6 w-full text-left text-sm">
					<thead>
						<tr class="border-b border-slate-200 text-xs font-semibold text-slate-500">
							<th class="py-2 pr-3">Titel</th>
							{#if zeigeISBN}<th class="py-2 pr-3">ISBN</th>{/if}
							<th class="py-2 text-right">Menge</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each bestellung.positionen as p (p.titel_name + p.isbn)}
							<tr>
								<td class="py-2 pr-3 font-medium text-slate-700">{p.titel_name}</td>
								{#if zeigeISBN}<td class="py-2 pr-3 text-slate-500">{p.isbn}</td>{/if}
								<td class="py-2 text-right text-slate-700">{p.menge}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			{#if bestellung.etiketten_vorhanden}
				<div class="rounded-xl bg-white p-8 shadow-sm">
					<h2 class="text-base font-bold text-slate-800">Etiketten drucken</h2>
					<p class="mt-1 text-sm text-slate-500">
						Beide Bögen enthalten dieselben Barcodes wie der Anhang der Bestellmail — Sie wählen nur
						das Format.
					</p>

					{#if bestellung.etiketten_formate?.length}
						<div class="mt-5 max-w-md space-y-1.5">
							<label for="etikettenformat" class="block text-xs font-medium text-slate-500">
								Bogenraster der kleinen Etiketten
							</label>
							<Select
								id="etikettenformat"
								bind:value={formatId}
								options={bestellung.etiketten_formate.map((/** @type {any} */ f) => ({
									value: f.id,
									label: f.name
								}))}
								aria-label="Bogenraster der kleinen Etiketten"
							/>
							<p class="text-xs text-slate-400">
								Passend zu den Etikettenbögen in Ihrem Drucker. Gilt nicht für die großen
								Lernmittel-Etiketten — die liegen zu viert auf einem A4-Blatt und werden
								ausgeschnitten.
							</p>
						</div>
					{/if}

					<div class="mt-4 flex flex-wrap gap-3">
						<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('klein')}>
							{#if geoeffneteGroesse === 'klein'}<Check size={16} aria-hidden="true" />{/if}
							Kleine Etiketten (Bogen A4)
						</Button>
						<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('gross')}>
							{#if geoeffneteGroesse === 'gross'}<Check size={16} aria-hidden="true" />{/if}
							Große Lernmittel-Etiketten (4 je A4-Blatt)
						</Button>
					</div>
					{#if geoeffneteGroesse}
						<p class="mt-3 text-xs text-slate-500">
							Der Bogen wurde in einem neuen Tab geöffnet. Erscheint er nicht, ist er vom Browser
							blockiert worden — dann bitte den Knopf erneut drücken.
						</p>
					{/if}
				</div>
			{/if}

			<div class="rounded-xl bg-white p-8 shadow-sm">
				{#if bestellung.bestaetigt_am}
					<h2 class="text-base font-bold text-emerald-700">Bestellung bestätigt</h2>
					<p class="mt-1 text-sm text-slate-500">
						Eingegangen am {datum(bestellung.bestaetigt_am)}. Die Schulbibliothek sieht die
						Bestätigung in ihrer Bestellhistorie — Sie müssen nichts weiter tun.
					</p>
				{:else}
					<h2 class="text-base font-bold text-slate-800">Bestellung bestätigen</h2>
					<p class="mt-1 text-sm text-slate-500">
						Damit meldet sich die Bestellung in der Schulbibliothek als von Ihnen bestätigt. Das ist
						einmal möglich.
					</p>
					{#if fehler}
						<p class="mt-3 text-sm font-medium text-rose-600">{fehler}</p>
					{/if}
					<Button size="lg" class="mt-4" disabled={sendet} onclick={bestaetigen}>
						{sendet ? 'Wird gesendet …' : 'Bestellung jetzt bestätigen'}
					</Button>
				{/if}
			</div>
			<p class="pb-4 text-center text-xs text-slate-400">
				Fragen zu dieser Bestellung? Antworten Sie einfach auf die Bestellmail der Schulbibliothek.
			</p>
		{/if}
	</div>
</div>

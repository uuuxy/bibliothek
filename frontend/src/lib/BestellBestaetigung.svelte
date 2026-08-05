<script>
	// Die Seite, die der Lieferant über den Link aus der Bestellmail öffnet.
	//
	// Sie ist die einzige Oberfläche des Systems ohne Anmeldung, die etwas verändert —
	// und sie zeigt deshalb bewusst wenig: eine Bestellung, ihre Etiketten, ein Knopf.
	// Der Token steht in der Adresse; einen Login gibt es hier nicht und soll es nicht
	// geben, sonst müsste der Lieferant ein Geheimnis verwalten.
	import { apiFetch } from './apiFetch.js';
	import Button from './components/ui/Button.svelte';

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
		window.open(
			`/api/public/bestellung/${encodeURIComponent(token)}/etiketten/${groesse}`,
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
					body: JSON.stringify({ etiketten_groesse: geoeffneteGroesse })
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
			<div class="rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm">
				<h1 class="text-lg font-bold text-slate-800">Dieser Link ist nicht mehr gültig</h1>
				<p class="mt-2 text-sm text-slate-500">
					Bestätigungs-Links laufen nach einiger Zeit ab und gehören immer zu genau einer
					Bestellung. Bitte wenden Sie sich an die Schulbibliothek, wenn Sie einen neuen benötigen.
				</p>
			</div>
		{:else}
			<div class="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
				<p class="text-xs font-semibold tracking-wide text-slate-400 uppercase">
					{bestellung.schule_name || 'Schulbibliothek'}
				</p>
				<h1 class="mt-1 text-xl font-bold text-slate-800">
					Bestellung vom {datum(bestellung.bestelldatum)}
				</h1>
				<!-- Eine Zeichenkette statt zusammengesetzter Markup-Schnipsel: Zwischen
				     {#if}-Blöcken verschluckt der Formatierer die Leerzeichen, und im Browser
				     stand „Naacher· Kundennummer". -->
				<p class="mt-1 text-sm text-slate-500">{kopfzeile}</p>

				<table class="mt-6 w-full text-left text-sm">
					<thead>
						<tr class="border-b border-slate-200 text-xs font-semibold text-slate-500">
							<th class="py-2">Titel</th>
							<th class="py-2">ISBN</th>
							<th class="py-2 text-right">Menge</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each bestellung.positionen as p (p.titel_name + p.isbn)}
							<tr>
								<td class="py-2 font-medium text-slate-700">{p.titel_name}</td>
								<td class="py-2 text-slate-500">{p.isbn}</td>
								<td class="py-2 text-right text-slate-700">{p.menge}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			{#if bestellung.etiketten_vorhanden}
				<div class="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
					<h2 class="text-base font-bold text-slate-800">Etiketten drucken</h2>
					<p class="mt-1 text-sm text-slate-500">
						Beide Bögen enthalten dieselben Barcodes wie der Anhang der Bestellmail — Sie wählen nur
						das Format.
					</p>
					<div class="mt-4 flex flex-wrap gap-3">
						<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('klein')}>
							Kleine Etiketten (Bogen A4)
						</Button>
						<Button size="lg" variant="secondary" onclick={() => etikettenOeffnen('gross')}>
							Große Lernmittel-Etiketten
						</Button>
					</div>
				</div>
			{/if}

			<div class="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
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
		{/if}
	</div>
</div>

<script>
	import { CircleCheck, Plus, Printer } from '@lucide/svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import CoverPeek from '../ui/CoverPeek.svelte';
	import BuchCover from '../ui/BuchCover.svelte';

	let { recommendations, onAddToCart } = $props();

	// Nur die ersten Einträge ins DOM (Muster wie BookTable/Inventur-Startseite).
	// Der Bestellbedarf umfasst schnell den halben Katalog — jeder Titel unter seinem
	// Meldebestand landet hier. Alle zu rendern hiess: tausende DOM-Knoten und ebenso
	// viele Cover-Requests. Das Scrollfenster wächst mit dem Viewport, damit man die
	// dringenden Titel sieht statt drei Zeilen durch ein Guckloch.
	let maxVisible = $state(60);

	// Schnellfilter: bei 335 Titeln ist Suchen schneller als Scrollen.
	let filter = $state('');

	let gefiltert = $derived(
		filter.trim()
			? recommendations.filter((/** @type {any} */ r) => {
					const q = filter.trim().toLowerCase();
					return (
						(r.titel || '').toLowerCase().includes(q) ||
						(r.isbn || '').toLowerCase().includes(q) ||
						(r.verlag || '').toLowerCase().includes(q) ||
						(r.signatur || '').toLowerCase().includes(q)
					);
				})
			: recommendations
	);

	let sichtbare = $derived(gefiltert.slice(0, maxVisible));

	// Nach einem Datenwechsel (z. B. Wareneingang) oder neuem Filter wieder von vorn.
	$effect(() => {
		// Nur gelesen, um die Abhaengigkeit herzustellen: Aendert sich die Liste oder der
		// Filter, faengt die Anzeige wieder bei 60 Eintraegen an. `void` statt des
		// Komma-Operators — den meldet die Typpruefung als wirkungslosen Ausdruck.
		void recommendations;
		void filter;
		maxVisible = 60;
	});

	/**
	 * Farbstufe einer Zeile. Die Liste ist bereits die Bestellbedarf-Liste (Backend:
	 * gesamt < konfigurierbare Schwelle). Herausgehoben wird nur der echte Notfall:
	 * 0 eigene Exemplare (Titel komplett weg) = kritisch, alles andere = knapp. Basis ist
	 * der Gesamtbestand (Besitz), nicht der Verfügbarbestand — ein verliehener Klassensatz
	 * (0 verfügbar, 30 vorhanden) taucht hier ohnehin nicht auf.
	 * @param {any} r
	 */
	function stufe(r) {
		return r.gesamt_bestand === 0 ? 'kritisch' : 'knapp';
	}

	let kritischeAnzahl = $derived(
		recommendations.filter((/** @type {any} */ r) => stufe(r) === 'kritisch').length
	);
</script>

<!-- Kein Kartenrahmen: Der Bestellbedarf ist die Arbeitsflaeche der Seite, kein Objekt
     darauf. Die Kopfzeile trennt sich ueber ihre Linie vom Listenkoerper, die Spalte
     daneben ueber die senkrechte Linie in BestellWorkspace. -->
<section class="flex min-w-0 flex-col">
	<!-- Header -->
	<header class="border-b border-slate-200 pb-4">
		<div class="flex items-start justify-between gap-4">
			<div class="min-w-0">
				<h2 class="text-lg font-bold text-slate-900 tracking-tight flex items-center gap-2">
					Bestellbedarf
					{#if recommendations.length}
						<span
							class="text-xs font-bold text-slate-500 bg-slate-100 rounded-full px-2 py-0.5 tabular-nums"
							>{recommendations.length}</span
						>
					{/if}
				</h2>
				{#if kritischeAnzahl}
					<!-- Eine Aussage, nicht zwei. Vorher stand hier „{'{n}'}× komplett fehlend · 0 Exemplare":
					     Beides beschreibt DENSELBEN Zustand (gesamt_bestand === 0), las sich aber wie zwei
					     verschiedene Kennzahlen — und die „0" war eine feste Null im Markup, kein Messwert.
					     Der Bezug auf die Gesamtzahl sagt stattdessen etwas Neues: wie groß der Notfall
					     innerhalb der Liste ist. -->
					<!-- Bewusst NICHT in Fehlerrot. Bei einer Lernmittel-Bedarfsliste ist „kein
					     Exemplar vorhanden" der Normalfall — 243 von 327 Titeln. Eine Zahl, die
					     fast immer gilt, ist keine Fehlermeldung; sie in der Error-Rolle zu
					     faerben erzieht das Auge dazu, Rot zu ueberlesen, und dann verschwindet
					     der eine echte Fehler darin. In M3 traegt die Error-Rolle Zustaende, die
					     korrigiert werden MUESSEN. Das hier ist eine Kennzahl. -->
					<p class="text-sm text-slate-500 mt-1">
						{kritischeAnzahl} von {recommendations.length} Titeln ohne ein einziges Exemplar
					</p>
				{:else if recommendations.length}
					<p class="text-sm text-slate-400 mt-1">Alle unter der Bestellbedarf-Schwelle.</p>
				{/if}
			</div>
			<a
				href="/api/bestellungen/pdf"
				download
				class="shrink-0 flex items-center gap-2 text-xs font-bold text-slate-600 bg-slate-50 hover:bg-slate-100 border border-slate-200 px-3 py-2 rounded-xl transition-colors"
			>
				<Printer class="h-4 w-4 shrink-0" aria-hidden="true" />
				<span class="hidden sm:inline">PDF-Bestellliste</span>
			</a>
		</div>

		{#if recommendations.length}
			<Suchpille
				id="bestellbedarf-suchfeld"
				bind:wert={filter}
				platzhalter="In {recommendations.length} Titeln filtern …"
				etikett="Bestellvorschläge filtern"
			/>
		{/if}
	</header>

	<!-- List -->
	{#if !recommendations.length}
		<div class="flex flex-col items-center justify-center text-center py-16 px-6 text-slate-400">
			<CircleCheck class="h-5 w-5" aria-hidden="true" />
			<p class="text-sm font-semibold text-slate-500">Bestände ausreichend</p>
			<p class="text-xs mt-1">Kein Titel liegt unter der Bestellbedarf-Schwelle.</p>
		</div>
	{:else if !gefiltert.length}
		<div class="text-center py-14 px-6 text-slate-400">
			<p class="text-sm font-medium">Kein Treffer für <em>„{filter}"</em></p>
		</div>
	{:else}
		<!-- Spaltenkopf. Die Zahlenspalte war bisher NUR über ein title-Attribut erklärt — ein
		     Hover-Tooltip, der beim ersten Hinsehen unsichtbar ist und auf Tablets gar nicht
		     erreichbar. Auf 332 Zeilen standen damit unbeschriftete Zahlen.
		     Der Kopf steht AUSSERHALB des Scroll-Containers: Die Liste scrollt in sich selbst,
		     die Beschriftung bleibt deshalb stehen, ohne sticky und ohne z-index-Fragen.
		     Die Beschriftung ist breiter als die Zahlen darunter — beide enden aber an
		     derselben Kante, weil auf beide dieselbe Lücke und die 36-px-Knopfspalte folgen.
		     Das ist die übliche Ausrichtung einer Zahlenspalte und kostet den Titeln keine
		     Breite, was eine feste Spaltenbreite getan hätte. -->
		<!-- Der Kopf UEBERNIMMT die Geometrie einer Zeile (-mx-3, transparenter Rahmen,
		     px-3, gap-3), statt sie mit eigenen Werten nachzubauen. Vorher war der
		     Platzhalter links w-4, der Cover-Knopf in der Zeile aber 32 px breit — „Titel"
		     stand damit 17 px links neben den Titeln. Unter dem Kartenrahmen fiel das nicht
		     auf, auf der flachen Flaeche sofort. -->
		<div class="-mx-3 border-b border-slate-100">
			<div
				class="flex items-center gap-3 border border-transparent px-3 py-2 text-label-small font-semibold uppercase tracking-wider text-slate-400 select-none"
			>
				<span class="w-8 shrink-0" aria-hidden="true"></span>
				<span class="flex-1 min-w-0">Titel</span>
				<span class="text-right">Verfügbar / Bestand</span>
				<span class="w-9 shrink-0" aria-hidden="true"></span>
			</div>
		</div>

		<!-- -mx-3 zieht den Zeilencontainer um genau das px-3 der Zeilen nach aussen: Der
		     Zeilentext steht damit auf derselben Kante wie Ueberschrift und Spaltenkopf,
		     waehrend die Hover-Flaeche weiterhin ueber den Text hinausreicht. Ohne den
		     Kartenrahmen faellt eine Fehlausrichtung von 12 px sofort auf. -->
		<div class="overflow-y-auto max-h-[calc(100vh-19rem)] -mx-3 py-3 space-y-1.5">
			{#each sichtbare as r, _i (_i)}
				<div
					class="group flex items-center gap-3 rounded-xl border border-transparent px-3 py-2 hover:bg-slate-50 hover:border-slate-200 transition-colors"
				>
					<!-- Cover IN der Zeile, zugleich Auslöser der Großansicht (so sieht CoverPeek es
					     über `children` vor). Frühere Gegengründe gemessen widerlegt: `loading="lazy"`
					     erspart die 247 Requests, 5.724 von 8.706 Titeln ohne Exemplar haben ein Cover. -->
					<CoverPeek isbn={r.isbn || ''} coverUrl={r.cover_url || ''} titel={r.titel}>
						<BuchCover coverUrl={r.cover_url || ''} isbn={r.isbn || ''} titel={r.titel} />
					</CoverPeek>

					<div class="min-w-0 flex-1">
						<h4 class="font-semibold text-slate-900 text-sm truncate leading-snug">{r.titel}</h4>
						<p class="text-xs text-slate-500 truncate">
							{#if r.isbn}<span class="font-mono text-slate-400">{r.isbn}</span>{/if}
							{#if r.verlag}<span class="mx-1.5 text-slate-400">·</span>{r.verlag}{/if}
							{#if r.signatur}<span class="mx-1.5 text-slate-400">·</span>{r.signatur}{/if}
						</p>
					</div>

					<!-- Durchgehend dieselbe Bestandsspalte statt eines Pills im Null-Fall. Das
					     „Fehlt komplett"-Pill stand auf 252 von 334 Zeilen — ein Signal, das auf drei
					     Vierteln der Liste steht, markiert den Normalzustand statt der Ausnahme.

					     Dieselbe Ueberlegung gilt fuer die FARBE, und dort stand sie noch aus: Rot
					     fuer gesamt_bestand === 0 traf 243 von 327 Zeilen. In M3 traegt die
					     Error-Rolle Zustaende, die korrigiert werden muessen — in einer Liste, die
					     ausschliesslich Bedarf enthaelt, ist Bedarf kein Fehler. Die Zugehoerigkeit
					     zur Liste IST das Signal; innerhalb der Liste rangiert jetzt die BETONUNG
					     (on-surface gegen on-surface-variant) statt einer zweiten Alarmfarbe. -->
					<div
						class="text-right shrink-0 leading-tight text-sm font-bold tabular-nums {r.gesamt_bestand ===
						0
							? 'text-slate-900'
							: 'text-slate-500'}"
						title="verfügbar / im Bestand"
					>
						{r.verfuegbarer_bestand}<span class="text-slate-400 font-medium">/</span
						>{r.gesamt_bestand}
					</div>

					<button
						onclick={() => onAddToCart(r)}
						aria-label="{r.titel} zur Bestellung hinzufügen"
						data-tip="Zur Bestellung hinzufügen"
						class="shrink-0 w-9 h-9 rounded-full border border-slate-200 text-slate-400 flex items-center justify-center hover:border-blue-500 hover:text-white hover:bg-blue-600 active:scale-90 transition-all cursor-pointer"
					>
						<Plus class="w-4 h-4" aria-hidden="true" />
					</button>
				</div>
			{/each}

			{#if gefiltert.length > maxVisible}
				<button
					onclick={() => (maxVisible += 60)}
					class="w-full py-2.5 text-xs font-bold text-slate-500 hover:text-slate-800 hover:bg-slate-50 rounded-xl transition-colors cursor-pointer"
				>
					Weitere {gefiltert.length - maxVisible} anzeigen
				</button>
			{/if}
		</div>
	{/if}
</section>

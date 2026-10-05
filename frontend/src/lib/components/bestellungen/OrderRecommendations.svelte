<script>
	import { CircleCheck, Printer } from '@lucide/svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import Button from '../ui/Button.svelte';
	import BedarfZeile from './BedarfZeile.svelte';
	import NeueAuflageDialog from './NeueAuflageDialog.svelte';
	import { filtereBedarf } from './bedarfFilter.js';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	let { recommendations, onAddToCart } = $props();

	// „Neue Auflage bestellen" (docs/OFFEN.md 4.18, Stufe 4): die Zeile, deren Dialog offen ist.
	/** @type {any | null} */
	let neueAuflageFuer = $state(null);

	// Nur die ersten Einträge ins DOM (Muster wie BookTable/Inventur-Startseite).
	// Der Bestellbedarf umfasst schnell den halben Katalog — jeder Titel unter seinem
	// Meldebestand landet hier. Alle zu rendern hiess: tausende DOM-Knoten und ebenso
	// viele Cover-Requests. Das Scrollfenster wächst mit dem Viewport, damit man die
	// dringenden Titel sieht statt drei Zeilen durch ein Guckloch.
	let maxVisible = $state(60);

	// Schnellfilter: bei 335 Titeln ist Suchen schneller als Scrollen.
	let filter = $state('');

	let gefiltert = $derived(filtereBedarf(recommendations, filter));

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
	<header class="border-b border-outline-variant pb-4">
		<div class="flex items-start justify-between gap-4">
			<div class="min-w-0">
				<h2 class="text-lg font-bold text-on-surface tracking-tight flex items-center gap-2">
					Bestellbedarf
					{#if recommendations.length}
						<span
							class="text-xs font-bold text-on-surface-variant bg-surface-container rounded-full px-2 py-0.5 tabular-nums"
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
					<p class="text-sm text-on-surface-variant mt-1">
						{kritischeAnzahl} von {recommendations.length} Titeln ohne ein einziges Exemplar
					</p>
				{:else if recommendations.length}
					<p class="text-sm text-on-surface-variant mt-1">Alle unter der Bestellbedarf-Schwelle.</p>
				{/if}
			</div>
			<a
				href="/api/bestellungen/pdf"
				download
				class="m3-state shrink-0 flex items-center gap-2 text-xs font-bold text-on-surface-variant bg-surface-container-low border border-outline-variant px-3 py-2 rounded-xl"
			>
				<Printer class="h-4 w-4 shrink-0" aria-hidden="true" />
				<span class="hidden sm:inline">PDF-Bestellliste</span>
			</a>
		</div>

		{#if recommendations.length}
			<Suchpille
				kamera
				id="bestellbedarf-suchfeld"
				bind:wert={filter}
				platzhalter="In {recommendations.length} Titeln filtern …"
				etikett="Bestellvorschläge filtern"
			/>
		{/if}
	</header>

	<!-- List -->
	{#if !recommendations.length}
		<div
			class="flex flex-col items-center justify-center text-center py-16 px-6 text-on-surface-variant"
		>
			<CircleCheck class="h-5 w-5" aria-hidden="true" />
			<p class="text-sm font-semibold">Bestände ausreichend</p>
			<p class="text-xs mt-1">Kein Titel liegt unter der Bestellbedarf-Schwelle.</p>
		</div>
	{:else if !gefiltert.length}
		<div class="text-center py-14 px-6 text-on-surface-variant">
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
		<div class="-mx-3 border-b border-outline-variant">
			<div
				class="flex items-center gap-3 border border-transparent px-3 py-2 text-label-small font-semibold uppercase tracking-wider text-on-surface-variant select-none"
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
				<BedarfZeile {r} {onAddToCart} onNeueAuflage={(z) => (neueAuflageFuer = z)} />
			{/each}

			{#if gefiltert.length > maxVisible}
				<!-- Hausbauteil statt rohem <button> mit text-xs (12 px kennt M3 fuer keinen Knopf).
				     typo-rollen sah ihn nie: erscheint erst ab 61 Titeln, auf dem Zielsystem (247) immer. -->
				<Button variant="ghost" class="w-full" onclick={() => (maxVisible += 60)}>
					Weitere {gefiltert.length - maxVisible} anzeigen
				</Button>
			{/if}
		</div>
	{/if}

	<!-- Nach dem Zuordnen ist die Zeile ein Buch mit zwei Auflagen: neu laden, damit sie so dasteht. -->
	<NeueAuflageDialog
		zeile={neueAuflageFuer}
		onschliessen={() => (neueAuflageFuer = null)}
		onbestellt={(titel) => {
			neueAuflageFuer = null;
			onAddToCart(titel);
			orderStore.loadRecommendations();
		}}
	/>
</section>

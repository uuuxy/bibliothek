<!-- @component Der Bestellbedarf als letzte Gruppe der Bestellung: die Schulbücher, deren Bestand
     nicht für das nächste Schuljahr reicht und die noch nicht in der Bestellung stehen. Wie
     „Für später gespeichert" unter den Artikeln im Warenkorb des Google Store; was hinzugefügt
     ist, steht oben mit der Marke „Bestellbedarf N" (bedarfFilter.js). -->
<script>
	import Icon from '../ui/Icon.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import { fokusAuffangen } from '../ui/fokusAuffangen.svelte.js';
	import BedarfZeile, { STELLEN, SPALTEN } from './BedarfZeile.svelte';
	import NeueAuflageDialog from './NeueAuflageDialog.svelte';
	import { offenerBedarf } from './bedarfFilter.js';
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import { anzahlText } from '../../utils/format.js';
	let { vorschlaege, onAddToCart } = $props();
	// Kam der Bedarf nicht, steht der Grund statt der leeren Liste da: „Kein Schulbuch fehlt" hieße,
	// es sei nichts zu bestellen. Scheitert nur das Auffrischen, bleibt die Liste von vorhin stehen.
	const ladefehler = $derived(bestellStore.nichtGeladen.bedarf);
	// Ohne Eintrag steht die erste Antwort noch aus; ein Leerzustand wäre dann geraten.
	const laedt = $derived(ladefehler === undefined);
	// Worauf die Rechnung fußt: Daran unterscheidet die leere Liste „nichts fehlt" von „noch kein
	// Schulbuch ausgegeben" und „ausgeschaltet" (internal/api/bestellung/bestellbedarf.go).
	const grundlage = $derived(bestellStore.bedarfGrundlage);
	const grundlageText = $derived(
		`gerechnet mit den Schulbüchern von ${grundlage.kinder_mit_schulbuch} der ${grundlage.kinder} Kinder`
	);

	const offene = $derived(offenerBedarf(vorschlaege, bestellStore.cart));
	const schonDrin = $derived(vorschlaege.length - offene.length);
	const fehlenSumme = $derived(
		offene.reduce((/** @type {number} */ s, /** @type {any} */ r) => s + (r.fehlen || 0), 0)
	);
	const kopfText = $derived(
		(schonDrin
			? anzahlText(offene.length, 'weiterer Titel', 'weitere Titel')
			: anzahlText(offene.length, 'Titel', 'Titel')) +
			` · ${anzahlText(fehlenSumme, 'Exemplar fehlt', 'Exemplare fehlen')}` +
			`${schonDrin ? ' noch' : ''} für das nächste Schuljahr`
	);

	// „Neue Auflage bestellen": die Zeile, deren Dialog offen ist.
	/** @type {any | null} */
	let neueAuflageFuer = $state(null);

	// Nur die ersten Einträge ins DOM (Muster wie Bestandstabelle/Inventur-Startseite).
	let maxVisible = $state(60);
	let sichtbare = $derived(offene.slice(0, maxVisible));

	// Nach einem Datenwechsel (z. B. Wareneingang) wieder von vorn. Nur gelesen, um die
	// Abhängigkeit herzustellen; `void` statt des Komma-Operators, den die Typprüfung als
	// wirkungslosen Ausdruck meldet.
	$effect(() => {
		void vorschlaege;
		maxVisible = 60;
	});

	// Ein hinzugefügter Titel verlässt die Liste samt dem Knopf, auf dem der Fokus stand: Er rückt
	// auf „hinzufügen" der nächsten Zeile, ohne Zeile auf den Anfang des Inhalts.
	/** @type {HTMLElement | undefined} */
	let wurzel = $state();
	fokusAuffangen(
		() => offene,
		() => wurzel,
		() => wurzel?.querySelector('[data-hinzufuegen]') ?? document.getElementById('hauptinhalt')
	);
</script>

{#snippet pdf()}
	<Button href="/api/bestellungen/pdf" download variant="secondary" class="shrink-0">
		<Icon name="print" />
		<span class="hidden sm:inline">PDF-Bestellliste</span>
	</Button>
{/snippet}

<!-- Eine Zeile statt einer Fläche, wenn nichts zu wählen ist: Der Platz gehört der Bestellung. -->
{#snippet zeile(/** @type {string} */ titel, /** @type {string} */ text, icon = false)}
	<div class="flex items-center justify-between gap-3 border-t border-outline-variant pt-5">
		<div class="flex min-w-0 gap-2 text-on-surface-variant">
			{#if icon}<Icon name="check_circle" />{/if}
			<div class="min-w-0">
				<p class="text-body-medium-emphasized text-on-surface">{titel}</p>
				<p class="text-xs">{text}</p>
			</div>
		</div>
		{@render pdf()}
	</div>
{/snippet}

<section aria-label="Bestellbedarf" bind:this={wurzel}>
	{#if ladefehler && !vorschlaege.length}
		<div class="border-t border-outline-variant pt-5">
			<LadeFehler
				titel="Bestellbedarf nicht geladen"
				text={ladefehler}
				onerneut={() => bestellStore.loadRecommendations()}
			/>
		</div>
	{:else if laedt && !vorschlaege.length}
		<div class="flex justify-center border-t border-outline-variant py-8">
			<Ladekreis size="lg" verzoegert etikett="Bestellbedarf lädt" />
		</div>
	{:else if !vorschlaege.length}
		{#if grundlage.ausgeschaltet}
			{@render zeile(
				'Bestellbedarf ausgeschaltet',
				'Einschalten unter Einstellungen, Bestellwesen'
			)}
		{:else if !grundlage.kinder_mit_schulbuch}
			{@render zeile(
				'Noch keine Grundlage',
				'In diesem Schuljahr wurde noch kein Schulbuch ausgegeben; der Bedarf ergibt sich aus den Ausleihen'
			)}
		{:else}
			{@render zeile(
				'Kein Schulbuch fehlt',
				`Der Bestand reicht für das nächste Schuljahr, ${grundlageText}`,
				true
			)}
		{/if}
	{:else if !offene.length}
		{@render zeile(
			'Der ganze Bestellbedarf steht in der Bestellung',
			`${anzahlText(vorschlaege.length, 'Titel', 'Titel')}, ${grundlageText}`,
			true
		)}
	{:else}
		<!-- Kopf wie bei den Töpfen darüber: Name, dann was zählt. -->
		<div class="flex items-center justify-between gap-3 border-b border-outline-variant pb-2">
			<h3 class="text-label-medium text-on-surface">
				Bestellbedarf
				<span class="font-normal text-on-surface-variant">· {kopfText}</span>
			</h3>
			{@render pdf()}
		</div>
		<div
			aria-hidden="true"
			class="hidden items-center gap-x-3 pt-2 text-xs font-medium text-on-surface-variant select-none @3xl:flex"
		>
			<span class="w-7 shrink-0"></span>
			<span class="flex-1">Titel</span>
			<span class={STELLEN}>
				<span class="{SPALTEN.bestand} text-right">Bestand</span>
				<span class={SPALTEN.knopf}></span>
			</span>
			<span class="order-2 w-8 shrink-0"></span>
		</div>
		<ul class="divide-y divide-outline-variant">
			{#each sichtbare as r (r.id)}
				<BedarfZeile {r} {onAddToCart} onNeueAuflage={(z) => (neueAuflageFuer = z)} />
			{/each}
		</ul>
		{#if offene.length > maxVisible}
			<!-- Hausbauteil statt rohem <button> mit text-xs (12 px kennt M3 fuer keinen Knopf).
			     typo-rollen sah ihn nie: erscheint erst ab 61 Titeln, auf dem Zielsystem (247) immer. -->
			<Button variant="ghost" class="w-full sm:w-auto" onclick={() => (maxVisible += 60)}>
				Weitere {offene.length - maxVisible} anzeigen
			</Button>
		{/if}
		<p class="pt-3 text-xs text-on-surface-variant">
			{grundlageText.charAt(0).toUpperCase() + grundlageText.slice(1)}
		</p>
	{/if}

	<!-- Nach dem Zuordnen ist die Zeile ein Buch mit zwei Auflagen: neu laden, damit sie so dasteht. -->
	<NeueAuflageDialog
		zeile={neueAuflageFuer}
		onschliessen={() => (neueAuflageFuer = null)}
		onbestellt={(titel) => {
			const menge = neueAuflageFuer?.fehlen;
			neueAuflageFuer = null;
			onAddToCart(titel, menge);
			bestellStore.loadRecommendations();
		}}
	/>
</section>

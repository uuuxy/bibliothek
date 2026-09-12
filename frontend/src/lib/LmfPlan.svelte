<!-- @component LmfPlan — der Planer: eine REIHENFOLGE von Klassen, die der Server auf
     Schultage × Stunden gießt (Peter, 05.09.2026): Rahmen, Reihenfolge mit Vorschau der
     Plätze und, darüber, „Noch nicht im Plan". Gespeichert wird ein ENTWURF (Migration
     100), für Portal und Kollegiums-PDF unsichtbar; erst „Veröffentlichen" macht ihn
     gültig und setzt beim Büchertausch die Fristen (die Schulleitung nimmt ihn vorher
     ab). System → „Schuljahreswechsel".

     Aufbau seit 06.09.2026 nach Arbeitsablauf, nicht nach Datenmodell (Peter: „es steht
     oben was und unten was"): Kopf haftet oben, freie Tage als Chips, fehlende Klassen
     direkt über der Reihenfolge; einplanen setzt an den Platz (lmfplanZeilen.js).

     Hier steht nur noch der Bildschirm. Zustand und die Wege, die ihn bewegen (laden,
     Vorschau, speichern, veröffentlichen, verwerfen), liegen seit dem Rasterdurchgang am
     06.09.2026 in lmfplanPlaner.svelte.js. -->
<script>
	import { onMount, untrack } from 'svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { abonniere } from './liveEvents.js';
	import PageShell from './components/layout/PageShell.svelte';
	import Button from './components/ui/Button.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import LmfPlanKopf from './components/lmfplan/LmfPlanKopf.svelte';
	import LmfPlanRahmen from './components/lmfplan/LmfPlanRahmen.svelte';
	import LmfPlanReihenfolge from './components/lmfplan/LmfPlanReihenfolge.svelte';
	import LmfPlanUngespeichert from './components/lmfplan/LmfPlanUngespeichert.svelte';
	import { erzeugePlaner } from './lmfplanPlaner.svelte.js';
	import * as dienst from './lmfplanDienst.js';
	import { uiStore } from './stores/uiStore.svelte.js';

	const planer = erzeugePlaner();
	const z = planer.zustand;

	// Vorschau: der Server rechnet die Plätze, sobald sich etwas ändert, wovon sie
	// abhängen (dienst.vorschauSchluessel, entprellt — die Zahl steht im Dienst). Der Effekt liest den
	// Entwurf und schreibt nichts von dem, was er liest — kein Effekt auf eigenen State.
	$effect(() => {
		void dienst.vorschauSchluessel(z.entwurf);
		void z.entwurf.zeilen.length;
		const art = z.art;
		const timer = setTimeout(
			() =>
				planer.vorschau(
					art,
					untrack(() => JSON.parse(JSON.stringify(z.entwurf)))
				),
			dienst.VORSCHAU_ENTPRELLUNG_MS
		);
		return () => clearTimeout(timer);
	});

	const marker = $derived(dienst.klassenMarker(z.stand));
	const draussen = $derived(dienst.bewusstDraussen(z.stand));
	const gueltig = $derived(
		dienst.ankerGesetzt(z.art, z.entwurf) &&
			z.entwurf.zeilen.every((x) => x.klassen.length > 0 || x.vermerk.trim() !== '') &&
			dienst.festePlaetzeVollstaendig(z.entwurf.zeilen)
	);

	// Nicht `onMount(planer.lade)`: Svelte nähme die zurückgegebene Zusage als
	// Aufräum-Funktion. Die Abmeldung des Abonnements IST die Aufräum-Funktion.
	//
	// Zwei Bibliothekskräfte bauen an derselben Reihenfolge — bisher erfuhr keine von
	// der anderen, bis sie neu lud, und wer danach speicherte, überschrieb den fremden
	// Stand. Nur abonnieren, nicht verbinden (liveEvents.js). Das Nachladen entscheidet
	// `fremdesSignal`: still, solange hier nichts Ungespeichertes steht.
	onMount(() => {
		planer.lade();
		return abonniere('lmf-plan', () => planer.fremdesSignal());
	});

	// Verlassen-Schutz (Register B, 07.09.2026): Bis dahin ging eine umgeordnete
	// Reihenfolge beim Klick auf einen anderen Menüpunkt wortlos verloren. Zwei Türen,
	// eine Wahrheit: Der Tab-Wechsel fragt den Wächter im uiStore, das Schließen des
	// Fensters den Browser-Dialog — beide über hatUngespeichertes(), kein zweites Flag.
	onMount(() => {
		uiStore.verlassenSperre = () => planer.hatUngespeichertes();
		/** @param {BeforeUnloadEvent} e */
		const warnung = (e) => {
			if (planer.hatUngespeichertes()) e.preventDefault();
		};
		window.addEventListener('beforeunload', warnung);
		return () => {
			uiStore.verlassenSperre = null;
			uiStore.blockierterWechsel = null;
			window.removeEventListener('beforeunload', warnung);
		};
	});
</script>

<LmfPlanUngespeichert
	offen={uiStore.blockierterWechsel !== null}
	onbleiben={() => uiStore.bleibe()}
	onverwerfen={() => uiStore.erzwingeWechsel()}
/>

<PageShell>
	<LmfPlanKopf
		art={z.art}
		stand={z.stand}
		laedt={z.laedt}
		ladeFehler={z.ladeFehler}
		{gueltig}
		speichert={z.speichert}
		onart={planer.waehleArt}
		onpdf={planer.pdf}
		onverwerfen={planer.verwerfen}
		onspeichern={planer.speichern}
		onveroeffentlichen={planer.veroeffentlichen}
	/>

	{#if z.fremdeAenderung}
		<!-- Kein Dialog und kein automatisches Nachladen: Hier steht ungespeicherte Arbeit,
		     und beides nähme sie weg. Wer neu lädt, entscheidet selbst. -->
		<div
			class="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-surface-container px-4 py-3"
			role="status"
		>
			<p class="text-sm text-on-surface-variant">
				Der Plan wurde an einem anderen Platz geändert. Hier stehen ungespeicherte Änderungen —
				deshalb wurde nichts überschrieben.
			</p>
			<Button variant="secondary" onclick={planer.lade}>Neu laden und verwerfen</Button>
		</div>
	{/if}

	{#if z.laedt}
		<div class="flex items-center justify-center py-12">
			<Ladekreis size="lg" />
		</div>
	{:else if z.ladeFehler}
		<LadeFehler
			onerneut={planer.lade}
			titel="Plan nicht geladen"
			text="Der gespeicherte Plan konnte nicht abgerufen werden. Der Planer bleibt geschlossen — sonst würde ein Klick auf „Plan speichern“ den echten Plan durch diesen Entwurf ersetzen und die Fristen der Klassen zurückstellen."
		/>
	{:else}
		<div class="space-y-8">
			<LmfPlanRahmen
				art={z.art}
				bind:entwurf={z.entwurf}
				ausfaelle={z.ausfaelle}
				beginn={z.beginn}
				sommerferien={z.stand?.sommerferien ?? null}
			/>
			<LmfPlanReihenfolge
				bind:zeilen={z.entwurf.zeilen}
				plaetze={z.plaetze}
				{marker}
				bereit={dienst.ankerGesetzt(z.art, z.entwurf)}
				ausgelassen={z.entwurf.ausgelassen}
				{draussen}
				markiert={z.markiert}
				onklasseraus={(k) => (z.entwurf = dienst.klasseRaus(z.entwurf, k))}
				onhinein={planer.klasseHinein}
				ontausch={(i, alt, neu) => (z.entwurf = dienst.klasseTauschen(z.entwurf, i, alt, neu))}
			/>
		</div>
	{/if}
</PageShell>

<script>
	import { etikettFormat } from './etikettformate.js';
	import { sendeBogen } from './etikettenDruck.js';
	import { ersteSeiteAlsBild } from './pdfSeite.js';
	import { etikettenStore } from '../../stores/etiketten.svelte.js';
	import { druckWarteschlange } from '../../stores/druckWarteschlange.svelte.js';
	import { leseApiFehler } from '../../apiFetch.js';
	import { fehlertext } from '../../utils/fehlertext.js';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';

	// Die Vorschau ist das Druck-PDF des Servers, wie bei Chrome: Eine eigene Zeichnung im Browser
	// lief auseinander, ihr fehlten Schulname, Jahr und Vermerk (docs/OFFEN.md 5.114). Gezeigt wird
	// die erste Seite in zwei Dritteln von A4, in einer schmaleren Spalte verkleinert; alle Bogen
	// zeigt das PDF, das „A4-Bogen drucken" öffnet.
	const HOECHSTBREITE_PX = (140 * 96) / 25.4;
	// Wer Kästchen anklickt oder die Startposition tippt, löst nicht für jeden Schritt ein PDF aus.
	const WARTEZEIT_MS = 300;

	const format = $derived(etikettFormat(etikettenStore.formatId));
	const ersterBogen = $derived(
		etikettenStore.bogen.slice(0, etikettenStore.felderJeBogen).filter((lbl) => !lbl.isBlank)
	);
	const bogenZahl = $derived(Math.ceil(etikettenStore.bogen.length / etikettenStore.felderJeBogen));
	const etikettenZahl = $derived(etikettenStore.bogen.filter((lbl) => !lbl.isBlank).length);

	let bild = $state('');
	let laedt = $state(false);
	let fehler = $state('');
	let versuch = $state(0);

	$effect(() => {
		const bogen = {
			formatId: etikettenStore.formatId,
			startFeld: etikettenStore.startFeld,
			alsQR: etikettenStore.barcodeArt === 'qr'
		};
		const etiketten = ersterBogen.map(({ barcode_id, titel, autor }) => ({
			barcode_id,
			titel,
			autor
		}));
		void versuch;
		if (etiketten.length === 0) return;
		// Nur die Antwort auf die jüngste Auswahl zählt; eine langsamere ältere zeigte sonst den
		// Bogen von vorhin.
		let veraltet = false;
		const zeitgeber = setTimeout(async () => {
			laedt = true;
			try {
				const antwort = await sendeBogen(bogen, etiketten);
				if (!antwort.ok) throw new Error(await leseApiFehler(antwort));
				const neu = await ersteSeiteAlsBild(await antwort.arrayBuffer(), HOECHSTBREITE_PX);
				if (veraltet) return;
				if (bild) URL.revokeObjectURL(bild);
				bild = URL.createObjectURL(neu);
				fehler = '';
			} catch (err) {
				if (!veraltet) fehler = fehlertext(err);
			} finally {
				if (!veraltet) laedt = false;
			}
		}, WARTEZEIT_MS);
		return () => {
			veraltet = true;
			clearTimeout(zeitgeber);
		};
	});

	$effect(() => () => {
		if (bild) URL.revokeObjectURL(bild);
	});
</script>

<!-- Die Vorschau bekommt die kleinere Hälfte (5 von 12): Die Arbeit geschieht im Formular
     daneben. Eine getönte Fläche ohne gestrichelten Rand, denn gestrichelt heißt in dieser
     Anwendung „hier gehört etwas hin", und die Vorschau ist die Ausgabe. -->
<div
	class="lg:col-span-5 flex flex-col items-center justify-start rounded-xl bg-surface-container p-6"
>
	<span class="text-xs text-on-surface-variant font-medium mb-4"
		>A4 Etiketten-Vorschau · {format.kurz}</span
	>

	{#if !etikettenStore.gewaehlterTitel && (druckWarteschlange.copies?.length ?? 0) === 0}
		<div class="grow flex flex-col items-center justify-center text-on-surface-variant py-12">
			<span>Kein Buch ausgewählt</span>
			<span class="mt-1 text-xs text-on-surface-variant"
				>Suche einen Titel links, um die Live-Vorschau zu aktivieren.</span
			>
		</div>
	{:else if etikettenStore.bogen.length === 0}
		<div class="grow flex flex-col items-center justify-center text-on-surface-variant py-12">
			<span>Keine Etiketten gewählt</span>
			<span class="mt-1 text-xs text-on-surface-variant"
				>Wähle mindestens ein Exemplar oder erhöhe die Menge.</span
			>
		</div>
	{:else}
		<div class="flex w-full justify-center">
			{#if fehler}
				<LadeFehler titel="Vorschau nicht geladen" text={fehler} onerneut={() => (versuch += 1)} />
			{:else if bild}
				<!-- Hell auch im dunklen Design: Die Vorschau zeigt Papier. Zeichnet der Server neu,
				     bleibt der alte Bogen abgeblendet stehen, darüber der Ladekreis (M3, Progress
				     indicators: 200 ms bis 5 s „Use a loading indicator"). -->
				<div class="relative w-full" style="max-width: {HOECHSTBREITE_PX}px;">
					<img
						src={bild}
						alt="Erster Bogen, wie er gedruckt wird"
						data-testid="etiketten-blatt"
						class="schema-hell w-full shadow-2xl transition-opacity {laedt ? 'opacity-60' : ''}"
					/>
					{#if laedt}
						<div class="absolute inset-0 flex items-center justify-center"><Ladekreis /></div>
					{/if}
				</div>
			{:else}
				<div class="py-12"><Ladekreis /></div>
			{/if}
		</div>
		{#if bogenZahl > 1}
			<p class="mt-4 text-xs font-medium text-on-surface-variant">
				Bogen 1 von {bogenZahl} · {etikettenZahl} Etiketten
			</p>
		{/if}
	{/if}
</div>

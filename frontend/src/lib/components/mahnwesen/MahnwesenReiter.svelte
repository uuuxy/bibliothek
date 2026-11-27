<!-- @component MahnwesenReiter — Register nach Dringlichkeit, mit den Zahlen dahinter.

     Die vier Register waren viermal derselbe Block mit vertauschten Wörtern; hier
     stehen sie als Liste, damit ein fünftes Register eine Zeile kostet und nicht
     zwanzig. Die Zählungen wohnen bei den Registern, weil sie sonst nirgends
     gebraucht werden.

     Die Leiste selbst kommt aus ui/Reiter.svelte, das die ARIA-Rollen und die Farben aus
     den M3-Rollen mitbringt. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import { bescheideStore } from '../../stores/bescheide.svelte.js';
	import Reiter from '../ui/Reiter.svelte';
	import { ANSICHTEN, berechneMahnstufe, maxTageUeberfaellig } from './mahnungen.js';

	// Die Mahnstufe eines Kindes wie in der Liste (stores/mahnwesen.svelte.js): Mit einer eigenen
	// Grenze hier liefe die Zahl am Reiter der Liste davon.
	/** @param {any} s */
	const mahnstufe = (s) => berechneMahnstufe(maxTageUeberfaellig(s.medien));
	/** @param {(s: any) => boolean} passt */
	const zaehle = (passt) =>
		mahnwesenStore.klassen.reduce(
			(/** @type {number} */ sum, /** @type {any} */ k) => sum + k.schueler.filter(passt).length,
			0
		);

	// `id` ist der Filterwert des Stores, damit Reiter und Liste dieselbe Sprache sprechen.
	// Vierter Eintrag: Schadensersatz — die Geld-Seite des Verfahrens. Die ersten drei
	// Reiter fragen „Wer hat Bücher zu spät?", dieser fragt „Wer schuldet Geld?": Kinder
	// mit gemeldetem Verlust oder Schaden, mit oder ohne Brief. Die Zahl zählt, was bei
	// der Schule liegt (bescheidStatus.liegtBeiDerSchule) — nicht nur die abgelaufenen
	// Fristen, sonst stand ein frischer Brief mit „0" am Reiter.
	const register = $derived(
		[
			{ id: 'Alle', anzahl: zaehle(() => true) },
			{ id: '1. Erinnerung', anzahl: zaehle((s) => mahnstufe(s) === '1. Erinnerung') },
			{ id: 'Mahnung', anzahl: zaehle((s) => mahnstufe(s) === 'Mahnung') },
			{ id: 'Schadensersatz', anzahl: bescheideStore.beiDerSchule }
		].map((r) => ({ ...r, label: ANSICHTEN[r.id] }))
	);
	// Das Register „Kollegium" (klasse='lehrer') ist mit Migration 072 gefallen:
	// Lehrkräfte sind Personal-Konten, ihre Handapparat-Ausleihen laufen bewusst
	// ohne Mahn-Eskalation (1 Jahr Frist).
</script>

<Reiter
	etikett="Mahnstufen"
	reiter={register}
	aktiv={mahnwesenStore.stufenFilter}
	onwahl={(id) => (mahnwesenStore.stufenFilter = id)}
	klasse="print:hidden"
/>

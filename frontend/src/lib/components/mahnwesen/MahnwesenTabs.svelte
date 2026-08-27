<!-- @component MahnwesenTabs — Register nach Dringlichkeit, mit den Zahlen dahinter.

     Die vier Register waren viermal derselbe Block mit vertauschten Wörtern; hier
     stehen sie als Liste, damit ein fünftes Register eine Zeile kostet und nicht
     zwanzig. Die Zählungen wohnen bei den Registern, weil sie sonst nirgends
     gebraucht werden.

     Die Leiste selbst kommt seit 27.08.2026 aus ui/Reiter.svelte (Anlass: PR #517
     wollte ihr ARIA-Rollen von Hand geben — genau das, was das gemeinsame Bauteil
     schon mitbringt, samt Farben aus der M3-Skala statt slate/blue). -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Reiter from '../ui/Reiter.svelte';

	/** Höchste Überfälligkeit eines Schülers in Tagen. @param {any} s */
	const maxTage = (s) =>
		s.medien.reduce(
			(/** @type {number} */ max, /** @type {any} */ m) =>
				m.tage_ueberfaellig > max ? m.tage_ueberfaellig : max,
			0
		);
	/** @param {(s: any) => boolean} passt */
	const zaehle = (passt) =>
		mahnwesenStore.klassen.reduce(
			(/** @type {number} */ sum, /** @type {any} */ k) => sum + k.schueler.filter(passt).length,
			0
		);

	// „Akut fällig" = überfällig bis 14 Tage (inkl. der <24h-Fälle mit maxTage 0),
	// passend zur Mahnstufe '1. Erinnerung'. So stimmt die Register-Zahl mit der Liste.
	// `id` ist der Filterwert des Stores, damit Reiter und Liste dieselbe Sprache sprechen.
	const register = $derived([
		{ id: 'Alle', label: 'Alle', anzahl: zaehle(() => true) },
		{ id: '1. Erinnerung', label: 'Akut fällig', anzahl: zaehle((s) => maxTage(s) <= 14) },
		{ id: 'Mahnung', label: 'Eskaliert', anzahl: zaehle((s) => maxTage(s) > 14) }
	]);
	// Das Register „Kollegium" (klasse='lehrer') ist mit Migration 072 gefallen:
	// Lehrkräfte sind Personal-Konten, ihre Handapparat-Ausleihen laufen bewusst
	// ohne Mahn-Eskalation (1 Jahr Frist) — siehe Befund F4, bewertung/.
</script>

<Reiter
	etikett="Mahnstufen"
	reiter={register}
	aktiv={mahnwesenStore.activeFilter}
	onwahl={(id) => (mahnwesenStore.activeFilter = id)}
	klasse="mt-6 print:hidden"
/>

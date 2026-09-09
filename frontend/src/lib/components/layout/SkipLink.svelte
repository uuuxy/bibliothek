<!-- @component SkipLink — das erste Fokusziel jeder App-Seite (WCAG 2.4.1).

     Wer mit der Tastatur arbeitet, müsste sonst auf jeder Seite erst die ganze
     Seitenleiste durchtabben, bevor er beim Inhalt ankommt. Der Link ist unsichtbar,
     bis er den Fokus hat, und springt zum <main> des Routers (Hauptbereich.svelte).
     Er steht in App.svelte VOR der Seitenleiste — die Reihenfolge im DOM ist die
     Reihenfolge der Tabs. Gate: e2e/barrierefreiheit-axe.spec.js (Gerüst). -->
<a
	href="#hauptinhalt"
	class="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-100 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-sm focus:font-medium focus:text-on-primary"
>
	Zum Inhalt springen
</a>

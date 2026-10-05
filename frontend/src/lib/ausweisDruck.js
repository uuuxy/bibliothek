// Der Ausweisdruck aus der Akte: eine Karte, Scheckkartenformat.
//
// Eigene Datei, weil StudentProfile.svelte an der Größen-Ratsche steht und das Setzen
// von Seitengröße und Druckmodus nichts mit dem Führen der Akte zu tun hat.
//
// Die @page-Regel wird als <style> eingehängt und danach wieder entfernt: Bliebe sie
// stehen, druckte die NÄCHSTE Seite dieses Tabs (Liste, Bericht) ebenfalls auf 85,6 ×
// 54 mm.

/**
 * Druckt die Ausweiskarte des gerade offenen Lesers.
 * @param {'front'|'back'|'both'} [seite] Zu druckende Ausweisseite(n).
 */
export function druckeAusweis(seite = 'both') {
	const stil = document.createElement('style');
	stil.textContent = '@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }';
	document.head.appendChild(stil);
	document.body.dataset.printMode = 'card-single';
	if (seite !== 'both') document.body.dataset.printCardSide = seite;
	window.print();
	stil.remove();
	delete document.body.dataset.printMode;
	delete document.body.dataset.printCardSide;
}

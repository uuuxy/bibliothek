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
	document.body.setAttribute('data-print-mode', 'card-single');
	if (seite !== 'both') document.body.setAttribute('data-print-card-side', seite);
	window.print();
	document.head.removeChild(stil);
	document.body.removeAttribute('data-print-mode');
	document.body.removeAttribute('data-print-card-side');
}

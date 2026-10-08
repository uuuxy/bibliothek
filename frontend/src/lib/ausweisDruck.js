// Der Ausweisdruck aus der Akte: eine Karte, Scheckkartenformat.
//
// Die @page-Regel wird als <style> eingehängt und danach wieder entfernt: Bliebe sie
// stehen, druckte die nächste Seite dieses Tabs (Liste, Bericht) ebenfalls auf 85,6 ×
// 54 mm.
import { designFuerDruck } from './designer/ausweisDesignLaden.js';

/**
 * Druckt die Ausweiskarte des gerade offenen Lesers, mit dem gespeicherten Design oder gar
 * nicht.
 * @param {'front'|'back'|'both'} [seite] Zu druckende Ausweisseite(n).
 */
export async function druckeAusweis(seite = 'both') {
	if (!(await designFuerDruck())) return;
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

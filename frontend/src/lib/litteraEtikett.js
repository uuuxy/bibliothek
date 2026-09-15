/**
 * Rückrechnung eines Littera-Etiketts auf die Exemplarnummer — der Zwilling von
 * internal/service/littera_etikett.go für die Theke ohne Netz.
 *
 * Littera druckt die Mediennummer als Klartext, codiert im Strichcode aber eine EAN-13:
 * [Mediennummer, rechts mit Nullen auf 8 Stellen][Bibliotheksnummer, 3 Stellen]
 * [Stellenzahl der Mediennummer][Prüfziffer]. Ohne Netz muss die Theke das Etikett selbst
 * zurückrechnen und die Nummer in der Barcode-Liste nachschlagen; die Liste führt nur Nummern
 * (Rasterdurchgang 15.09.2026, OFFEN.md 5.15).
 *
 * Die Bibliotheksnummer wird wie am Server nicht geprüft. Beide Seiten lesen dieselben Prüffälle
 * (litteraEtikett.faelle.json); rechnen sie verschieden, wird der Go- oder der Vitest rot.
 *
 * @param {string} scan
 * @returns {string | null} die Exemplarnummer, oder null, wenn der Scan kein Littera-Etikett ist
 */
export function dekodiereLitteraEtikett(scan) {
	if (typeof scan !== 'string' || !/^\d{13}$/.test(scan)) return null;
	if (!ean13PruefzifferStimmt(scan)) return null;

	const laenge = Number(scan[11]);
	if (laenge < 1 || laenge > 8) return null;
	const nummer = scan.slice(0, laenge);
	if (nummer[0] === '0') return null;
	// Zwischen Mediennummer und Bibliotheksnummer steht ausschließlich Polsterung.
	if (!/^0*$/.test(scan.slice(laenge, 8))) return null;
	return nummer;
}

/** @param {string} scan dreizehn Ziffern */
function ean13PruefzifferStimmt(scan) {
	let summe = 0;
	for (let i = 0; i < 12; i++) {
		summe += Number(scan[i]) * (i % 2 === 1 ? 3 : 1);
	}
	return Number(scan[12]) === (10 - (summe % 10)) % 10;
}

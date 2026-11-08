import { SICHERHEITSRAND_MM, STRICHCODE_RAND_MM } from './kartenRand.js';

/**
 * Wie die gestrichelte Linie des sicheren Rands aussieht, wie der Rand eines Textfelds in
 * Material 3: ruhig Outline in einfacher Stärke, aktiv in doppelter, beim Einrasten Primary und
 * Error, solange etwas zu nah am Rand steht.
 * @param {{ zuNah: boolean, eingerastet: boolean, aktiv: boolean }} lage
 */
export function linienStil({ zuNah, eingerastet, aktiv }) {
	const farbe = zuNah ? 'border-error' : eingerastet ? 'border-primary' : 'border-outline';
	return `${farbe} ${aktiv ? 'border-2' : 'border'}`;
}

/**
 * Wo die Linie steht: beim Ziehen an der Grenze des gezogenen Elements (beim Strichcode die seiner
 * Striche), sonst am sicheren Rand.
 * @param {{ links: number, oben: number, rechts: number, unten: number } | null} soll
 */
export function linienAbstand(soll) {
	if (!soll) return `${SICHERHEITSRAND_MM}mm`;
	return `${soll.oben}mm ${soll.rechts}mm ${soll.unten}mm ${soll.links}mm`;
}

/**
 * Der Satz unter der Leinwand zur Linie. Ist sie rot, sagt er warum: Die Farbe allein wäre kein
 * Hinweis für alle.
 * @param {{ type: string }[]} zuNah  die Elemente, die dem Rand zu nah stehen
 */
export function hinweisZumRand(zuNah) {
	if (zuNah.length === 0) return 'Gestrichelt: sicherer Druckbereich';
	if (zuNah.every((el) => el.type === 'barcode')) {
		const abstand = String(STRICHCODE_RAND_MM).replace('.', ',');
		return `Rot: Die Striche des Strichcodes brauchen ${abstand} mm Abstand zum Rand`;
	}
	return 'Rot: Ein Element steht zu nah am Rand';
}

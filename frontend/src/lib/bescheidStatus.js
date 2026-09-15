import { AlertTriangle, CheckCircle2 } from '@lucide/svelte';

/**
 * Liegt der Brief noch bei der Schule — ist also hier etwas zu tun oder abzuwarten?
 *
 * Offen (Frist läuft oder abgelaufen) und die Rückgabe nach der Übergabe zählen; ein
 * übergebener Brief liegt bei der Aufsicht, ein erledigter ist bezahlt oder storniert.
 * Dieselbe Frage entscheidet die Zahl am Reiter „Schadensersatz".
 *
 * @param {any} b Bescheid, wie ihn die API liefert
 * @returns {boolean}
 */
export function liegtBeiDerSchule(b) {
	return !!(b.rueckgabe_nach_uebergabe || b.status === 'offen');
}

/**
 * Der Zustand eines Schadensersatz-Bescheids als Angaben für einen StatusChip.
 *
 * EINE Stelle für beide Anzeigen: die Arbeitsliste im Mahnwesen (BescheideTabelle) und
 * die Schülerakte (StudentBescheideCard). Zwei Abschriften derselben fünf Fälle wären
 * genau die Art Doppelung, an der die Liste später „übergeben" sagt und die Akte noch
 * „offen" — bei einem Brief, der Geld fordert.
 *
 * Die Reihenfolge ist Absicht: Die Rückgabe nach der Übergabe schlägt alles, weil sie eine
 * Handlung verlangt (die Aufsicht ist zu informieren). Jeder Zustand nennt im Tipp den
 * nächsten Schritt — bis zum 15.09.2026 stand bei einem frischen Brief nur „offen", und
 * was danach kommt, stand nirgends auf dem Bildschirm.
 *
 * @param {any} b Bescheid, wie ihn die API liefert
 * @param {(iso: string) => string} datum Formatierer für das Übergabedatum
 * @returns {{ton: 'warten'|'neutral'|'erfolg'|'fehler', text: string, icon?: any, tip?: string, detail?: string}}
 */
export function bescheidStatus(b, datum) {
	if (b.rueckgabe_nach_uebergabe) {
		return {
			ton: 'warten',
			icon: AlertTriangle,
			text: 'Rückgabe nach Übergabe',
			tip: 'Das Buch kam zurück, nachdem der Fall abgegeben war — die Aufsicht ist zu informieren.'
		};
	}
	if (b.status === 'uebergeben') {
		return {
			ton: 'neutral',
			text: 'übergeben',
			detail: b.uebergeben_am ? datum(b.uebergeben_am) : undefined,
			tip: 'Der Fall liegt bei der Aufsicht; die Schule veranlasst nichts mehr.'
		};
	}
	if (b.status === 'erledigt') {
		return { ton: 'erfolg', icon: CheckCircle2, text: 'erledigt' };
	}
	if (b.frist_abgelaufen) {
		return {
			ton: 'fehler',
			icon: AlertTriangle,
			text: 'Frist abgelaufen',
			tip: 'Original und Buchungsbeleg gehen jetzt an die Aufsicht.'
		};
	}
	return {
		ton: 'warten',
		text: 'Frist läuft',
		tip: 'Zahlung oder Rückgabe abwarten. Nach der Frist: an die Aufsicht übergeben.'
	};
}

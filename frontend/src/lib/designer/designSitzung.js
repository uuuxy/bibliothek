// Was die Sitzung über das zentral gespeicherte Ausweis-Design weiß, ohne Teil davon zu sein:
// ob es schon geladen wurde und auf welchem Änderungsstand. Nicht im idStore: Der wird
// serialisiert und ginge beim Speichern mit an den Server.

// Der Merker steuert, ob ein Bildschirm beim Öffnen nachlädt. Ohne ihn holte jeder Mount
// (Schülerdatei, Stapeldruck, Profil-Einzeldruck) den Serverstand erneut und überschrieb
// damit frische, noch nicht fertig gespeicherte Änderungen aus dem Designer: Wer dort auf
// Etikettenbogen stellte und binnen der Auto-Save-Entprellung zur Schülerdatei wechselte,
// dessen GET konnte den alten Stand zurückbringen, während der PUT noch unterwegs war.
// Innerhalb einer Sitzung ist der Store nach dem ersten Laden die Wahrheit.
let geladen = false;

// Der Änderungsstand, den der Server mit dem Design ausliefert (leer: noch keines gespeichert).
// Der Designer schickt ihn beim Speichern zurück; hat ein anderer Platz inzwischen gespeichert,
// lehnt der Server ab, statt dessen Design zu ersetzen.
let stand = '';

/** Muss ein Bildschirm das zentrale Design noch laden? */
export function designWurdeGeladen() {
	return geladen;
}

/** Der Stand des Designs, das im Store steht. */
export function designStand() {
	return stand;
}

/** Ein Design vom Server steht im Store (applyDesign). @param {unknown} neuerStand */
export function merkeDesignGeladen(neuerStand) {
	geladen = true;
	merkeDesignStand(neuerStand);
}

/** Der Server hat gespeichert und nennt den neuen Stand. @param {unknown} neuerStand */
export function merkeDesignStand(neuerStand) {
	stand = typeof neuerStand === 'string' ? neuerStand : '';
}

/** Nur für Tests: setzt den Sitzungs-Merker zurück. */
export function vergissDesignLadezustand() {
	geladen = false;
	stand = '';
}

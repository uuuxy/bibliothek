/**
 * Der sichere Rand des Ausweises. Die Kartendrucker der Schule (Datacard Sigma DS1 und DS2)
 * drucken bis an den Rand, lassen dort aber einen schmalen weißen Streifen frei, weil der
 * Druckkopf am Rand angehoben und abgesenkt wird (Datacard, Installation and Administrator's
 * Guide, „Margins at the Edge of a Card"). Was darin steht, fehlt auf der Karte.
 */

/** Abstand zum Kartenrand in mm für Text, Bilder und Strichcode; der Designer zeigt ihn gestrichelt. */
export const SICHERHEITSRAND_MM = 3;

/** Abstand der Striche eines Strichcodes zum Kartenrand in mm, nach Datacard („Bar Code Placement"). */
export const STRICHCODE_RAND_MM = 6.3;

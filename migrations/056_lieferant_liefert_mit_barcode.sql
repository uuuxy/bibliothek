-- Manche Händler bekleben die Bücher mit UNSEREN Barcodes, bevor sie liefern.
--
-- Das Kennzeichen etikett_gedruckt am Exemplar bedeutet „wir haben gedruckt", nicht „am
-- Buch klebt ein Etikett". Ein fertig beklebt geliefertes Buch war deshalb nicht von
-- einem unbeklebten zu unterscheiden: Es stand auf der Nachdruck-Liste und wäre dort
-- dauerhaft stehen geblieben — eine Karteileiche, die den Hinweis im Bestellwesen
-- entwertet.
--
-- Bisher gab es dafür nur ein Häkchen je Warenkorb-Position („Barcodes generieren").
-- Wer bei einem Händler immer beklebt beliefert wird, musste es bei JEDEM Titel neu
-- setzen; einmal vergessen genügte für die Karteileiche. Die Eigenschaft gehört an den
-- Händler, nicht an die einzelne Bestellzeile.
--
-- Vorgabe false: Das ist das bisherige Verhalten. Wer beklebt beliefert wird, schaltet
-- es in der Lieferantenverwaltung ausdrücklich ein — eine stille Umstellung bestehender
-- Händler beim Update wäre falsch, weil sie Exemplare unsichtbar machen würde, die
-- wirklich noch ein Etikett brauchen.
ALTER TABLE lieferanten
	ADD COLUMN IF NOT EXISTS liefert_mit_barcode BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN lieferanten.liefert_mit_barcode IS
	'Händler beklebt die Bücher vor der Lieferung mit unseren Barcodes. Der Barcodebogen geht weiterhin mit dem Bestellbrief mit; die Exemplare gelten sofort als beklebt und erscheinen nicht auf der Nachdruck-Liste.';

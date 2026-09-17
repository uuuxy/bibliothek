-- Migration 127: Der Listenpreis am Titel und die Zustands-Abwertung am Exemplar.
--
-- Beides verlangt die Anforderungsliste „Mahnverfahren" des Medienzentrums, bestätigt in
-- der Sichtung vom 16.09.2026 (docs/OFFEN.md 9.8):
--
--   Nr. 2  „Bei der Katalogisierung und Rückgabe von Büchern sollen Beschädigungen
--          dokumentiert und mit einem prozentualen Abwertungswert versehen werden können
--          (z. B. 20 % durch Wasserschaden). Der Buchwert soll sich dadurch automatisch
--          reduzieren."
--   Nr. 3  „Sowohl der tatsächliche Einkaufspreis als auch der Listenpreis sollen im
--          System hinterlegt werden können. Für das Mahnwesen soll auswählbar sein,
--          welcher Preis als Berechnungsgrundlage verwendet wird."
--
-- Nr. 3 ist keine Komfortfrage: Die Arbeitshilfe zum Erlass vom 17.12.2014 rechnet ab dem
-- ZWEITEN Verleihjahr mit „80 % des Neupreises des Lehrwerks zum Zeitpunkt des Verlusts".
-- Diesen Preis führt das System bis heute nicht; api/bescheid_handler.go übergibt hart 0,
-- und die Staffel weicht ersatzweise auf den Kaufpreis aus. Der Listenpreis IST der
-- Neupreis der Arbeitshilfe — ein Wort für eine Sache, und zwar das des Medienzentrums.

-- ── Listenpreis: NULLBAR, nicht DEFAULT 0 ────────────────────────────────────────────
--
-- Der Unterschied trägt eine Bedeutung, und sie steht am Ende in einem Bescheid an
-- Erziehungsberechtigte: NULL heißt „kein Listenpreis erfasst" — dann nimmt die Staffel
-- ersatzweise den Kaufpreis und sagt das in ihrer Herleitung. Eine 0 hieße „dieses Buch
-- kostet heute nichts" und ergäbe einen Ersatzbetrag von 0,00 €.
--
-- Das ist die Bugklasse „fehlendes Feld, zwei Bedeutungen": Wer hier DEFAULT 0.00 setzt,
-- macht aus einer ehrlichen Lücke eine falsche Zahl, und niemand sieht es dem Bescheid an.
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS listenpreis DECIMAL(10, 2);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_listenpreis_nonneg') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_listenpreis_nonneg
            CHECK (listenpreis IS NULL OR listenpreis >= 0);
    END IF;
END $$;

-- ── Zustands-Abwertung: NOT NULL DEFAULT 0 ──────────────────────────────────────────
--
-- Hier ist es umgekehrt: „kein Schaden erfasst" und „0 % abgewertet" sind dasselbe, und
-- jedes Exemplar hat einen Zustand. Ein NULL wäre eine dritte Bedeutung ohne Inhalt — und
-- jede Rechnung müsste sie abfangen (COALESCE in jeder Abfrage, vergisst man einmal, ist
-- der Betrag NULL statt einer Zahl).
--
-- SMALLINT mit CHECK 0..100: Der Wert ist ein Prozentsatz, keine Kommazahl. „20 % durch
-- Wasserschaden" ist die Sprache der Anforderung; halbe Prozent hat noch niemand verlangt,
-- und eine Nachkommastelle in einem Ermessensabschlag wäre Scheingenauigkeit.
--
-- 100 ist ausdrücklich erlaubt: ein Buch, das so beschädigt ist, dass es unbenutzbar ist —
-- die zweite Fallgruppe des Musteranschreibens. Der Ersatzbetrag ist dann 0, und das ist
-- richtig: Für ein wertloses Buch fordert die Schule nichts, sie sondert es aus.
ALTER TABLE buecher_exemplare
    ADD COLUMN IF NOT EXISTS zustand_abwertung_prozent SMALLINT NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_zustand_abwertung_bereich') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_zustand_abwertung_bereich
            CHECK (zustand_abwertung_prozent BETWEEN 0 AND 100);
    END IF;
END $$;

-- KEINE Rückfüllung, kein Trigger, keine Umdeutung vorhandener Daten.
--
-- Der Altbestand startet mit leerem Listenpreis und 0 % Abwertung — das ist der ehrliche
-- Zustand. `zustand_notiz` steht daneben und bleibt, was sie ist: Menschentext. Sie
-- maschinell nach „Wasserschaden" zu durchsuchen und daraus Prozente zu raten, wäre genau
-- die Sorte stiller Schätzung, die in einer Forderung nichts zu suchen hat. Wer einen
-- Abschlag will, trägt ihn ein.

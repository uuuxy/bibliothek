-- =============================================================================
-- Migration 054: Namenssuche schreibweisen-unabhängig machen
-- =============================================================================
-- Die Schülersuche lief über blankes ILIKE. Damit war jeder Name mit Diakritika
-- über die Tastatur unerreichbar: "Garcia" fand García nicht, "Ozturk" nicht
-- Öztürk, "Muller" nicht Müller. An einer deutschen Tastatur tippt an der Theke
-- niemand ć, ł, ș, ğ oder ệ — betroffen sind also genau die Namen, bei denen die
-- Suche am nötigsten ist. Ebenso wenig fand "Mueller" die Müller: Die deutsche
-- Ersatzschreibung ist in Namenslisten alltäglich, in beide Richtungen — mal ist
-- der Name mit Umlaut erfasst und wird ohne getippt, mal umgekehrt.
--
-- suchnorm() bildet beide Seiten des Vergleichs auf eine gemeinsame Normalform ab.
-- Rein additiv: eine Extension, zwei Funktionen, Indizes. Keine Datenänderung.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "unaccent";

-- suchnorm() ist die Normalform für alle Namens- und Titelvergleiche. Zwei Schritte:
--
--   1. unaccent(lower(...)) — Diakritika weg: Öztürk → ozturk, García → garcia,
--      Nguyễn → nguyen, Łukasz → lukasz, Straße → strasse (ß wird zu ss).
--   2. Die deutschen Ersatzschreibungen auf denselben Nenner ziehen: ss→s, ue→u,
--      oe→o, ae→a. Erst dadurch treffen sich "Mueller" und "Müller" (beide → muller),
--      "Oeztuerk" und "Öztürk" (beide → ozturk), "Strasse" und "Straße" (beide →
--      strase). Schritt 2 muss NACH unaccent laufen, sonst käme ß nie bei ss an.
--
-- Der Preis ist etwas mehr Unschärfe: "Bauer" und "Baur" fallen auf dieselbe Form,
-- ebenso "Goethe" und "Gothe". Bei einer Namenssuche ist ein zusätzlicher Kandidat
-- in der Liste harmlos — die Sortierung stellt die genauere Schreibweise nach oben,
-- und die Alternative wäre, dass die Kraft an der Theke gar nichts findet.
--
-- IMMUTABLE ist Pflicht: unaccent() selbst ist nur STABLE (es hängt an einem
-- Wörterbuch, das sich zur Laufzeit ändern könnte). Erst der hier festgenagelte
-- Wörterbuchname macht den Ausdruck indexierbar — ohne das wären die Indizes unten
-- nicht anlegbar und jede Suche bliebe ein Seq Scan.
CREATE OR REPLACE FUNCTION suchnorm(text)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
PARALLEL SAFE
AS $$
    SELECT replace(replace(replace(replace(
               public.unaccent('public.unaccent'::regdictionary, lower($1)),
           'ss', 's'), 'ue', 'u'), 'oe', 'o'), 'ae', 'a')
$$;

-- Die Suche vergleicht immer suchnorm(spalte) — die Indizes müssen exakt diesen
-- Ausdruck tragen, sonst greifen sie nicht.
CREATE INDEX IF NOT EXISTS idx_schueler_vorname_suchnorm_trgm
    ON schueler USING gin (suchnorm(vorname) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_schueler_nachname_suchnorm_trgm
    ON schueler USING gin (suchnorm(nachname) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_schueler_barcode_lower_trgm
    ON schueler USING gin (lower(barcode_id) gin_trgm_ops);

-- Die Titelsuche der Omnibox läuft über dieselbe Token-Logik und braucht daher
-- dieselben Ausdrucks-Indizes. Die vorhandenen Trigramm-Indizes auf den rohen
-- Spalten (idx_buecher_titel_trgm & Co.) greifen für den suchnorm-Ausdruck
-- nicht — bei 80.000 Titeln wäre das ein Seq Scan pro Tastendruck.
CREATE INDEX IF NOT EXISTS idx_buecher_titel_suchnorm_trgm
    ON buecher_titel USING gin (suchnorm(titel) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_buecher_autor_suchnorm_trgm
    ON buecher_titel USING gin (suchnorm(autor) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_buecher_isbn_lower_trgm
    ON buecher_titel USING gin (lower(isbn) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_buecher_signatur_lower_trgm
    ON buecher_titel USING gin (lower(signatur) gin_trgm_ops);

-- ISBN-Treffer sollen unabhängig von Bindestrichen greifen. Ohne diesen Index ist
-- der replace()-Vergleich der einzige nicht-indexierbare Zweig der OR-Kette — und
-- ein einziger solcher Zweig zwingt die gesamte Abfrage in den Seq Scan.
CREATE INDEX IF NOT EXISTS idx_buecher_isbn_normalisiert
    ON buecher_titel (replace(isbn, '-', ''));

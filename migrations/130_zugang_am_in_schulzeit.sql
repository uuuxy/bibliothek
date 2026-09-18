-- =============================================================================
-- Migration 130: das Zugangsdatum in der Schulzeitzone stempeln
-- =============================================================================
-- Migration 129 stempelt beim Wareneingang CURRENT_DATE. Die Datenbank-Sitzung läuft in
-- UTC; zwischen Mitternacht in Berlin und Mitternacht UTC ist CURRENT_DATE noch der
-- Vortag. Ein Wareneingang, der um halb eins gebucht wird, trägt damit den Vortag als
-- Zugang — und das Zugangsbuch ist der Nachweis zum Stichtag (15.3./15.9.): Am Stichtag
-- selbst landete die Lieferung so im falschen Halbjahr.
--
-- Gemessen am 18.09.2026 um 00:22 Uhr an der Testdatenbank:
--   SELECT CURRENT_DATE, (now() AT TIME ZONE 'Europe/Berlin')::date;
--   → 2026-09-17 | 2026-09-18
--
-- Festgehalten haben es TestZugangsdatum_ErstBeimEintreffen und
-- TestZugangsbuch_ZulaufIstKeinZugang. Ab 2 Uhr werden beide von allein grün, ohne dass
-- sich etwas geändert hätte; in der CI (UTC) fällt der Fehler nie auf.
--
-- Der Ausdruck ist derselbe wie in pkg/schulzeit (SQLHeute) — die vierte Stelle dieser
-- Art nach Bescheid-Frist, Volljährigkeit und Mahnlauf. Er steht hier ausgeschrieben,
-- weil eine SQL-Funktion kein Go-Paket importiert; die Fassung in pkg/schulzeit bleibt
-- die Quelle für alles, was aus Go kommt.
--
-- Der INSERT-Zweig bleibt unverändert: Dort zählt das Datum, das die Zeile mitbringt
-- (Littera-Übernahme, Listenimport), nicht der heutige Tag.
--
-- Bereits gestempelte Zeilen werden NICHT nachgebessert: Ob ein Zugang vom 17.09. in
-- jenen zwei Stunden entstanden ist, steht nirgends — das ließe sich nur raten.
--
-- Idempotent: CREATE OR REPLACE. Die beiden Trigger zeigen auf die Funktion und bleiben
-- unberührt.

CREATE OR REPLACE FUNCTION stempel_zugang_am()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW.zugang_am := NEW.erworben_am;
    ELSE
        NEW.zugang_am := (now() AT TIME ZONE 'Europe/Berlin')::date;
    END IF;
    RETURN NEW;
END $$;

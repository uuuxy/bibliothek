-- Migration 112: Eine abholbereite Vormerkung folgt ihrem Exemplar — als Regel der
-- Datenbank, nicht als Verabredung zwischen sieben Schreibpfaden.
--
-- „abholbereit" heißt: GENAU DIESES Exemplar liegt für das Kind im Abholfach
-- (bereitgestellt_exemplar_id). Verschwindet es aus dem Umlauf — ausgesondert, nicht mehr
-- ausleihbar (Defekt) oder gelöscht —, muss die Vormerkung zurück auf „wartend". Sonst
-- löscht der Verfall-Lauf sie nach drei Tagen als „nicht abgeholt" (das Kind verliert
-- seinen Platz, ohne etwas versäumt zu haben), und der Nächste wird nie bedient.
--
-- Bis zum 10.09.2026 tat das nur ReportDamage (repository/damage.go). Inventur-Abschluss
-- (das Abholfach steht nicht im Regal → VERLUST), Status-Editor, Aussondern, Ausbuchen,
-- Bestandskorrektur, Defekt-Markierung und das Löschen von Fehlbeständen ließen sie
-- stehen (Bestands-Durchgang, Bugklasse „Ausgang ohne Folgeschritt"). Beim Löschen hätte
-- ON DELETE SET NULL die Bindung zwar gelöst, den Status aber auf „abholbereit" gelassen.
--
-- Die Warteschlangen-Position bleibt: erstellt_am ändert sich nicht, das Kind steht
-- weiter vorn und bekommt das nächste zurückkommende Exemplar.
--
-- Vorher die Altlast: abholbereite Vormerkungen, deren Exemplar schon nicht mehr im Umlauf
-- ist (oder deren Bindung der SET NULL schon gelöst hat), gehen auf „wartend".

UPDATE vormerkungen v
   SET status = 'wartend', bereitgestellt_exemplar_id = NULL, bereitgestellt_bis = NULL
 WHERE v.status = 'abholbereit'
   AND (v.bereitgestellt_exemplar_id IS NULL
        OR EXISTS (SELECT 1 FROM buecher_exemplare e
                   WHERE e.id = v.bereitgestellt_exemplar_id
                     AND (e.ist_ausgesondert OR NOT e.ist_ausleihbar)));

CREATE OR REPLACE FUNCTION abholfach_folgt_dem_exemplar()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE vormerkungen
       SET status = 'wartend', bereitgestellt_exemplar_id = NULL, bereitgestellt_bis = NULL
     WHERE bereitgestellt_exemplar_id = OLD.id AND status = 'abholbereit';
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_exemplar_aus_dem_umlauf ON buecher_exemplare;
CREATE TRIGGER trg_exemplar_aus_dem_umlauf
AFTER UPDATE OF ist_ausgesondert, ist_ausleihbar ON buecher_exemplare
FOR EACH ROW
WHEN ((NEW.ist_ausgesondert AND NOT OLD.ist_ausgesondert)
      OR (OLD.ist_ausleihbar AND NOT NEW.ist_ausleihbar))
EXECUTE FUNCTION abholfach_folgt_dem_exemplar();

DROP TRIGGER IF EXISTS trg_exemplar_geloescht_abholfach ON buecher_exemplare;
CREATE TRIGGER trg_exemplar_geloescht_abholfach
BEFORE DELETE ON buecher_exemplare
FOR EACH ROW EXECUTE FUNCTION abholfach_folgt_dem_exemplar();

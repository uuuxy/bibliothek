-- =============================================================================
-- Migration 084: schueler.lusd_bestaetigt_am — „zuletzt im LUSD-Export bestätigt"
-- =============================================================================
-- Anlass: Der LUSD-Export der Schule enthält KEINE Schüler-ID (und sie ist auch nicht
-- zu bekommen). Der Import muss Schüler deshalb über Name + Geburtsdatum zuordnen
-- (Namensmodus, api/lusd_parser.go). Damit fehlt ihm das Merkmal, an dem der ID-Modus
-- „LUSD-verwaltet" erkennt — dort ist es schlicht lusd_id IS NOT NULL.
--
-- Warum das Merkmal unverzichtbar ist: Abgänger sind „stand im Export, steht jetzt
-- nicht mehr drin". Ohne Gedächtnis müsste der Namensmodus JEDE Handanlage, die nicht
-- im Export vorkommt (Gastschüler, Quereinsteiger vor dem nächsten Export), als
-- Abgänger behandeln — und die Abgänger-Behandlung anonymisiert irreversibel.
--
-- Diese Spalte ist das Gedächtnis: Jeder Import setzt sie für jede Zeile, die er im
-- Export wiedergefunden oder aus ihm angelegt hat. NULL heißt „nie in einem Export
-- gesehen". Abgänger im Namensmodus = bestätigt UND fehlt jetzt; nie bestätigte
-- Schüler bleiben unangetastet und werden in der Vorschau als „nicht im Export"
-- gemeldet. Der ID-Modus setzt sie ebenfalls (einheitliche Semantik), entscheidet
-- Abgänger aber weiterhin über lusd_id.
--
-- Strukturelle Spalte statt Marker im Freitext (Lehre aus Befund F1, bestellstatus):
-- Eine Marke in lusd_id ('littera:…') hat genau diese Verwechslung schon einmal
-- erzeugt (Littera-Schüler galten als LUSD-verwaltet und als Abgänger).
--
-- Backfill: Eine ECHTE LUSD-ID (nicht die Littera-Herkunftsmarke) kann nur der Import
-- oder eine bewusste Handeingabe gesetzt haben — der Schüler ist der LUSD bekannt.
-- aktualisiert_am ist der beste vorhandene Näherungswert für „zuletzt gesehen".
-- Idempotent: ADD COLUMN IF NOT EXISTS, Backfill nur WHERE … IS NULL.
-- =============================================================================

ALTER TABLE schueler ADD COLUMN IF NOT EXISTS lusd_bestaetigt_am TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN schueler.lusd_bestaetigt_am IS
    'Zeitpunkt, zu dem ein LUSD-Import diesen Schüler zuletzt im Export wiedergefunden oder aus ihm angelegt hat. NULL = nie in einem LUSD-Export gesehen (Handanlage/Littera). Namensmodus: nur bestätigte Schüler können Abgänger werden.';

UPDATE schueler
SET lusd_bestaetigt_am = aktualisiert_am
WHERE lusd_bestaetigt_am IS NULL
  AND lusd_id IS NOT NULL
  AND lusd_id NOT LIKE 'littera:%';

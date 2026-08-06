-- Der Lieferant druckt die kleinen Etiketten auf SEIN Material, und davon gibt es
-- verschiedene Bogenraster (3×7, 3×8, 4×13 …). Bis hierher stand im Bestätigungs-Link
-- fest „zweckform_l4760" — wer andere Bögen im Drucker hatte, bekam einen Ausdruck, der
-- danebenliegt. Er kann das Raster jetzt selbst wählen; diese Spalte hält fest, welches
-- es war.
--
-- Für die Bibliothek ist das die Antwort auf die Frage „wie sehen die Aufkleber aus, die
-- gleich ankommen?" — bisher stand dort nur klein/groß.
ALTER TABLE bestellungen_verlauf
	ADD COLUMN IF NOT EXISTS etiketten_format TEXT;

COMMENT ON COLUMN bestellungen_verlauf.etiketten_format IS
	'Beim Bestätigen gewähltes Bogenraster der KLEINEN Etiketten (z. B. ''zweckform_l4760''), NULL wenn nicht angegeben oder Größe ''gross''.';

-- BEWUSST OHNE CHECK-Constraint, anders als bei etiketten_groesse eine Spalte weiter.
--
-- Die gültigen Werte stehen in api/label_formats.go und sind eine Liste, die WÄCHST —
-- jeder neue Etikettenbogen kommt dort dazu. Ein CHECK hier hieße: Diese Liste steht an
-- zwei Orten, und wer nur den einen pflegt, bekommt beim Speichern einen
-- Constraint-Verstoß, den die Anwendung als HTTP 500 ausliefert (vgl. „zwei Türen zum
-- selben Zustand"). Geprüft wird der Wert deshalb an der Eingangstür —
-- istBekanntesEtikettFormat() weist Unbekanntes mit 400 ab, bevor es hier ankommt.
-- etiketten_groesse ist der andere Fall: 'klein'/'gross' sind zwei Werte, die sich nie
-- ändern.

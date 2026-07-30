-- Helfer dürfen den Katalog sehen (Betreiber-Entscheidung 30.07.2026).
--
-- Warum eine Migration und nicht nur die Vorgabe in db/seed.go: InitPermissions legt
-- fehlende Zeilen an (ON CONFLICT DO NOTHING) und rührt bestehende nie an — auf einer
-- laufenden Installation bliebe die Vorgabe also wirkungslos. Ohne diese Zeile müsste
-- die Berechtigung von Hand in der Oberfläche nachgezogen werden, und genau das
-- vergisst man beim Deploy.
--
-- Rein lesendes Recht auf Buchdaten. Personendaten hängen an view_students, das für
-- die Helfer-Rolle verschlossen bleibt.
UPDATE role_permissions
SET allowed = true
WHERE UPPER(role) = 'HELFER' AND permission = 'view_books';

-- 072: Lehrkräfte raus aus der Schüler-Tabelle (Befund F4 der unabhängigen Prüfung).
--
-- Der Spezialwert schueler.klasse='lehrer' machte eine Zeile "zur Lehrkraft":
-- eigene Mahnschiene im Frontend, still übersprungen vom Versetzungslauf,
-- als Schein-Klasse in jedem Klassen-Dropdown — und LEBENSGEFÄHRLICH am
-- LUSD-Abgleich: Wer nie im LUSD-Export steht, gilt dort als Abgänger und
-- rutscht über ist_abgaenger in die DSGVO-Löschung. Die Datenbank hat für
-- Lehrkräfte längst den richtigen Ort (benutzer + ausleihen.ausleiher_benutzer_id,
-- der Handapparat-Weg) — dieser Umzug führt beide Welten zusammen.
--
-- Regeln:
--  * Schadensfälle ziehen mit um: schadensfaelle kennt beide Seiten
--    (schueler_id ODER benutzer_id) — der Fall wechselt einfach die Spalte.
--  * E-Mail: Platzhalter unter @lehrer-umzug.invalid (benutzer.email ist NOT
--    NULL UNIQUE; gleiche Technik wie der Littera-Import mit @littera.invalid).
--    Peter ersetzt sie in der Benutzerverwaltung durch die echte Schul-Mail.
--  * Ausweis-Barcode zieht mit um (Kollision → Präfix UMZUG-), die Karte
--    scannt also weiter — jetzt über die Lehrer-Stufe der Auflösung.
--  * ALLE Ausleihen (offene wie Historie) zeigen danach auf das Konto;
--    Vormerkungen und Fotos der Schein-Zeile fallen mit ihr (CASCADE).
--  * Wiederholbar: Ein zweiter Lauf findet keine Zeilen mehr.

DO $$
DECLARE
	p RECORD;
	neue_id UUID;
	mail TEXT;
	basis_mail TEXT;
	ausweis TEXT;
	lauf INT;
	umgezogen INT := 0;
BEGIN
	FOR p IN
		SELECT id, vorname, nachname, barcode_id
		FROM schueler
		WHERE lower(btrim(klasse)) = 'lehrer' AND deleted_at IS NULL
	LOOP
		basis_mail := lower(regexp_replace(p.vorname || '.' || p.nachname, '[^a-zA-Z0-9.]+', '', 'g'));
		IF basis_mail = '' OR basis_mail = '.' THEN
			basis_mail := 'lehrkraft';
		END IF;
		mail := basis_mail || '@lehrer-umzug.invalid';
		lauf := 1;
		WHILE EXISTS (SELECT 1 FROM benutzer WHERE email = mail) LOOP
			lauf := lauf + 1;
			mail := basis_mail || '.' || lauf || '@lehrer-umzug.invalid';
		END LOOP;

		ausweis := NULLIF(btrim(coalesce(p.barcode_id, '')), '');
		IF ausweis IS NOT NULL AND EXISTS (SELECT 1 FROM benutzer WHERE barcode_id = ausweis) THEN
			ausweis := 'UMZUG-' || ausweis;
		END IF;

		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, barcode_id)
		VALUES (p.vorname, p.nachname, mail, 'kollegium', true, ausweis)
		RETURNING id INTO neue_id;

		UPDATE ausleihen
		SET ausleiher_benutzer_id = neue_id, schueler_id = NULL
		WHERE schueler_id = p.id;

		UPDATE schadensfaelle
		SET benutzer_id = neue_id, schueler_id = NULL
		WHERE schueler_id = p.id;

		DELETE FROM schueler WHERE id = p.id;
		umgezogen := umgezogen + 1;
	END LOOP;

	IF umgezogen > 0 THEN
		RAISE NOTICE 'F4-Umzug: % Schein-Schüler zu Personal-Konten umgezogen (E-Mails unter @lehrer-umzug.invalid ersetzen!)', umgezogen;
	END IF;
END $$;

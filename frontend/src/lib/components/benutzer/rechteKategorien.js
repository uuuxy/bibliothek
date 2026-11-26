// Kategorien und ihre zugeordneten Berechtigungen für BenutzerUndRechte, als Daten statt
// Markup.
//
// Jede Beschreibung nennt, was der Server unter diesem Recht tatsächlich freigibt, abgelesen
// an den verlangeRecht-Aufrufen in internal/api/routen_*.go: Wer ein Recht anhand seines Textes
// erteilt, muss dem Text trauen können. Die Namen hält internal/api/rechte_paritaet_test.go
// deckungsgleich mit den Routen; die Texte bleiben Handarbeit, wer eine Route umhängt, zieht
// die Zeile hier nach.

export const rechteBeschreibungen = [
	{
		category: 'Leserdatei',
		icon: 'group',
		items: [
			{
				key: 'view_students',
				label: 'Leserdatei anzeigen',
				desc: 'Leserdatei und Klassen einsehen, Ausleihhistorie eines Titels; Mahnwesen einsehen und die Mahnliste drucken; Kontoauszug, Ersatzforderung, Schüler-Etiketten und Ausweise drucken, im Druck-Center auch klassenweise'
			},
			{
				key: 'create_students',
				label: 'Leser anlegen',
				desc: 'Leser von Hand anlegen, mit „Leser anlegen“ in der Leserdatei'
			},
			{
				key: 'edit_students',
				label: 'Leser bearbeiten',
				desc: 'Stammdaten und Abgangsjahr ändern, Ausleihsperre setzen und aufheben, Buchschäden melden'
			},
			{
				key: 'manage_claims',
				label: 'Forderungen und Bescheide',
				desc: 'Schadensfälle als bezahlt buchen oder stornieren, Bescheide über Schadensersatz erstellen und als übergeben vermerken'
			},
			{
				key: 'send_dunning',
				label: 'Mahnungen versenden',
				desc: 'Mahnbriefe drucken, was die Mahnung zählt, sowie Mahnlisten und die Kontoauszüge der Abgänger an die Klassenleitungen mailen'
			},
			{
				key: 'delete_students',
				label: 'Leser löschen',
				desc: 'Schüler und Kollegen in den Papierkorb verschieben, Papierkorb einsehen und Einträge wiederherstellen. Mit einem Kollegen fällt sein Zugang zu „Mein Portal"; hat er einen, nur zusammen mit „Benutzer & Rechte verwalten", ein Administrator-Konto nur als Administrator.'
			},
			{
				key: 'import_students',
				label: 'LUSD-Import',
				desc: 'LUSD-Export als CSV- oder Excel-Datei in der Vorschau prüfen und einspielen'
			},
			{
				key: 'upload_photos',
				label: 'Ausweisfotos hochladen',
				desc: 'Erlaubt die Aufnahme und Zuweisung von Ausweisfotos per Webcam'
			}
		]
	},
	{
		category: 'Medien & Inventar',
		icon: 'menu_book',
		items: [
			{
				key: 'view_books',
				label: 'Medienkatalog anzeigen',
				desc: 'Titel, Exemplare, Signaturen, Systematik und Geräte einsehen; Buch-Etiketten drucken, im Druck-Center zusammen mit „Medienkatalog bearbeiten“; Schulklassen-Übersicht'
			},
			{
				key: 'edit_books',
				label: 'Medienkatalog bearbeiten',
				desc: 'Titel und Exemplare anlegen und ändern: Barcode, Status, Schadensnotiz, Defekt, Aussonderung; Etiketten-Druckstatus, Geräte und Systematik pflegen; den LMF-Plan pflegen und veröffentlichen, wobei ein Rückgabe-Plan die Fristen der Lernmittel setzt'
			},
			{
				key: 'delete_books',
				label: 'Titel und Exemplare löschen',
				desc: 'Exemplare und ganze Titel endgültig löschen'
			},
			{
				key: 'inventory_scan',
				label: 'Inventur durchführen',
				desc: 'Laufende Inventur einsehen und Exemplare einscannen. Öffnet den Menüpunkt „Inventur".'
			},
			{
				key: 'manage_inventory',
				label: 'Bestand verwalten',
				desc: 'Inventuren starten, abbrechen und abschließen; Fehlbestand als gefunden buchen oder endgültig löschen; Littera- und Bestandsimport; Cover-Abgleich'
			}
		]
	},
	{
		category: 'Bestellungen & Theke',
		icon: 'shopping_cart',
		items: [
			{
				key: 'perform_actions',
				label: 'Theke bedienen',
				desc: 'Ausleihe, Rückgabe, Scan und Suche an der Theke, die Kernfunktion der Helfer-Rolle, ohne Leserdatei und Mahnwesen'
			},
			{
				key: 'extend_loans',
				label: 'Ausleihen verlängern und Fristen ändern',
				desc: 'Ausleihen in der Leserakte verlängern oder ihre Fälligkeit setzen und unter Einstellungen, LMF-Aktionen die Lernmittel einer Klasse verlängern. Die Ausleihen dazu sieht nur, wer die Leserdatei anzeigen darf.'
			},
			{
				key: 'create_reservations',
				label: 'Klassensatz reservieren',
				desc: 'Erlaubt das Absenden einer Klassensatz-Reservierung im Kollegiums-Portal. Öffnet keine Schülerdaten, das Portal sucht über den öffentlichen OPAC.'
			},
			{
				key: 'manage_vormerkungen',
				label: 'Vormerkungen verwalten',
				desc: 'Warteliste eines Titels einsehen, Schüler vormerken und Vormerkungen löschen. Zeigt nur Name und Klasse, als enges Theken-Recht für Helfer zuschaltbar, ohne die Leserdatei zu öffnen.'
			},
			{
				key: 'view_orders',
				label: 'Bestellungen anzeigen',
				desc: 'Bestellungen, Zulauf, Bestellhistorie und Lieferanten einsehen; offene Klassensatz-Reservierungen und Lehrer-Anliegen sehen; Bestell-PDF'
			},
			{
				key: 'create_orders',
				label: 'Bestellungen verwalten',
				desc: 'Bestellungen anlegen, bestätigen und Lieferungen einbuchen; Titel per ISBN anlegen und Signatur setzen; Lieferanten pflegen; Reservierungen und Anliegen erledigen'
			},
			{
				key: 'view_graduates',
				label: 'Abgängerliste einsehen',
				desc: 'Abgängerliste mit ausstehenden Büchern einsehen und als PDF ausgeben'
			}
		]
	},
	{
		category: 'Administration & System',
		icon: 'settings',
		items: [
			{
				key: 'view_stats',
				label: 'Statistiken anzeigen',
				desc: 'Ausleih-Statistiken und Auswertungen ohne Klarnamen der Schüler einsehen'
			},
			{
				key: 'audit_logs',
				label: 'System-Logs einsehen',
				desc: 'Allgemeines Logbuch der Systemereignisse einsehen. Das Verwaltungsprotokoll braucht zusätzlich „Benutzer & Rechte verwalten".'
			},
			{
				key: 'audit_details',
				label: 'Theken-Auskunft über gelöschte Exemplare',
				desc: 'Zu einem Buch-Barcode die protokollierte Ausleihhistorie nachschlagen, auch wenn das Exemplar längst gelöscht ist. Nennt Entleiher mit Namen; jeder Abruf wird selbst protokolliert. Der Reiter erscheint unter System-Logs.'
			},
			{
				key: 'manage_settings',
				label: 'Einstellungen verwalten',
				desc: 'Alle Systemeinstellungen zu Schule, Fristen, Mahnwesen, Bestellwesen, Datenschutz und Erreichbarkeit, dazu Mail-Konfiguration und -Vorlagen, Ausweislayout, Klassen→Lehrkraft-Zuordnung, Backup-Status und Betriebsbereitschaft. Öffnet den Menüpunkt „Einstellungen".'
			},
			{
				key: 'manage_students_admin',
				label: 'Leserdatei: Sonderrechte',
				desc: 'Versetzung im Schuljahreswechsel und DSGVO-Auskunft (Art. 15) über alle Daten eines Lesers, bei einem Leser mit Zugangskonto nur zusammen mit „Benutzer & Rechte verwalten"; außerdem das sofortige endgültige Löschen aus dem Papierkorb mit dem Knopf „Endgültig löschen". Ohne diesen Knopf löscht der nächtliche Löschjob gelöschte Leser nach 180 Tagen endgültig, sofern nichts mehr offen ist.'
			},
			{
				key: 'merge_students',
				label: 'Leser zusammenführen',
				desc: 'Zwei Datensätze derselben Person zu einem vereinen, über „Doppelter Datensatz?" im Reiter Stammdaten & Adresse der Leserakte. Unumkehrbar: Ausleihen, Gebühren und Historie wandern, der zweite Datensatz wird endgültig gelöscht. Ohne das Recht ist der Knopf unsichtbar.'
			},
			{
				key: 'manage_users',
				label: 'Benutzer & Rechte verwalten',
				desc: 'Benutzerkonten anlegen und ändern, Rechte je Rolle festlegen, Verwaltungsprotokoll einsehen. Öffnet den Menüpunkt „Benutzer & Rechte". Zusammen mit „Leserdatei: Sonderrechte" auch die DSGVO-Auskunft über einen Leser mit Zugangskonto, zusammen mit „Leser löschen" das Löschen eines Kollegen mit Zugang in der Leserdatei.'
			}
		]
	}
];

// Kategorien und ihre zugeordneten Berechtigungen für den PermissionManager.
// Aus der Komponente ausgelagert (Daten statt Markup) — hält PermissionManager schlank.
//
// Jede Beschreibung nennt, was der Server unter diesem Recht tatsächlich freigibt —
// abgelesen an den RequirePermission-Aufrufen in api/routes_*.go, nicht aus der
// Erinnerung. Bis zum 24.08.2026 beschrieb z. B. edit_books nur „Schadensnotizen",
// öffnete aber auch Barcode, Status, Aussonderung, Verlängerung, Geräte und
// Systematik. Wer ein Recht anhand seines Textes erteilt, muss dem Text trauen können.
// Die NAMEN hält api/rechte_paritaet_test.go deckungsgleich mit den Routen; die Texte
// bleiben Handarbeit — wer eine Route umhängt, zieht die Zeile hier nach.
export const permissionsMetadata = [
	{
		category: 'Schülerverwaltung',
		icon: '👤',
		items: [
			{
				key: 'view_students',
				label: 'Schülerdatei anzeigen',
				desc: 'Schülerdatei und Klassen einsehen, Ausleihhistorie eines Titels; Mahnwesen, Kontoauszug, Ersatzforderung, Schüler-Etiketten und Ausweise drucken; Druck-Center'
			},
			{
				key: 'create_students',
				label: 'Schüler hinzufügen',
				desc: 'Ermöglicht das manuelle Anlegen neuer Schüler'
			},
			{
				key: 'edit_students',
				label: 'Schülerdaten ändern',
				desc: 'Stammdaten und Abgangsjahr ändern, Ausleihsperre setzen/aufheben, Buchschäden melden sowie Schadensfälle als bezahlt buchen oder stornieren'
			},
			{
				key: 'delete_students',
				label: 'Schüler löschen',
				desc: 'Schüler in den Papierkorb verschieben, Papierkorb einsehen und Einträge wiederherstellen (endgültiges Löschen: „Benutzer & Rechte verwalten")'
			},
			{
				key: 'import_students',
				label: 'LUSD / CSV Import',
				desc: 'LUSD-Export (CSV/XLSX) als Vorschau prüfen und einspielen'
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
		icon: '📚',
		items: [
			{
				key: 'view_books',
				label: 'Medienkatalog anzeigen',
				desc: 'Titel, Exemplare, Signaturen, Systematik, Fächer und Geräte einsehen; Buch-Etiketten drucken; Schulklassen-Übersicht'
			},
			{
				key: 'edit_books',
				label: 'Bücher bearbeiten',
				desc: 'Titel und Exemplare anlegen und ändern: Barcode, Status, Schadensnotiz, Defekt, Aussonderung; Ausleihen verlängern und Fälligkeit setzen; Etiketten-Druckstatus, Geräte und Systematik pflegen'
			},
			{
				key: 'delete_books',
				label: 'Bücher & Exemplare löschen',
				desc: 'Exemplare und ganze Titel endgültig löschen'
			},
			{
				key: 'inventory_scan',
				label: 'Inventur durchführen',
				desc: 'Laufende Inventur einsehen und Exemplare einscannen — öffnet den Menüpunkt „Inventur"'
			},
			{
				key: 'manage_inventory',
				label: 'Bestand verwalten (Import, Inventur-Abschluss)',
				desc: 'Inventuren starten, abbrechen und abschließen; Fehlbestand bearbeiten (gefunden / endgültig löschen); Littera- und Bestandsimport; Cover-Abgleich'
			}
		]
	},
	{
		category: 'Bestellungen & Kiosk',
		icon: '🛒',
		items: [
			{
				key: 'perform_actions',
				label: 'Kiosk / Terminal bedienen',
				desc: 'Ausleihe, Rückgabe, Scan und Suche am Terminal — die Kernfunktion der Helfer-Rolle (ohne Zugriff auf Schülerlisten oder Mahnwesen)'
			},
			{
				key: 'create_reservations',
				label: 'Klassensatz reservieren',
				desc: 'Erlaubt das Absenden einer Klassensatz-Reservierung im Kollegiums-Portal. Öffnet keine Schülerdaten — das Portal sucht über den öffentlichen OPAC.'
			},
			{
				key: 'manage_vormerkungen',
				label: 'Vormerkungen verwalten (Warteliste)',
				desc: 'Warteliste eines Titels einsehen, Schüler vormerken und Vormerkungen löschen. Zeigt nur Name und Klasse — als enges Theken-Recht für Helfer zuschaltbar, ohne die Schülerdatei zu öffnen.'
			},
			{
				key: 'view_orders',
				label: 'Bestellungen anzeigen',
				desc: 'Bestellungen, Zulauf, Bestellhistorie und Lieferanten einsehen; offene Klassensatz-Reservierungen und Lehrer-Anliegen sehen; Bestell-PDF'
			},
			{
				key: 'create_orders',
				label: 'Bestellungen verwalten',
				desc: 'Bestellungen anlegen, bestätigen und Lieferungen einbuchen; Titel per ISBN anlegen und Signatur setzen; Lieferanten pflegen; Mahn- und Abgänger-Mails versenden; Reservierungen und Anliegen erledigen'
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
		icon: '⚙️',
		items: [
			{
				key: 'view_stats',
				label: 'Statistiken anzeigen',
				desc: 'Ausleih-Statistiken und Auswertungen einsehen (ohne Schüler-Klarnamen)'
			},
			{
				key: 'audit_logs',
				label: 'Sicherheits-Logbuch einsehen',
				desc: 'Allgemeines Logbuch der Systemereignisse einsehen (das Admin-Audit-Log zusätzlich: „Benutzer & Rechte verwalten")'
			},
			{
				key: 'manage_users',
				label: 'Benutzer & Rechte verwalten',
				desc: 'Benutzer und Rechte, alle Systemeinstellungen, Mail-Konfiguration und -Vorlagen, Ausweislayout, Backup-Status, Betriebsbereitschaft, Klassen-Zuordnung, Versetzung, Admin-Audit-Log, DSGVO-Auskunft und endgültiges Löschen aus dem Papierkorb'
			}
		]
	}
];

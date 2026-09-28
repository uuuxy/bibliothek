//go:build ignore

// nachtsicherung führt die Nachtsicherung einmal aus — mit demselben Code wie der Job um 02:30
// (jobs.RunDatabaseBackup), damit die Generalprobe genau das zurückspielt, was im Betrieb
// entsteht. Gebaut und gestartet von scripts/generalprobe/generalprobe.sh:
//
//	go build -o nachtsicherung scripts/generalprobe/nachtsicherung.go
//	env -i PATH=… DATABASE_URL=… BACKUP_DIR=… BACKUP_ENCRYPTION_KEY=… ./nachtsicherung
//
// env -i lässt S3_* und SMTP_* weg: Die Probe lädt nichts hoch und schreibt keine Mail.
package main

import "bibliothek/jobs"

func main() { (&jobs.BackupJob{}).RunDatabaseBackup() }

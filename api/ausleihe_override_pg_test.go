package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/stretchr/testify/assert"
)

func TestOverrideDueDateHandler(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	sid := seedSchueler(t, pool, "S-OVD-1", "Test", "5a")
	alteFrist := time.Date(2025, 9, 1, 23, 59, 59, 0, time.UTC)
	ausleiheID := seedAusleihe(t, pool, sid, "Mathebuch", alteFrist)

	srv := &Server{DB: &db.Database{Pool: pool}}

	t.Run("Fehlende Ausleihe-ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/ausleihen/override", strings.NewReader(`{"faellig_am":"2026-07-31"}`))
		req.SetPathValue("id", "")

		rec := httptest.NewRecorder()
		srv.OverrideDueDateHandler()(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "fehlende Ausleihe-ID")
	})

	t.Run("Erfolgreiches Override (YYYY-MM-DD)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/ausleihen/override", strings.NewReader(`{"faellig_am":"2026-07-31"}`))
		req.SetPathValue("id", ausleiheID)

		rec := httptest.NewRecorder()
		srv.OverrideDueDateHandler()(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		frist := fristVon(t, pool, ausleiheID)
		erwartet := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
		assert.Equal(t, erwartet.Format(time.RFC3339), frist.Format(time.RFC3339))
	})

	t.Run("Ungültiges Datumsformat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/ausleihen/override", strings.NewReader(`{"faellig_am":"31.07.2026"}`))
		req.SetPathValue("id", ausleiheID)

		rec := httptest.NewRecorder()
		srv.OverrideDueDateHandler()(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "ungültiges Datumsformat")
	})

	t.Run("Nicht existierende Ausleihe", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/ausleihen/override", strings.NewReader(`{"faellig_am":"2026-07-31"}`))
		req.SetPathValue("id", "99999999-9999-9999-9999-999999999999")

		rec := httptest.NewRecorder()
		srv.OverrideDueDateHandler()(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Mahnstufe zurücksetzen (Zukunft)", func(t *testing.T) {
		// Erstes Mahndatum setzen
		_, err := pool.Exec(context.Background(), "UPDATE ausleihen SET mahnstufe = 1, letztes_mahndatum = CURRENT_TIMESTAMP WHERE id = $1", ausleiheID)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/ausleihen/override", strings.NewReader(`{"faellig_am":"2099-01-01"}`))
		req.SetPathValue("id", ausleiheID)

		rec := httptest.NewRecorder()
		srv.OverrideDueDateHandler()(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var mahnstufe int
		var letztesMahndatum *time.Time
		err = pool.QueryRow(context.Background(), "SELECT mahnstufe, letztes_mahndatum FROM ausleihen WHERE id = $1", ausleiheID).Scan(&mahnstufe, &letztesMahndatum)
		assert.NoError(t, err)
		assert.Equal(t, 0, mahnstufe)
		assert.Nil(t, letztesMahndatum)
	})

	t.Run("Mahnstufe bleibt bei Vergangenheit", func(t *testing.T) {
		// Mahnstufe setzen
		_, err := pool.Exec(context.Background(), "UPDATE ausleihen SET mahnstufe = 1, letztes_mahndatum = '2023-01-01' WHERE id = $1", ausleiheID)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/ausleihen/override", strings.NewReader(`{"faellig_am":"2023-02-01"}`))
		req.SetPathValue("id", ausleiheID)

		rec := httptest.NewRecorder()
		srv.OverrideDueDateHandler()(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var mahnstufe int
		var letztesMahndatum *time.Time
		err = pool.QueryRow(context.Background(), "SELECT mahnstufe, letztes_mahndatum FROM ausleihen WHERE id = $1", ausleiheID).Scan(&mahnstufe, &letztesMahndatum)
		assert.NoError(t, err)
		assert.Equal(t, 1, mahnstufe)
		assert.NotNil(t, letztesMahndatum)
	})
}

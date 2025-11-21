package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
)

type Alat struct {
	ID        int    `json:"id"`
	Nama      string `json:"nama"`
	Harga     int    `json:"harga"`
	Kategori  string `json:"kategori"`
	FotoURL   string `json:"fotourl"`
}

func connect() (*pgx.Conn, error) {
	url := os.Getenv("DATABASE_URL")
	return pgx.Connect(context.Background(), url)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	conn, err := connect()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer conn.Close(context.Background())

	switch r.Method {

	// ─── GET (ambil semua data) ─────────────────────────────
	case http.MethodGet:
		rows, err := conn.Query(context.Background(),
			`SELECT id, nama, harga, kategori, fotourl FROM alatpertanian ORDER BY id`,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var result []Alat
		for rows.Next() {
			var a Alat
			rows.Scan(&a.ID, &a.Nama, &a.Harga, &a.Kategori, &a.FotoURL)
			result = append(result, a)
		}

		json.NewEncoder(w).Encode(result)
		return

	// ─── POST (tambah data baru) ────────────────────────────
	case http.MethodPost:
		var a Alat
		json.NewDecoder(r.Body).Decode(&a)

		err := conn.QueryRow(context.Background(),
			`INSERT INTO alatpertanian (nama, harga, kategori, fotourl)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			a.Nama, a.Harga, a.Kategori, a.FotoURL,
		).Scan(&a.ID)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(a)
		return

	// ─── PUT (update data) ──────────────────────────────────
	case http.MethodPut:
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.Atoi(idStr)

		var a Alat
		json.NewDecoder(r.Body).Decode(&a)

		_, err := conn.Exec(context.Background(),
			`UPDATE alatpertanian 
			 SET nama=$1, harga=$2, kategori=$3, fotourl=$4 
			 WHERE id=$5`,
			a.Nama, a.Harga, a.Kategori, a.FotoURL, id,
		)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		a.ID = id
		json.NewEncoder(w).Encode(a)
		return

	// ─── DELETE (hapus data) ────────────────────────────────
	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.Atoi(idStr)

		_, err := conn.Exec(context.Background(),
			`DELETE FROM alatpertanian WHERE id=$1`, id,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"deleted": id,
		})
		return
	}

	http.Error(w, "Method Not Allowed", 405)
}

package handler

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
    ID           int     `json:"id"`
    Photo        string  `json:"photo"`
    Name         string  `json:"name"`
    PricePerWeek float64 `json:"price_per_week"`
    Category     string  `json:"category"`
    CreatedAt    string  `json:"created_at,omitempty"`
    UpdatedAt    string  `json:"updated_at,omitempty"`
}

// Main Serverless Entry
func Handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
    if err != nil {
        writeError(w, err, http.StatusInternalServerError)
        return
    }
    defer db.Close()

    path := r.URL.Path
    method := r.Method

    // Routing ala Serverless
    switch {
    case method == "GET" && path == "/api/alatpertanian":
        getAllItems(w, r, db)
    case method == "POST" && path == "/api/alatpertanian":
        createItem(w, r, db)
    case strings.HasPrefix(path, "/api/alatpertanian/"):
        idStr := strings.TrimPrefix(path, "/api/alatpertanian/")
        id, err := strconv.Atoi(idStr)
        if err != nil {
            writeError(w, errors.New("invalid id"), http.StatusBadRequest)
            return
        }

        if method == "GET" {
            getItemByID(w, r, db, id)
        } else if method == "PUT" {
            updateItem(w, r, db, id)
        } else if method == "DELETE" {
            deleteItem(w, r, db, id)
        } else {
            writeError(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
        }
    default:
        writeError(w, errors.New("route not found"), http.StatusNotFound)
    }
}

func getAllItems(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
    rows, err := db.Query(context.Background(), `
        SELECT id, photo, name, price_per_week, category, created_at, updated_at
        FROM alatpertanian ORDER BY id DESC`)
    if err != nil {
        writeError(w, err, http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var items []Item
    for rows.Next() {
        var it Item
        var created, updated time.Time

        err := rows.Scan(&it.ID, &it.Photo, &it.Name, &it.PricePerWeek, &it.Category, &created, &updated)
        if err != nil {
            writeError(w, err, http.StatusInternalServerError)
            return
        }

        it.CreatedAt = created.Format(time.RFC3339)
        it.UpdatedAt = updated.Format(time.RFC3339)

        items = append(items, it)
    }

    json.NewEncoder(w).Encode(items)
}

func getItemByID(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, id int) {
    var it Item
    var created, updated time.Time

    err := db.QueryRow(context.Background(), `
        SELECT id, photo, name, price_per_week, category, created_at, updated_at
        FROM alatpertanian WHERE id=$1`, id).
        Scan(&it.ID, &it.Photo, &it.Name, &it.PricePerWeek, &it.Category, &created, &updated)

    if err != nil {
        writeError(w, errors.New("not found"), http.StatusNotFound)
        return
    }

    it.CreatedAt = created.Format(time.RFC3339)
    it.UpdatedAt = updated.Format(time.RFC3339)

    json.NewEncoder(w).Encode(it)
}

func createItem(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
    var payload Item
    json.NewDecoder(r.Body).Decode(&payload)

    if payload.Name == "" || payload.Photo == "" {
        writeError(w, errors.New("photo & name required"), http.StatusBadRequest)
        return
    }

    var id int
    var created, updated time.Time

    err := db.QueryRow(context.Background(), `
        INSERT INTO alatpertanian (photo, name, price_per_week, category)
        VALUES ($1,$2,$3,$4) RETURNING id, created_at, updated_at`,
        payload.Photo, payload.Name, payload.PricePerWeek, payload.Category,
    ).Scan(&id, &created, &updated)

    if err != nil {
        writeError(w, err, http.StatusInternalServerError)
        return
    }

    payload.ID = id
    payload.CreatedAt = created.Format(time.RFC3339)
    payload.UpdatedAt = updated.Format(time.RFC3339)

    json.NewEncoder(w).Encode(payload)
}

func updateItem(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, id int) {
    var payload Item
    json.NewDecoder(r.Body).Decode(&payload)

    var created, updated time.Time

    err := db.QueryRow(context.Background(), `
        UPDATE alatpertanian
        SET photo=$1, name=$2, price_per_week=$3, category=$4, updated_at=NOW()
        WHERE id=$5 RETURNING id, created_at, updated_at`,
        payload.Photo, payload.Name, payload.PricePerWeek, payload.Category, id,
    ).Scan(&id, &created, &updated)

    if err != nil {
        writeError(w, errors.New("not found"), http.StatusNotFound)
        return
    }

    payload.ID = id
    payload.CreatedAt = created.Format(time.RFC3339)
    payload.UpdatedAt = updated.Format(time.RFC3339)

    json.NewEncoder(w).Encode(payload)
}

func deleteItem(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, id int) {
    _, err := db.Exec(context.Background(), "DELETE FROM alatpertanian WHERE id=$1", id)
    if err != nil {
        writeError(w, errors.New("not found"), http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(map[string]any{"deleted": id})
}

func writeError(w http.ResponseWriter, err error, code int) {
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	_ "github.com/lib/pq"
)

type MarkdownEntry struct {
	ID       int
	Content  string
	Category string
}

func initDB() *sql.DB {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres",psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	createTable := `CREATE TABLE IF NOT EXISTS markdown (
		id SERIAL PRIMARY KEY,
		content TEXT NOT NULL,
		category TEXT NOT NULL
	);`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the table is empty and add a dummy entry if it is
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM markdown").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	if count == 0 {
		_, err = db.Exec("INSERT INTO markdown (content, category) VALUES ($1, $2)", "# Hello, World!", "default")
		if err != nil {
			log.Fatal(err)
		}
	}

	return db
}

func getRandomMarkdown(db *sql.DB, category string) (MarkdownEntry, error) {
	row := db.QueryRow("SELECT id, content, category FROM markdown WHERE category=$1 ORDER BY RANDOM() LIMIT 1", category)

	var entry MarkdownEntry
	err := row.Scan(&entry.ID, &entry.Content, &entry.Category)
	return entry, err
}

func addMarkdown(db *sql.DB, content, category string) error {
	_, err := db.Exec("INSERT INTO markdown (content, category) VALUES ($1, $2)", content, category)
	return err
}

func markdownToHTML(md string) string {
	unsafeHTML := blackfriday.Run([]byte(md))
	safeHTML := bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML)
	return string(safeHTML)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	db := initDB()
	defer db.Close()

	r := gin.Default()

	// Define a function map to use custom functions in templates
	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	r.SetFuncMap(funcMap)
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		category := c.DefaultQuery("category", "default")
		entry, err := getRandomMarkdown(db, category)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error fetching markdown: %v", err)
			return
		}

		htmlContent := markdownToHTML(entry.Content)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"HTMLContent": htmlContent,
			"Category":    category,
		})
	})

	r.POST("/shuffle", func(c *gin.Context) {
		category := c.PostForm("category")
		entry, err := getRandomMarkdown(db, category)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error fetching markdown: %v", err)
			return
		}

		htmlContent := markdownToHTML(entry.Content)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"HTMLContent": htmlContent,
			"Category":    category,
		})
	})

	r.GET("/new", func(c *gin.Context) {
		c.HTML(http.StatusOK, "new.html", nil)
	})

	r.POST("/new", func(c *gin.Context) {
		content := c.PostForm("content")
		category := c.PostForm("category")
		if content == "" || category == "" {
			c.String(http.StatusBadRequest, "Content and category cannot be empty")
			return
		}

		err := addMarkdown(db, content, category)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error adding markdown: %v", err)
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	r.Run(":8080")
}


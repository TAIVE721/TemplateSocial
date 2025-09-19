// cmd/api/main.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"GopherSocial/internal/auth"
	"GopherSocial/internal/db"
	"GopherSocial/internal/mailer"
	"GopherSocial/internal/store"

	"GopherSocial/internal/store/cache"

	"github.com/go-chi/chi/v5"
)

const version = "1.0.0"

type config struct {
	addr string
	env  string
	db   struct { // Configuración de la DB
		addr string
	}
	auth struct { // Configuración de auth
		secret string
	}
	redis struct { // Configuración de Redis
		addr string
	}
}

type application struct {
	config        config
	db            *sql.DB
	store         store.Storage
	authenticator *auth.JWTAuthenticator
	cacheStorage  cache.Storage
	mailer        mailer.Client
	logger        *log.Logger
}

func main() {
	var cfg config
	cfg.addr = ":8080"
	cfg.env = "development"
	cfg.db.addr = "postgres://admin:adminpassword@localhost/socialnetwork?sslmode=disable"
	cfg.auth.secret = "una-clave-super-secreta-kamen-rider"
	cfg.redis.addr = "localhost:6379"

	rdb := cache.NewRedisClient(cfg.redis.addr, "", 0)
	fmt.Println("¡Conexión a Redis exitosa!")

	cacheStorage := cache.NewRedisStorage(rdb)

	authenticator := auth.NewJWTAuthenticator(cfg.auth.secret, "gophersocial", "gophersocial")

	db, err := db.New(cfg.db.addr)
	if err != nil {
		log.Fatalf("No se pudo conectar a la base de datos: %v", err)
	}
	defer db.Close() // Aseguramos que la conexión se cierre al terminar.
	fmt.Println("¡Conexión a la base de datos exitosa!")

	storage := store.NewStorage(db)

	mailerClient := mailer.MailtrapClient{
		Host:     "sandbox.smtp.mailtrap.io",
		Port:     2525,
		Username: "TU_USUARIO_DE_MAILTRAP",
		Password: "TU_PASSWORD_DE_MAILTRAP",
		From:     "no-reply@gophersocial.net",
	}

	// Crea una instancia del logger que escribirá en la consola.
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	app := &application{
		config:        cfg,
		db:            db,
		store:         storage,
		authenticator: authenticator,
		cacheStorage:  cacheStorage,
		mailer:        mailerClient,
		logger:        logger,
	}

	srv := &http.Server{
		Addr:    cfg.addr,
		Handler: app.mount(), // ¡Aquí montaremos nuestras rutas!
	}

	fmt.Printf("Servidor escuchando en %s en modo %s\n", cfg.addr, cfg.env)
	log.Fatal(srv.ListenAndServe())
}

// mount registrará y devolverá todas las rutas de nuestra aplicación.
func (app *application) mount() http.Handler {
	// Creamos una nueva instancia de Chi router.
	r := chi.NewRouter()

	r.Get("/v1/health", app.healthCheckHandler)
	r.Post("/v1/authentication/user", app.registerUserHandler)
	r.Post("/v1/authentication/token", app.createTokenHandler)
	r.Put("/v1/users/activate/{token}", app.activateUserHandler)

	r.Group(func(r chi.Router) {
		r.Use(app.AuthTokenMiddleware) // ¡Aplicamos el guardián!

		// Añadimos una ruta de prueba para verificar que el middleware funciona
		r.Get("/v1/test-protected", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(userCtxKey)
			app.jsonResponse(w, http.StatusOK, map[string]any{"message": "Acceso concedido", "user_id": userID})
		})

		r.Post("/v1/posts", app.createPostHandler)
		r.Get("/v1/posts/{postID}", app.getPostHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(app.AuthTokenMiddleware)

		r.Post("/v1/posts", app.createPostHandler)

		// Creamos un sub-grupo para rutas que operan sobre un post específico
		r.Route("/v1/posts/{postID}", func(r chi.Router) {
			r.Use(app.postsContextMiddleware) // Carga el post en el contexto
			r.Get("/", app.getPostHandler)

			// Solo el dueño puede actualizar o borrar
			r.Group(func(r chi.Router) {
				r.Use(app.checkPostOwnership)
				r.Patch("/", app.updatePostHandler)
				r.Delete("/", app.deletePostHandler)
			})
		})
	})

	return r
}

// healthCheckHandler es nuestro primer handler.
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Preparamos una respuesta simple en formato JSON.
	data := map[string]string{
		"status":  "ok",
		"env":     app.config.env,
		"version": version,
	}

	err := app.jsonResponse(w, http.StatusOK, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "El servidor encontró un problema", http.StatusInternalServerError)
	}
}

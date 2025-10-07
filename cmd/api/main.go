// cmd/api/main.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time" // Asegúrate de tener este import

	"GopherSocial/internal/auth"
	"GopherSocial/internal/db"
	"GopherSocial/internal/env"
	"GopherSocial/internal/mailer"
	"GopherSocial/internal/ratelimiter" // <-- Importa el nuevo paquete
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
	rateLimiter ratelimiter.Config
}

type application struct {
	config        config
	db            *sql.DB
	store         store.Storage
	authenticator *auth.JWTAuthenticator
	cacheStorage  cache.Storage
	mailer        mailer.Client
	rateLimiter   ratelimiter.Limiter
	logger        *log.Logger
}

func main() {
	var cfg config
	cfg.addr = env.GetString("ADDR", ":8080")
	cfg.env = env.GetString("ENV", "development")
	cfg.db.addr = env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost/socialnetwork?sslmode=disable")
	cfg.auth.secret = env.GetString("AUTH_TOKEN_SECRET", "una-clave-super-secreta-kamen-rider")
	cfg.redis.addr = env.GetString("REDIS_ADDR", "localhost:6379")

	cfg.rateLimiter = ratelimiter.Config{
		RequestsPerTimeFrame: env.GetInt("RATELIMITER_REQUESTS", 20),
		TimeFrame:            time.Second * 60, // 20 peticiones por minuto
		Enabled:              env.GetBool("RATELIMITER_ENABLED", true),
	}

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
		Username: "96a39335c1f4ba",
		Password: "8daf541a756e21",
		From:     "no-reply@gophersocial.net",
	}

	rateLimiter := ratelimiter.NewFixedWindowLimiter(
		cfg.rateLimiter.RequestsPerTimeFrame,
		cfg.rateLimiter.TimeFrame,
	)

	// Crea una instancia del logger que escribirá en la consola.
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	app := &application{
		config:        cfg,
		db:            db,
		store:         storage,
		authenticator: authenticator,
		cacheStorage:  cacheStorage,
		mailer:        mailerClient,
		rateLimiter:   rateLimiter,
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

	if app.config.rateLimiter.Enabled {
		r.Use(app.RateLimiterMiddleware)
	}

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

		r.Put("/v1/users/{userID}/follow", app.followUserHandler)
		r.Put("/v1/users/{userID}/unfollow", app.unfollowUserHandler)
		r.Get("/v1/users/feed", app.getUserFeedHandler)

	})

	r.Group(func(r chi.Router) {
		r.Use(app.AuthTokenMiddleware)

		r.Post("/v1/posts", app.createPostHandler)

		// Creamos un sub-grupo para rutas que operan sobre un post específico
		r.Route("/v1/posts/{postID}", func(r chi.Router) {
			r.Use(app.postsContextMiddleware)
			r.Get("/", app.getPostHandler)

			// Solo el dueño o un moderador (nivel >= 2) puede actualizar
			r.With(app.checkPermission(2)).Patch("/", app.updatePostHandler)
			// Solo el dueño o un admin (nivel >= 3) puede borrar
			r.With(app.checkPermission(3)).Delete("/", app.deletePostHandler)
			r.Post("/comments", app.createCommentHandler)
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

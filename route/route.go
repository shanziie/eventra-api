package route

import (
	"context"
	"time"

	"eventra-api/app/service"
	"eventra-api/helper"
	"eventra-api/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Setup mendaftarkan semua route ke aplikasi Fiber.
func Setup(
	app *fiber.App,
	db *pgxpool.Pool,
	authService *service.AuthService,
	userService *service.UserService,
	categoryService *service.CategoryService,
	eventService *service.EventService,
	registrationService *service.RegistrationService,
	jwtManager *helper.JWTManager,
) {
	api := app.Group("/api/v1")

	// Endpoint health check
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("database tidak dapat dijangkau")
		}
		return helper.Success(c, "server berjalan normal", fiber.Map{"status": "ok"})
	})

	// Grup Auth (Modul 5, PRD 8.2 endpoint 2 - 6)
	auth := api.Group("/auth")
	auth.Post("/register", middleware.RequireJSON, authService.Register)
	auth.Post("/login", middleware.RequireJSON, middleware.LoginRateLimiter(), authService.Login)
	auth.Post("/refresh", middleware.RequireJSON, authService.Refresh)
	auth.Post("/logout", middleware.RequireJSON, authService.Logout)
	auth.Get("/me", middleware.RequireAuth(jwtManager), authService.Me)

	// Grup Users (Modul 2, 3, 6, PRD 8.2 endpoint 7 - 12, 35)
	users := api.Group("/users", middleware.RequireAuth(jwtManager))
	users.Get("", middleware.RequirePermission("user:list"), userService.GetUsers)
	users.Get("/:id", userService.GetUserByID)
	users.Put("/:id", middleware.RequireJSON, userService.PutUser)
	users.Patch("/:id", middleware.RequireJSON, userService.PatchUser)
	users.Delete("/:id", middleware.RequirePermission("user:delete"), userService.DeleteUser)
	users.Patch("/:id/role", middleware.RequireJSON, middleware.RequirePermission("role:assign"), userService.AssignRole)
	users.Get("/:id/registrations", registrationService.GetRegistrationsByUser) // #35

	// Grup Categories (PRD 8.2 endpoint 13 - 16)
	categories := api.Group("/categories", middleware.RequireAuth(jwtManager))
	categories.Get("", categoryService.GetCategories)
	categories.Post("", middleware.RequireJSON, middleware.RequirePermission("category:create"), categoryService.CreateCategory)
	categories.Put("/:id", middleware.RequireJSON, middleware.RequirePermission("category:update"), categoryService.UpdateCategory)
	categories.Delete("/:id", middleware.RequirePermission("category:delete"), categoryService.DeleteCategory)

	// Grup Events & Ticket Types (PRD 8.2 endpoint 17 - 27)
	events := api.Group("/events")
	events.Get("", eventService.GetEvents)                                                                  // #17
	events.Get("/:id", eventService.GetEventByID)                                                           // #18
	events.Post("", middleware.RequireAuth(jwtManager), middleware.RequirePermission("event:create"), middleware.RequireJSON, eventService.CreateEvent) // #19
	events.Put("/:id", middleware.RequireAuth(jwtManager), middleware.RequireJSON, eventService.UpdateEvent) // #20
	events.Patch("/:id", middleware.RequireAuth(jwtManager), middleware.RequireJSON, eventService.PatchEvent) // #21
	events.Delete("/:id", middleware.RequireAuth(jwtManager), eventService.DeleteEvent)                     // #22
	events.Patch("/:id/status", middleware.RequireAuth(jwtManager), middleware.RequireJSON, eventService.UpdateEventStatus) // #23

	events.Get("/:id/ticket-types", eventService.GetTicketTypes)                                            // #24
	events.Post("/:id/ticket-types", middleware.RequireAuth(jwtManager), middleware.RequireJSON, eventService.CreateTicketType) // #25

	// Registrations under events (PRD 8.2 endpoint 28, 29)
	events.Post("/:id/registrations", middleware.RequireAuth(jwtManager), middleware.RequirePermission("registration:create"), middleware.RequireJSON, registrationService.CreateRegistration) // #28
	events.Get("/:id/registrations", middleware.RequireAuth(jwtManager), registrationService.GetRegistrationsByEvent) // #29

	ticketTypes := api.Group("/ticket-types", middleware.RequireAuth(jwtManager))
	ticketTypes.Put("/:id", middleware.RequireJSON, eventService.UpdateTicketType)                          // #26
	ticketTypes.Delete("/:id", eventService.DeleteTicketType)                                               // #27

	// Registrations standalone endpoints (PRD 8.2 endpoint 30 - 34)
	registrations := api.Group("/registrations", middleware.RequireAuth(jwtManager))
	registrations.Get("/:id", registrationService.GetRegistrationByID)                                      // #30
	registrations.Post("/:id/payment", middleware.RequireJSON, registrationService.SubmitPayment)             // #31
	registrations.Patch("/:id/payment", middleware.RequireJSON, registrationService.VerifyPayment)            // #32
	registrations.Delete("/:id", registrationService.CancelRegistration)                                    // #33
	registrations.Post("/:id/check-in", registrationService.CheckInRegistration)                            // #34

	// Tangkap semua route yang tidak terdaftar — harus di posisi paling akhir
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})
}

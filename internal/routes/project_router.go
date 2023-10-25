package routes

import (
	"github.com/gofiber/fiber/v2"

	"www.github.com/ic-ETITE-24/icetite-24-backend/internal/controllers"
	"www.github.com/ic-ETITE-24/icetite-24-backend/internal/middleware"
)

func ProjectsRoutes(incomingRoutes *fiber.App) {
	projectRoutes := incomingRoutes.Group("/project")
	projectRoutes.Post("/testing", middleware.VerifyAccessToken, controllers.CreateTeam)
	projectRoutes.Post("/finalise", middleware.VerifyAccessToken, controllers.FinaliseProject)
	projectRoutes.Post("/create", middleware.VerifyAccessToken, controllers.CreateProject)
	projectRoutes.Get("/get", middleware.VerifyAccessToken, controllers.GetProject)
	projectRoutes.Delete("/delete", middleware.VerifyAccessToken, controllers.DeleteProject)
	projectRoutes.Get("/getall", middleware.VerifyAccessToken, controllers.GetAllProject)
	projectRoutes.Post("/update", middleware.VerifyAccessToken, controllers.UpdateProject)
}

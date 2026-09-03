// Package handler implements the HTTP layer for the admin domain.
// It depends only on the service.Service interface — never on the repo or DB.
package handler

import (
	"net/http"
	"strconv"

	adminservice "slipwise/internal/admin/service"

	"github.com/labstack/echo/v5"
)

// AdminHandler handles all /v1/admin/* HTTP requests.
type AdminHandler struct {
	svc adminservice.Service
}

// New returns a new AdminHandler wired to the given service.
func New(svc adminservice.Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// GetDashboardStats godoc
// @Summary      Admin — Dashboard Stats
// @Description  Returns aggregated app-wide metrics (total users, tickets won/lost/pending, etc.)
// @Tags         admin
// @Produce      json
// @Success      200  {object}  model.DashboardStatsResponse
// @Failure      401  {object}  apitypes.ErrorResponse
// @Failure      403  {object}  apitypes.ErrorResponse
// @Failure      500  {object}  apitypes.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/dashboard [get]
func (h *AdminHandler) GetDashboardStats(c *echo.Context) error {
	stats, err := h.svc.GetDashboardStats(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch dashboard stats"})
	}
	return c.JSON(http.StatusOK, stats)
}

// GetUsers godoc
// @Summary      Admin — List Users
// @Description  Returns a paginated list of all registered users.
// @Tags         admin
// @Produce      json
// @Param        page   query  int  false  "Page number (default: 1)"
// @Param        limit  query  int  false  "Items per page (default: 20, max: 100)"
// @Success      200  {object}  model.PaginatedAdminUsersResponse
// @Failure      401  {object}  apitypes.ErrorResponse
// @Failure      403  {object}  apitypes.ErrorResponse
// @Failure      500  {object}  apitypes.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/users [get]
func (h *AdminHandler) GetUsers(c *echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	resp, err := h.svc.GetUsers(c.Request().Context(), page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch users"})
	}
	return c.JSON(http.StatusOK, resp)
}

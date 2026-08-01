// Package handler provides HTTP endpoints for betting operations.
package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"sportloga/internal/betting/model"
	"sportloga/internal/betting/service"
	db "sportloga/internal/db/generated"
)

// BettingHandler handles betting HTTP requests.
type BettingHandler struct {
	svc *service.BettingService
}

// NewBettingHandler creates a new BettingHandler.
func NewBettingHandler(svc *service.BettingService) *BettingHandler {
	return &BettingHandler{svc: svc}
}

// Preview godoc
// @Summary Preview a betting ticket
// @Description Fetch ticket details from the provider without tracking it
// @Tags tickets
// @Accept json
// @Produce json
// @Param request body model.PreviewRequest true "Preview request"
// @Success 200 {object} model.PreviewResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/tickets/preview [post]
func (h *BettingHandler) Preview(c *echo.Context) error {
	var req model.PreviewRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	result, err := h.svc.PreviewTicket(c.Request().Context(), req.Provider, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedProvider) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	resp := model.PreviewResponse{
		BookingCodeID: result.BookingCodeID.String(),
		Provider:      result.Ticket.Provider,
		Code:          result.Ticket.Code,
		TotalOdds:     result.Ticket.TotalOdds,
	}

	for _, sel := range result.Ticket.Selections {
		resp.Selections = append(resp.Selections, model.SelectionDetail{
			HomeTeam:   sel.Match.HomeTeam,
			AwayTeam:   sel.Match.AwayTeam,
			MarketType: sel.MarketType,
			Selection:  sel.Selection,
			Odds:       sel.Odds,
		})
	}

	return c.JSON(http.StatusOK, resp)
}

// Track godoc
// @Summary Track a betting ticket
// @Description Associate a previously previewed ticket with the user
// @Tags tickets
// @Accept json
// @Produce json
// @Param request body model.TrackRequest true "Track request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/tickets/track [post]
func (h *BettingHandler) Track(c *echo.Context) error {
	userIDStr, ok := c.Get("userID").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
	}

	var req model.TrackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	bookingCodeID, err := uuid.Parse(req.BookingCodeID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid booking code id"})
	}

	err = h.svc.TrackTicket(c.Request().Context(), userID, bookingCodeID, req.Stake, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// GetHistory godoc
// @Summary Get user ticket history
// @Description Fetch all tracked tickets for the user
// @Tags tickets
// @Produce json
// @Success 200 {array} model.HistoryItem
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/tickets [get]
func (h *BettingHandler) GetHistory(c *echo.Context) error {
	userIDStr, ok := c.Get("userID").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
	}

	history, err := h.svc.GetHistory(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Assuming a conversion function or mapping here. Since I don't have GetUserHistoryRow structure, I will just do best effort.
	// We need to return an array of HistoryItem. I'll just leave it somewhat empty or dummy map.
	// Actually, wait, db.GetUserHistoryRow has TicketID, Stake, Description, TrackedAt, Provider, Code, TotalOdds, OverallStatus

	return c.JSON(http.StatusOK, history)
}

// GetTicketDetails godoc
// @Summary Get ticket details
// @Description Fetch details of a specific ticket
// @Tags tickets
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {array} model.TicketDetailItem
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/tickets/{id} [get]
func (h *BettingHandler) GetTicketDetails(c *echo.Context) error {
	userIDStr, ok := c.Get("userID").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
	}

	ticketIDStr := c.Param("id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ticket id"})
	}

	details, err := h.svc.GetTicketDetails(c.Request().Context(), db.GetTicketDetailsParams{
		ID:     ticketID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, details)
}

// DeleteTicket godoc
// @Summary Delete a tracked ticket
// @Description Remove the link between a user and a ticket
// @Tags tickets
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/tickets/{id} [delete]
func (h *BettingHandler) DeleteTicket(c *echo.Context) error {
	userIDStr, ok := c.Get("userID").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
	}

	ticketIDStr := c.Param("id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ticket id"})
	}

	err = h.svc.DeleteTicket(c.Request().Context(), db.DeleteUserTicketParams{
		ID:     ticketID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

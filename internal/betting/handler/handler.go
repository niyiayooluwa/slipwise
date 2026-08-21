// Package handler provides HTTP endpoints for betting operations.
package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"slipwise/internal/apitypes"
	"slipwise/internal/betting/model"
	"slipwise/internal/betting/service"
	db "slipwise/internal/db/generated"
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
// @Failure 400 {object} apitypes.ErrorResponse
// @Failure 500 {object} apitypes.ErrorResponse
// @Router /v1/tickets/preview [post]
func (h *BettingHandler) Preview(c *echo.Context) error {
	var req model.PreviewRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid request body"})
	}

	result, err := h.svc.PreviewTicket(c.Request().Context(), req.Provider, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedProvider) {
			return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: err.Error()})
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
// @Success 200 {object} apitypes.MessageResponse
// @Failure 400 {object} apitypes.ErrorResponse
// @Failure 500 {object} apitypes.ErrorResponse
// @Security BearerAuth
// @Router /v1/tickets/track [post]
func (h *BettingHandler) Track(c *echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: "unauthorized"})
	}

	var req model.TrackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid request body"})
	}

	bookingCodeID, err := uuid.Parse(req.BookingCodeID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid booking code id"})
	}

	err = h.svc.TrackTicket(c.Request().Context(), userID, bookingCodeID, req.Stake, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apitypes.MessageResponse{Message: "success"})
}

// GetHistory godoc
// @Summary Get user ticket history
// @Description Fetch all tracked tickets for the user
// @Tags tickets
// @Produce json
// @Success 200 {array} model.HistoryItem
// @Failure 401 {object} apitypes.ErrorResponse
// @Failure 500 {object} apitypes.ErrorResponse
// @Security BearerAuth
// @Router /v1/tickets [get]
func (h *BettingHandler) GetHistory(c *echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: "unauthorized"})
	}

	history, err := h.svc.GetHistory(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: err.Error()})
	}

	var resp []model.HistoryItem
	for _, row := range history {
		stake, _ := row.Stake.Float64Value()
		var stakePtr *float64
		if stake.Valid {
			v := stake.Float64
			stakePtr = &v
		}
		totalOdds, _ := row.TotalOdds.Float64Value()

		resp = append(resp, model.HistoryItem{
			TicketID:      row.TicketID.String(),
			Stake:         stakePtr,
			Description:   row.Description,
			TrackedAt:     row.TrackedAt.Time.Format(time.RFC3339),
			Provider:      row.Provider,
			Code:          row.Code,
			TotalOdds:     totalOdds.Float64,
			OverallStatus: row.OverallStatus,
		})
	}

	if resp == nil {
		resp = []model.HistoryItem{}
	}

	return c.JSON(http.StatusOK, resp)
}

// GetTicketDetails godoc
// @Summary Get ticket details
// @Description Fetch details of a specific ticket
// @Tags tickets
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {array} model.TicketDetailItem
// @Failure 400 {object} apitypes.ErrorResponse
// @Failure 401 {object} apitypes.ErrorResponse
// @Failure 500 {object} apitypes.ErrorResponse
// @Security BearerAuth
// @Router /v1/tickets/{id} [get]
func (h *BettingHandler) GetTicketDetails(c *echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: "unauthorized"})
	}

	ticketIDStr := c.Param("id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid ticket id"})
	}

	details, err := h.svc.GetTicketDetails(c.Request().Context(), db.GetTicketDetailsParams{
		ID:     ticketID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: err.Error()})
	}

	var resp []model.TicketDetailItem
	for _, row := range details {
		odds, _ := row.Odds.Float64Value()
		var marketSpec string
		if row.MarketSpec != nil {
			marketSpec = *row.MarketSpec
		}

		resp = append(resp, model.TicketDetailItem{
			SelectionID:     row.SelectionID.String(),
			MarketType:      row.MarketType,
			MarketSpec:      marketSpec,
			Selection:       row.Selection,
			Odds:            odds.Float64,
			SelectionStatus: row.SelectionStatus,
			HomeTeam:        row.HomeTeam,
			AwayTeam:        row.AwayTeam,
			StartTime:       row.StartTime.Time.Format(time.RFC3339),
			MatchStatus:     row.MatchStatus,
		})
	}

	if resp == nil {
		resp = []model.TicketDetailItem{}
	}

	return c.JSON(http.StatusOK, resp)
}

// DeleteTicket godoc
// @Summary Delete a tracked ticket
// @Description Remove the link between a user and a ticket
// @Tags tickets
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {object} apitypes.MessageResponse
// @Failure 400 {object} apitypes.ErrorResponse
// @Failure 401 {object} apitypes.ErrorResponse
// @Failure 500 {object} apitypes.ErrorResponse
// @Security BearerAuth
// @Router /v1/tickets/{id} [delete]
func (h *BettingHandler) DeleteTicket(c *echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: "unauthorized"})
	}

	ticketIDStr := c.Param("id")
	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid ticket id"})
	}

	err = h.svc.DeleteTicket(c.Request().Context(), db.DeleteUserTicketParams{
		ID:     ticketID,
		UserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, apitypes.MessageResponse{Message: "success"})
}

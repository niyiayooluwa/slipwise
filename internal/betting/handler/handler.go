// Package handler provides HTTP endpoints for betting operations.
package handler

import (
	"errors"
	"net/http"
	"strconv"
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
// @Summary      Preview a betting ticket from a provider
// @Description  Fetches the latest odds and selections for a given booking code
// @Description  from the specified provider (e.g. SPORTYBET) directly.
// @Description  This does NOT save the ticket to the user's history, it merely
// @Description  resolves the code and allows them to preview it before tracking.
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        request body model.PreviewRequest true "Provider name and booking code"
// @Success      200 {object} model.PreviewResponse "Successfully retrieved ticket details"
// @Failure      400 {object} apitypes.ErrorResponse "Invalid provider, missing fields, or invalid code"
// @Failure      500 {object} apitypes.ErrorResponse "External provider API failure or internal error"
// @Router       /v1/tickets/preview [post]
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
		var specStr string
		if sel.MarketSpec != nil {
			specStr = *sel.MarketSpec
		}
		resp.Selections = append(resp.Selections, model.SelectionDetail{
			HomeTeam:   sel.Match.HomeTeam,
			AwayTeam:   sel.Match.AwayTeam,
			MarketType: sel.MarketType,
			MarketSpec: specStr,
			Selection:  sel.Selection,
			Odds:       sel.Odds,
		})
	}

	return c.JSON(http.StatusOK, resp)
}

// Track godoc
// @Summary      Track a betting ticket
// @Description  Links a previously previewed ticket (via booking_code_id) to the authenticated user.
// @Description  Optionally includes a stake and a description (e.g., "Weekend acca").
// @Description  Once tracked, the system will actively poll this ticket for live status updates.
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        request body model.TrackRequest true "Booking code ID, stake, and optional description"
// @Success      200 {object} apitypes.MessageResponse "Ticket successfully tracked"
// @Failure      400 {object} apitypes.ErrorResponse "Invalid booking_code_id or missing fields"
// @Failure      401 {object} apitypes.ErrorResponse "Unauthorized - missing or invalid token"
// @Failure      500 {object} apitypes.ErrorResponse "Database insertion error"
// @Security     BearerAuth
// @Router       /v1/tickets/track [post]
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
// @Summary      Get paginated user ticket history
// @Description  Retrieves a paginated list of all betting tickets tracked by the authenticated user.
// @Description  The tickets are ordered by when they were tracked (most recent first).
// @Description  Provides high-level details like the total odds and overall ticket status.
// @Tags         tickets
// @Produce      json
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20, max 100)"
// @Success      200 {object} model.PaginatedHistoryResponse "Paginated history list"
// @Failure      401 {object} apitypes.ErrorResponse "Unauthorized - missing or invalid token"
// @Failure      500 {object} apitypes.ErrorResponse "Database retrieval error"
// @Security     BearerAuth
// @Router       /v1/tickets [get]
func (h *BettingHandler) GetHistory(c *echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: "unauthorized"})
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	history, total, err := h.svc.GetHistory(c.Request().Context(), userID, int32(limit), int32(offset))
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

	return c.JSON(http.StatusOK, model.PaginatedHistoryResponse{
		Data: resp,
		Meta: model.PaginationMeta{
			Total:   total,
			Page:    int32(page),
			Limit:   int32(limit),
			HasNext: int64(page*limit) < total,
		},
	})
}

// GetTicketDetails godoc
// @Summary      Get ticket details
// @Description  Fetches the deep details of a specific ticket tracked by the user.
// @Description  This includes a breakdown of every single match/selection on the ticket,
// @Description  including the live status of the matches and the current odds.
// @Tags         tickets
// @Produce      json
// @Param        id path string true "UUID of the tracked user_ticket"
// @Success      200 {array} model.TicketDetailItem "Detailed breakdown of the ticket's selections"
// @Failure      400 {object} apitypes.ErrorResponse "Invalid ticket UUID format"
// @Failure      401 {object} apitypes.ErrorResponse "Unauthorized - missing or invalid token"
// @Failure      500 {object} apitypes.ErrorResponse "Database retrieval error"
// @Security     BearerAuth
// @Router       /v1/tickets/{id} [get]
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
// @Summary      Delete a tracked ticket
// @Description  Removes the link between the authenticated user and a specific ticket.
// @Description  The underlying booking_code is not immediately deleted so that other users
// @Description  tracking the same code are unaffected. A background job later sweeps orphaned codes.
// @Tags         tickets
// @Produce      json
// @Param        id path string true "UUID of the tracked user_ticket"
// @Success      200 {object} apitypes.MessageResponse "Ticket successfully deleted"
// @Failure      400 {object} apitypes.ErrorResponse "Invalid ticket UUID format"
// @Failure      401 {object} apitypes.ErrorResponse "Unauthorized - missing or invalid token"
// @Failure      500 {object} apitypes.ErrorResponse "Database deletion error"
// @Security     BearerAuth
// @Router       /v1/tickets/{id} [delete]
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

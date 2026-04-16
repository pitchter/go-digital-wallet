package handler

import (
	"net/http"

	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (a *API) topUp(c *gin.Context) {
	request, ok := bindJSON[dto.TopUpRequest](c)
	if !ok {
		return
	}

	response, statusCode, err := a.transactions.TopUp(backgroundContext(c), request, c.GetHeader("Idempotency-Key"))
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, statusCode, response)
}

func (a *API) transfer(c *gin.Context) {
	request, ok := bindJSON[dto.TransferRequest](c)
	if !ok {
		return
	}

	response, statusCode, err := a.transactions.Transfer(backgroundContext(c), request, c.GetHeader("Idempotency-Key"))
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, statusCode, response)
}

func (a *API) getTransaction(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	response, err := a.transactions.Get(backgroundContext(c), id)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (a *API) listTransactions(c *gin.Context) {
	page, limit := parsePagination(c)
	filters := repository.TransactionFilters{}

	if walletID := c.Query("wallet_id"); walletID != "" {
		parsedWalletID, err := uuid.Parse(walletID)
		if err != nil {
			writeJSON(c, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "bad_request",
				Message: "request payload is invalid",
			})
			return
		}
		filters.WalletID = &parsedWalletID
	}
	if txType := c.Query("type"); txType != "" {
		parsedType := model.TransactionType(txType)
		filters.Type = &parsedType
	}
	if status := c.Query("status"); status != "" {
		parsedStatus := model.TransactionStatus(status)
		filters.Status = &parsedStatus
	}

	response, err := a.transactions.List(backgroundContext(c), page, limit, filters)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (a *API) reverseTransaction(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	response, statusCode, err := a.transactions.Reverse(backgroundContext(c), id, c.GetHeader("Idempotency-Key"))
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, statusCode, response)
}

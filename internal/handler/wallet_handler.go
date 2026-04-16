package handler

import (
	"net/http"

	"go-digital-wallet/internal/dto"

	"github.com/gin-gonic/gin"
)

func (a *API) createWallet(c *gin.Context) {
	request, ok := bindJSON[dto.CreateWalletRequest](c)
	if !ok {
		return
	}

	response, err := a.wallets.Create(backgroundContext(c), request)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, response)
}

func (a *API) getWallet(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	response, err := a.wallets.Get(backgroundContext(c), id)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (a *API) getWalletBalance(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	response, err := a.wallets.GetBalance(backgroundContext(c), id)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (a *API) updateWalletStatus(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	request, ok := bindJSON[dto.UpdateWalletStatusRequest](c)
	if !ok {
		return
	}

	response, err := a.wallets.UpdateStatus(backgroundContext(c), id, request.Status)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

package handler

import (
	"net/http"

	"go-digital-wallet/internal/dto"

	"github.com/gin-gonic/gin"
)

func (a *API) createUser(c *gin.Context) {
	request, ok := bindJSON[dto.CreateUserRequest](c)
	if !ok {
		return
	}

	response, err := a.users.Create(backgroundContext(c), request)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, response)
}

func (a *API) listUsers(c *gin.Context) {
	page, limit := parsePagination(c)

	response, err := a.users.List(backgroundContext(c), page, limit)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (a *API) getUser(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	response, err := a.users.Get(backgroundContext(c), id)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (a *API) updateUser(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	request, ok := bindJSON[dto.UpdateUserRequest](c)
	if !ok {
		return
	}

	response, err := a.users.Update(backgroundContext(c), id, request)
	if err != nil {
		writeError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, response)
}

func (a *API) deleteUser(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := a.users.Delete(backgroundContext(c), id); err != nil {
		writeError(c, err)
		return
	}

	noContent(c)
}

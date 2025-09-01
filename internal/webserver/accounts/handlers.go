package accounts

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/austinwofford/account-management/internal/database"
	"github.com/austinwofford/account-management/internal/webserver/httputils"
	"github.com/go-chi/chi/v5"
)

type accountsHandler struct {
	db *database.DB

	http.Handler
}

type HandlerDeps struct {
	DB *database.DB
}

func NewHandler(deps HandlerDeps) http.Handler {
	mux := chi.NewMux()

	h := accountsHandler{
		db: deps.DB,
	}

	mux.Post("/register", h.register)
	mux.Post("/login", h.login)
	mux.Post("/refresh", h.refresh)
	mux.Post("/logout", h.logout)

	h.Handler = mux

	return h
}

type registerRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponseBody struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

func (h *accountsHandler) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqBody registerRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		slog.ErrorContext(ctx, "error decoding register request body", "error", err.Error())
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "error reading request body",
			StatusCode: http.StatusBadRequest,
		})

		return
	}

	// check if user already exists, if so return err
	// add user to db
	// return user ID
	httputils.WriteJSONResponse(w, r, http.StatusCreated, registerResponseBody{
		Message: "Account created successfully",
		UserID:  "user123",
	})
}

type loginRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// newTokenResponseBody is used for both login and refresh responses
type newTokenResponseBody struct {
	Message      string `json:"message"`
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (h *accountsHandler) login(w http.ResponseWriter, r *http.Request) {
	// read body
	// check email and password
	// create a refresh token and persist in the db
	// return the refresh token and access token
}

type refreshRequestBody struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *accountsHandler) refresh(w http.ResponseWriter, r *http.Request) {
	// validate the refresh token
	// if validation fails, return a 401
	// if refresh token is valid, create a new refresh token and a new access token?
}

func (h *accountsHandler) logout(w http.ResponseWriter, r *http.Request) {
	// revoke the refresh token (forces user to log in again)
}

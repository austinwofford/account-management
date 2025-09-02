package accounts

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/austinwofford/account-management/internal/database"
	"github.com/austinwofford/account-management/internal/service/auth"
	"github.com/austinwofford/account-management/internal/webserver/httputils"
	"github.com/go-chi/chi/v5"
)

type accountsHandler struct {
	db         *database.DB
	authClient *auth.Client

	http.Handler
}

type HandlerDeps struct {
	DB         *database.DB
	AuthClient *auth.Client
}

func NewHandler(deps HandlerDeps) http.Handler {
	mux := chi.NewMux()

	h := accountsHandler{
		db:         deps.DB,
		authClient: deps.AuthClient,
	}

	mux.Post("/register", h.register)
	mux.Post("/login", h.login)
	mux.Post("/refresh", h.refresh)
	mux.Post("/logout", h.logout)

	h.Handler = mux

	return h
}

const (
	tokenTypeBearer = "Bearer"

	unexpectedAccountCreationErrorMessage = "There was an unexpected error creating the account"
	unexpectedLoginError                  = "There was an unexpected error logging in"

	errTypeAccountCreation      = "account_creation_error"
	errTypeAccountAlreadyExists = "account_already_exists"
	errTypeValidationError      = "validation_error"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	Message   string `json:"message"`
	AccountID string `json:"account_id"`
}

func (h *accountsHandler) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqBody registerRequest

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		slog.ErrorContext(ctx, "error decoding register request body", "error", err.Error())
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "error reading request body",
			Type:       errTypeAccountCreation,
			StatusCode: http.StatusBadRequest,
		})

		return
	}

	if !auth.IsValidEmail(reqBody.Email) {
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "The provided email address is invalid",
			Type:       errTypeValidationError,
			StatusCode: http.StatusUnprocessableEntity,
		})
		return
	}

	hashedPassword, err := auth.HashPassword(reqBody.Password)
	if err != nil {
		var validationErr auth.ValidationError
		if errors.As(err, &validationErr) {
			httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
				Message:    err.Error(),
				Type:       errTypeValidationError,
				StatusCode: http.StatusUnprocessableEntity,
			})
			return
		}
		slog.ErrorContext(ctx, "error hashing password", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message: unexpectedAccountCreationErrorMessage,
			Type:    errTypeAccountCreation,
		})
		return
	}

	// unset the plaintext password
	reqBody.Password = ""

	// check if account already exists, if so return err
	// add account to db
	createdAccount, err := h.db.CreateAccount(ctx, database.AccountCreationParams{
		Email:        reqBody.Email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		if errors.Is(err, database.ErrAccountAlreadyExists) {
			httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
				Message:    "An account with this email already exists",
				Type:       errTypeAccountAlreadyExists,
				StatusCode: http.StatusConflict,
			})
			return
		}

		slog.ErrorContext(ctx, "error persisting new account to the db", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    unexpectedAccountCreationErrorMessage,
			Type:       errTypeAccountCreation,
			StatusCode: http.StatusInternalServerError,
		})

		return
	}
	// return user ID
	httputils.WriteJSONResponse(w, r, http.StatusCreated, registerResponse{
		Message:   "Account created successfully",
		AccountID: createdAccount.ID,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginOrRefreshResponse is used for both login and refresh responses
type loginOrRefreshResponse struct {
	Message      string `json:"message"`
	AccountID    string `json:"account_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (h *accountsHandler) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqBody loginRequest

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		slog.ErrorContext(ctx, "error decoding login request body", "error", err.Error())
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "error reading request body",
			StatusCode: http.StatusBadRequest,
		})

		return
	}

	// check email and password
	account, err := h.db.GetAccount(ctx, reqBody.Email)
	if err != nil {
		if errors.Is(err, database.ErrAccountNotFound) {
			httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
				Message:    "No account was found matching this email",
				Type:       "account_not_found",
				StatusCode: http.StatusUnauthorized,
			})

			return
		}
		slog.ErrorContext(ctx, "error getting account for login", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    unexpectedLoginError,
			StatusCode: http.StatusInternalServerError,
		})

		return
	}

	if !auth.PasswordIsCorrect(reqBody.Password, account.PasswordHash) {
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "Password is incorrect",
			StatusCode: http.StatusUnauthorized,
		})

		return
	}

	// unset the plaintext password
	reqBody.Password = ""

	// create a refresh token and persist in the db
	refreshToken, refreshTokenExpiresAt := h.authClient.NewRefreshToken()

	err = h.db.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token:     refreshToken,
		AccountID: account.ID,
		ExpiresAt: refreshTokenExpiresAt,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error creating refresh token", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    unexpectedLoginError,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	accessToken, accessTokenExpiresAt, err := h.authClient.NewAccessToken(auth.AccountManagementClaims{
		AccountID: account.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error creating new access token", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "Error creating new access token",
			StatusCode: http.StatusInternalServerError,
		})

		return
	}

	now := time.Now()
	accessTokenExpiresIn := accessTokenExpiresAt.Sub(now).Seconds()

	httputils.WriteJSONResponse(w, r, http.StatusOK, loginOrRefreshResponse{
		Message:      "Login successful",
		AccountID:    account.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTokenExpiresIn),
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *accountsHandler) refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqBody refreshRequest

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		slog.ErrorContext(ctx, "error decoding login request body", "error", err.Error())
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "error reading request body",
			StatusCode: http.StatusBadRequest,
		})

		return
	}
	// if validation fails, return a 401
	token, err := h.db.GetRefreshToken(ctx, reqBody.RefreshToken)
	if err != nil {
		if errors.Is(err, database.ErrRefreshTokenNotFound) {
			httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
				Message:    "Your session has expired",
				Type:       "refresh_token_expired",
				StatusCode: http.StatusUnauthorized,
			})
			return
		}

		slog.ErrorContext(ctx, "error getting refresh token from db")
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "Error validating session",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// if the refresh token is expired, return a 401
	if token.ExpiresAt.Before(time.Now()) {
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "Your session has expired",
			Type:       "refresh_token_expired",
			StatusCode: http.StatusUnauthorized,
		})
		return
	}

	// TODO: DRY up the below code. Should probably put in its own method or package.
	// Likely would be better to get the account from the db after validating the token?

	// if not, refresh the refresh token and return it with a new access token
	// create a refresh token and persist in the db
	refreshToken, refreshTokenExpiresAt := h.authClient.NewRefreshToken()

	err = h.db.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token:     refreshToken,
		AccountID: token.AccountID,
		ExpiresAt: refreshTokenExpiresAt,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error creating refresh token", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    unexpectedLoginError,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	accessToken, accessTokenExpiresAt, err := h.authClient.NewAccessToken(auth.AccountManagementClaims{
		AccountID: token.AccountID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error creating new access token", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "Error creating new access token",
			StatusCode: http.StatusInternalServerError,
		})

		return
	}

	now := time.Now()
	accessTokenExpiresIn := accessTokenExpiresAt.Sub(now).Seconds()

	httputils.WriteJSONResponse(w, r, http.StatusOK, loginOrRefreshResponse{
		Message:      "Login successful",
		AccountID:    token.AccountID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTokenExpiresIn),
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *accountsHandler) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqBody logoutRequest

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		slog.ErrorContext(ctx, "error decoding logout request body", "error", err.Error())
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "error reading request body",
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	// Get the refresh token to find the account ID
	token, err := h.db.GetRefreshToken(ctx, reqBody.RefreshToken)
	if err != nil {
		if errors.Is(err, database.ErrRefreshTokenNotFound) {
			// Token doesn't exist, but that's okay for logout
			httputils.WriteJSONResponse(w, r, http.StatusOK, map[string]string{
				"message": "Logged out successfully",
			})
			return
		}
		slog.ErrorContext(ctx, "error getting refresh token for logout", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "There was an unexpected error logging out",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// Delete the refresh token to revoke the session
	err = h.db.DeleteRefreshToken(ctx, token.AccountID)
	if err != nil {
		slog.ErrorContext(ctx, "error deleting refresh token", "error", err)
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "There was an unexpected error logging out",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	httputils.WriteJSONResponse(w, r, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

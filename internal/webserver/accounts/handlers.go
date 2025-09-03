package accounts

import (
	"context"
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

// Repository defines the DB methods needed by account handlers
type Repository interface {
	CreateAccount(ctx context.Context, params database.AccountCreationParams) (*database.Account, error)
	GetAccount(ctx context.Context, email string) (*database.Account, error)
	CreateRefreshToken(ctx context.Context, params database.CreateRefreshTokenParams) error
	GetRefreshToken(ctx context.Context, token string) (*database.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, accountID string) error
}

type handler struct {
	db         Repository
	authClient *auth.Client

	http.Handler
}

type HandlerDeps struct {
	DB         *database.DB
	AuthClient *auth.Client
}

func NewHandler(deps HandlerDeps) http.Handler {
	mux := chi.NewMux()

	h := handler{
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

	errTypeAccountAlreadyExists = "account_already_exists"
	errTypeAccountNotFound      = "account_not_found"
	errTypeIncorrectPassword    = "incorrect_password"
	errTypeInvalidRefreshToken  = "invalid_refresh_token"
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

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqBody registerRequest

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		slog.ErrorContext(ctx, "error decoding register request body", "error", err.Error())
		httputils.WriteErrorResponse(w, r, httputils.ErrorResponse{
			Message:    "error reading request body",
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
			Message:    unexpectedAccountCreationErrorMessage,
			StatusCode: http.StatusInternalServerError,
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

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
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
				Type:       errTypeAccountNotFound,
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
			Type:       errTypeIncorrectPassword,
			StatusCode: http.StatusUnauthorized,
		})
		return
	}

	// unset the plaintext password
	reqBody.Password = ""

	// Generate and persist tokens
	response, errResponse := h.generateAndPersistTokens(ctx, account.ID)
	if errResponse != nil {
		httputils.WriteErrorResponse(w, r, *errResponse)
		return
	}

	httputils.WriteJSONResponse(w, r, http.StatusOK, *response)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *handler) refresh(w http.ResponseWriter, r *http.Request) {
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
				Type:       errTypeInvalidRefreshToken,
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
			Type:       errTypeInvalidRefreshToken,
			StatusCode: http.StatusUnauthorized,
		})
		return
	}

	// Generate and persist new tokens
	response, errResponse := h.generateAndPersistTokens(ctx, token.AccountID)
	if errResponse != nil {
		httputils.WriteErrorResponse(w, r, *errResponse)
		return
	}

	httputils.WriteJSONResponse(w, r, http.StatusOK, *response)
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
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
	// (prevents using the refresh token to get a new access token without another login)
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

// generateAndPersistTokens creates new access and refresh tokens for the given account
func (h *handler) generateAndPersistTokens(ctx context.Context, accountID string) (*loginOrRefreshResponse, *httputils.ErrorResponse) {
	// Create a refresh token and persist in the db
	refreshToken, refreshTokenExpiresAt := h.authClient.NewRefreshToken()

	err := h.db.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token:     refreshToken,
		AccountID: accountID,
		ExpiresAt: refreshTokenExpiresAt,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error creating refresh token", "error", err)
		return nil, &httputils.ErrorResponse{
			Message:    "Error creating new token",
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Create access token
	accessToken, accessTokenExpiresAt, err := h.authClient.NewAccessToken(auth.Claims{
		AccountID: accountID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "error creating new access token", "error", err)
		return nil, &httputils.ErrorResponse{
			Message:    "Error creating new token",
			StatusCode: http.StatusInternalServerError,
		}
	}

	now := time.Now()
	accessTokenExpiresIn := accessTokenExpiresAt.Sub(now).Seconds()

	return &loginOrRefreshResponse{
		Message:      "Success",
		AccountID:    accountID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTokenExpiresIn),
	}, nil
}

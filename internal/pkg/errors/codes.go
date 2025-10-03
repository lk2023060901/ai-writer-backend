package errors

import (
	"fmt"
	"net/http"
)

// Code represents an error code with HTTP status and message
type Code struct {
	Code    int    // Business error code
	Status  int    // HTTP status code
	Message string // Error message
}

// Error codes for different modules
const (
	// Success
	Success = 0

	// Common errors (1000-1999)
	ErrInternalServer   = 1000
	ErrInvalidParams    = 1001
	ErrNotFound         = 1002
	ErrUnauthorized     = 1003
	ErrForbidden        = 1004
	ErrConflict         = 1005
	ErrTooManyRequests  = 1006
	ErrBadRequest       = 1007
	ErrServiceUnavail   = 1008

	// Auth errors (2000-2999)
	ErrAuthInvalidCredentials = 2000
	ErrAuthUserNotFound       = 2001
	ErrAuthEmailExists        = 2002
	ErrAuthAccountLocked      = 2003
	ErrAuthInvalid2FA         = 2004
	ErrAuthEmailNotVerified   = 2005
	ErrAuthInvalidToken       = 2006
	ErrAuthTokenExpired       = 2007
	ErrAuthWeakPassword       = 2008
	ErrAuthInvalidEmail       = 2009

	// User errors (3000-3999)
	ErrUserNotFound     = 3000
	ErrUserExists       = 3001
	ErrUserInvalidInput = 3002

	// Knowledge Base errors (4000-4999)
	ErrKBNotFound          = 4000
	ErrKBInvalidParams     = 4001
	ErrKBUnauthorized      = 4002
	ErrKBDocumentNotFound  = 4003
	ErrKBProcessingFailed  = 4004
	ErrKBStorageFailed     = 4005
	ErrKBVectorDBFailed    = 4006
	ErrKBEmbeddingFailed   = 4007
	ErrKBInvalidFileType   = 4008
	ErrKBFileTooLarge      = 4009
	ErrKBQuotaExceeded     = 4010

	// Agent errors (5000-5999)
	ErrAgentNotFound     = 5000
	ErrAgentInvalidInput = 5001
	ErrAgentUnauthorized = 5002
)

// codeMap maps error codes to their details
var codeMap = map[int]Code{
	Success: {Success, http.StatusOK, "Success"},

	// Common errors
	ErrInternalServer:  {ErrInternalServer, http.StatusInternalServerError, "Internal server error"},
	ErrInvalidParams:   {ErrInvalidParams, http.StatusBadRequest, "Invalid parameters"},
	ErrNotFound:        {ErrNotFound, http.StatusNotFound, "Resource not found"},
	ErrUnauthorized:    {ErrUnauthorized, http.StatusUnauthorized, "Unauthorized"},
	ErrForbidden:       {ErrForbidden, http.StatusForbidden, "Forbidden"},
	ErrConflict:        {ErrConflict, http.StatusConflict, "Resource conflict"},
	ErrTooManyRequests: {ErrTooManyRequests, http.StatusTooManyRequests, "Too many requests"},
	ErrBadRequest:      {ErrBadRequest, http.StatusBadRequest, "Bad request"},
	ErrServiceUnavail:  {ErrServiceUnavail, http.StatusServiceUnavailable, "Service unavailable"},

	// Auth errors
	ErrAuthInvalidCredentials: {ErrAuthInvalidCredentials, http.StatusUnauthorized, "Invalid email or password"},
	ErrAuthUserNotFound:       {ErrAuthUserNotFound, http.StatusNotFound, "User not found"},
	ErrAuthEmailExists:        {ErrAuthEmailExists, http.StatusConflict, "Email already exists"},
	ErrAuthAccountLocked:      {ErrAuthAccountLocked, http.StatusForbidden, "Account locked due to too many failed attempts"},
	ErrAuthInvalid2FA:         {ErrAuthInvalid2FA, http.StatusUnauthorized, "Invalid 2FA code"},
	ErrAuthEmailNotVerified:   {ErrAuthEmailNotVerified, http.StatusForbidden, "Email not verified"},
	ErrAuthInvalidToken:       {ErrAuthInvalidToken, http.StatusUnauthorized, "Invalid or expired token"},
	ErrAuthTokenExpired:       {ErrAuthTokenExpired, http.StatusUnauthorized, "Token expired"},
	ErrAuthWeakPassword:       {ErrAuthWeakPassword, http.StatusBadRequest, "Password is too weak"},
	ErrAuthInvalidEmail:       {ErrAuthInvalidEmail, http.StatusBadRequest, "Invalid email format"},

	// User errors
	ErrUserNotFound:     {ErrUserNotFound, http.StatusNotFound, "User not found"},
	ErrUserExists:       {ErrUserExists, http.StatusConflict, "User already exists"},
	ErrUserInvalidInput: {ErrUserInvalidInput, http.StatusBadRequest, "Invalid user input"},

	// Knowledge Base errors
	ErrKBNotFound:         {ErrKBNotFound, http.StatusNotFound, "Knowledge base not found"},
	ErrKBInvalidParams:    {ErrKBInvalidParams, http.StatusBadRequest, "Invalid knowledge base parameters"},
	ErrKBUnauthorized:     {ErrKBUnauthorized, http.StatusForbidden, "Unauthorized access to knowledge base"},
	ErrKBDocumentNotFound: {ErrKBDocumentNotFound, http.StatusNotFound, "Document not found"},
	ErrKBProcessingFailed: {ErrKBProcessingFailed, http.StatusInternalServerError, "Document processing failed"},
	ErrKBStorageFailed:    {ErrKBStorageFailed, http.StatusInternalServerError, "Storage operation failed"},
	ErrKBVectorDBFailed:   {ErrKBVectorDBFailed, http.StatusInternalServerError, "Vector database operation failed"},
	ErrKBEmbeddingFailed:  {ErrKBEmbeddingFailed, http.StatusInternalServerError, "Embedding generation failed"},
	ErrKBInvalidFileType:  {ErrKBInvalidFileType, http.StatusBadRequest, "Unsupported file type"},
	ErrKBFileTooLarge:     {ErrKBFileTooLarge, http.StatusBadRequest, "File size exceeds limit"},
	ErrKBQuotaExceeded:    {ErrKBQuotaExceeded, http.StatusForbidden, "Storage quota exceeded"},

	// Agent errors
	ErrAgentNotFound:     {ErrAgentNotFound, http.StatusNotFound, "Agent not found"},
	ErrAgentInvalidInput: {ErrAgentInvalidInput, http.StatusBadRequest, "Invalid agent input"},
	ErrAgentUnauthorized: {ErrAgentUnauthorized, http.StatusForbidden, "Unauthorized access to agent"},
}

// GetCode returns the Code for a given error code
func GetCode(code int) Code {
	if c, ok := codeMap[code]; ok {
		return c
	}
	return codeMap[ErrInternalServer]
}

// GetHTTPStatus returns HTTP status for a given error code
func GetHTTPStatus(code int) int {
	return GetCode(code).Status
}

// GetMessage returns the message for a given error code
func GetMessage(code int) string {
	return GetCode(code).Message
}

// IsSuccess checks if the code represents success
func IsSuccess(code int) bool {
	return code == Success
}

// IsClientError checks if the code represents a client error (4xx)
func IsClientError(code int) bool {
	status := GetHTTPStatus(code)
	return status >= 400 && status < 500
}

// IsServerError checks if the code represents a server error (5xx)
func IsServerError(code int) bool {
	status := GetHTTPStatus(code)
	return status >= 500
}

// FormatError formats an error message with code
func FormatError(code int, details ...string) string {
	msg := GetMessage(code)
	if len(details) > 0 && details[0] != "" {
		return fmt.Sprintf("%s: %s", msg, details[0])
	}
	return msg
}

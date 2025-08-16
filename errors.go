package macaudio

import "fmt"

// ErrorHandler defines the interface for handling engine errors
type ErrorHandler interface {
	HandleError(error)
}

// DefaultErrorHandler provides a basic error handling implementation
type DefaultErrorHandler struct{}

// HandleError implements ErrorHandler interface with basic logging
func (h *DefaultErrorHandler) HandleError(err error) {
	// TODO: Replace with proper logging framework
	fmt.Printf("Engine Error: %v\n", err)
}

// LoggingErrorHandler wraps another handler and logs errors
type LoggingErrorHandler struct {
	underlying ErrorHandler
	logger     func(error)
}

// NewLoggingErrorHandler creates a new logging error handler
func NewLoggingErrorHandler(underlying ErrorHandler, logger func(error)) *LoggingErrorHandler {
	return &LoggingErrorHandler{
		underlying: underlying,
		logger:     logger,
	}
}

// HandleError implements ErrorHandler interface with logging
func (h *LoggingErrorHandler) HandleError(err error) {
	if h.logger != nil {
		h.logger(err)
	}
	if h.underlying != nil {
		h.underlying.HandleError(err)
	}
}

// PanicErrorHandler panics on any error (useful for development)
type PanicErrorHandler struct{}

// HandleError implements ErrorHandler interface by panicking
func (h *PanicErrorHandler) HandleError(err error) {
	panic(fmt.Sprintf("Engine error: %v", err))
}

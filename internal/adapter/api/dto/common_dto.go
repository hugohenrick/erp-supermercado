package dto

// ErrorResponse representa a estrutura de resposta para erros
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse representa uma resposta genérica de sucesso
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse cria uma nova resposta de sucesso
func NewSuccessResponse(message string, data interface{}) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse cria uma nova resposta de erro
func NewErrorResponse(code int, message, details string) ErrorResponse {
	return ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// PaginationParams representa os parâmetros de paginação
type PaginationParams struct {
	Page     int
	PageSize int
}

// GetPagination retorna parâmetros de paginação com valores padrão
func GetPagination(page, pageSize int) PaginationParams {
	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 10
	} else if pageSize > 100 {
		pageSize = 100 // Limitar a 100 itens por página
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

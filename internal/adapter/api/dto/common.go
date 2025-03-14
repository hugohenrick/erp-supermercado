package dto

// ErrorResponse representa a estrutura de resposta para erros
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse representa a estrutura de resposta para operações bem-sucedidas
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Pagination representa a estrutura de paginação
type Pagination struct {
	Page     int
	PageSize int
}

// GetPagination retorna uma estrutura de paginação com valores padrão
func GetPagination(page, pageSize int) Pagination {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 10
	} else if pageSize > 100 {
		pageSize = 100
	}

	return Pagination{
		Page:     page,
		PageSize: pageSize,
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

// NewSuccessResponse cria uma nova resposta de sucesso
func NewSuccessResponse(message string, data interface{}) SuccessResponse {
	return SuccessResponse{
		Message: message,
		Data:    data,
	}
}

// calculateTotalPages calcula o número total de páginas com base no total de registros e no tamanho da página
func calculateTotalPages(totalCount, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return totalPages
}

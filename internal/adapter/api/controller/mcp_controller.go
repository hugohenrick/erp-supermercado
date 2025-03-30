package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/pkg/mcp"
)

// MCPController handles MCP-related requests
type MCPController struct {
	mcpClient *mcp.MCPClient
}

// NewMCPController creates a new MCP controller
func NewMCPController(mcpClient *mcp.MCPClient) *MCPController {
	return &MCPController{
		mcpClient: mcpClient,
	}
}

type MCPMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

// ProcessMessage godoc
// @Summary Process a message through MCP
// @Description Process a user message and return the response with chat history
// @Tags MCP
// @Accept json
// @Produce json
// @Param message body MCPMessageRequest true "Message to process"
// @Success 200 {object} chat.Message
// @Router /api/v1/mcp/message [post]
func (c *MCPController) ProcessMessage(ctx *gin.Context) {
	var req MCPMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mcpContextData := mcp.GetContextFromRequest(ctx)
	if mcpContextData == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "MCP context not found"})
		return
	}

	response, err := c.mcpClient.ProcessWithContext(ctx, req.Message, mcpContextData)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	history, err := c.mcpClient.GetChatHistory(ctx, mcpContextData.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"response": response,
		"history":  history,
	})
}

// GetHistory godoc
// @Summary Get chat history
// @Description Get the chat history for the current user
// @Tags MCP
// @Produce json
// @Success 200 {array} chat.Message
// @Router /api/v1/mcp/history [get]
func (c *MCPController) GetHistory(ctx *gin.Context) {
	mcpContextData := mcp.GetContextFromRequest(ctx)
	if mcpContextData == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "MCP context not found"})
		return
	}

	history, err := c.mcpClient.GetChatHistory(ctx, mcpContextData.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}

// DeleteHistory godoc
// @Summary Delete chat history
// @Description Delete the chat history for the current user
// @Tags MCP
// @Success 200 {object} string
// @Router /api/v1/mcp/history [delete]
func (c *MCPController) DeleteHistory(ctx *gin.Context) {
	mcpContextData := mcp.GetContextFromRequest(ctx)
	if mcpContextData == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "MCP context not found"})
		return
	}

	err := c.mcpClient.DeleteChatHistory(ctx, mcpContextData.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Chat history deleted successfully",
	})
}

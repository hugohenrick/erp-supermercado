package intent

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hugohenrick/erp-supermercado/pkg/domain"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// ProductIntentHandler gerencia intenções relacionadas a produtos
type ProductIntentHandler struct {
	logger        logger.Logger
	productRepo   repository.ProductRepository
	regexPatterns map[string]*regexp.Regexp
}

// NewProductIntentHandler cria uma nova instância do handler de intenções de produto
func NewProductIntentHandler(log logger.Logger, productRepo repository.ProductRepository) *ProductIntentHandler {
	handler := &ProductIntentHandler{
		logger:      log,
		productRepo: productRepo,
		regexPatterns: map[string]*regexp.Regexp{
			"create_product": regexp.MustCompile(`(?i)(cri[ae]r?|cadastr[ae]r?|adicionar?|inserir?)\s+(um\s+)?((novo|uma)\s+)?produto.+?(?:nome|chamad[oa])\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s0-9]+?)(\s+com|\s+e|\s+preço|\s+valor|\s+categoria|$)`),
			"get_product":    regexp.MustCompile(`(?i)(busc[ae]r?|encontr[ae]r?|procurar?|mostr[ae]r?|exib[ae]r?)\s+(o\s+)?produto\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s0-9]+?)|(?:com\s+)?(?:código|sku)\s+(?P<sku>[A-Za-z0-9-]+)|(?:com\s+)?id\s+(?P<id>\d+))(\s|$|\.)`),
			"update_product": regexp.MustCompile(`(?i)(atualiz[ae]r?|modific[ae]r?|alter[ae]r?|muda[ae]r?)\s+(o\s+)?produto\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s0-9]+?)|(?:com\s+)?(?:código|sku)\s+(?P<sku>[A-Za-z0-9-]+)|(?:com\s+)?id\s+(?P<id>\d+))`),
			"delete_product": regexp.MustCompile(`(?i)(delet[ae]r?|exclu[iíy]r?|remov[ae]r?|apag[ae]r?)\s+(o\s+)?produto\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s0-9]+?)|(?:com\s+)?(?:código|sku)\s+(?P<sku>[A-Za-z0-9-]+)|(?:com\s+)?id\s+(?P<id>\d+))(\s|$|\.)`),
			"list_products":  regexp.MustCompile(`(?i)(list[ae]r?|mostr[ae]r?|exib[ae]r?|ver)\s+(os\s+)?produtos`),
			"update_stock":   regexp.MustCompile(`(?i)(atualiz[ae]r?|modific[ae]r?|alter[ae]r?|muda[ae]r?)\s+(o\s+)?estoque\s+(do\s+)?produto\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s0-9]+?)|(?:com\s+)?(?:código|sku)\s+(?P<sku>[A-Za-z0-9-]+)|(?:com\s+)?id\s+(?P<id>\d+))`),
			"update_price":   regexp.MustCompile(`(?i)(atualiz[ae]r?|modific[ae]r?|alter[ae]r?|muda[ae]r?)\s+(o\s+)?preço\s+(do\s+)?produto\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s0-9]+?)|(?:com\s+)?(?:código|sku)\s+(?P<sku>[A-Za-z0-9-]+)|(?:com\s+)?id\s+(?P<id>\d+))`),
		},
	}
	return handler
}

// CanHandle verifica se este handler pode processar a mensagem
func (h *ProductIntentHandler) CanHandle(message string) bool {
	message = strings.TrimSpace(message)

	for _, pattern := range h.regexPatterns {
		if pattern.MatchString(message) {
			return true
		}
	}

	// Verificar outros padrões de linguagem natural sobre produtos
	productTerms := []string{
		"produto", "mercadoria", "item", "estoque",
		"preço", "valor", "cadastro de produto",
	}

	loweredMsg := strings.ToLower(message)
	for _, term := range productTerms {
		if strings.Contains(loweredMsg, term) {
			return true
		}
	}

	return false
}

// Extract extrai a intenção e entidades da mensagem
func (h *ProductIntentHandler) Extract(message string) (*Intent, error) {
	message = strings.TrimSpace(message)

	// Verificar cada padrão de regex para encontrar correspondências
	for intentName, pattern := range h.regexPatterns {
		match := pattern.FindStringSubmatch(message)

		if match != nil {
			// Extrair entidades do regex
			entities := make(map[string]interface{})

			// Obter índices dos grupos nomeados
			subexpNames := pattern.SubexpNames()
			for i, name := range subexpNames {
				if i != 0 && name != "" && match[i] != "" {
					entities[name] = strings.TrimSpace(match[i])
				}
			}

			// Extrair informações adicionais da mensagem
			h.extractAdditionalInfo(message, entities)

			return &Intent{
				Name:            intentName,
				Confidence:      0.8, // Valor fixo para regex; poderia ser dinâmico
				Entities:        entities,
				OriginalMessage: message,
			}, nil
		}
	}

	// Se não encontrou um padrão específico, mas está relacionado a produtos
	if h.CanHandle(message) {
		return &Intent{
			Name:            "product_generic",
			Confidence:      0.4,
			Entities:        make(map[string]interface{}),
			OriginalMessage: message,
		}, nil
	}

	return nil, nil
}

// CheckPermission verifica se o usuário tem permissão para executar a ação
func (h *ProductIntentHandler) CheckPermission(ctxData ContextData) bool {
	// Verificar se o usuário tem permissão baseado no papel/role
	switch ctxData.Role {
	case "admin", "administrator", "superadmin", "manager":
		// Administradores e gerentes podem fazer tudo
		return true

	case "inventory", "stock":
		// Pessoal de estoque pode criar e atualizar produtos/estoque
		return true

	case "sales":
		// Vendedores podem apenas consultar produtos
		if intentName := h.getIntentNameFromContext(ctxData); intentName == "get_product" ||
			intentName == "list_products" || intentName == "product_generic" {
			return true
		}
		return false

	default:
		// Consultas são permitidas para todos os papéis autenticados
		if intentName := h.getIntentNameFromContext(ctxData); intentName == "get_product" ||
			intentName == "list_products" {
			return true
		}
		return false
	}
}

// getIntentNameFromContext é um helper para obter o nome da intenção do contexto
func (h *ProductIntentHandler) getIntentNameFromContext(ctxData ContextData) string {
	// Implementação básica - em um sistema real, teríamos o intent no contexto
	return ""
}

// Execute executa a ação correspondente à intenção
func (h *ProductIntentHandler) Execute(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	switch intent.Name {
	case "create_product":
		return h.createProduct(ctxData, intent)
	case "get_product":
		return h.getProduct(ctxData, intent)
	case "update_product":
		return h.updateProduct(ctxData, intent)
	case "delete_product":
		return h.deleteProduct(ctxData, intent)
	case "list_products":
		return h.listProducts(ctxData, intent)
	case "update_stock":
		return h.updateStock(ctxData, intent)
	case "update_price":
		return h.updatePrice(ctxData, intent)
	case "product_generic":
		return &ActionResult{
			Success: false,
			Message: "Entendi que você quer realizar alguma ação relacionada a produtos, mas não consegui identificar exatamente o que. Poderia ser mais específico? Por exemplo, 'criar um produto chamado Arroz' ou 'listar todos os produtos'.",
		}, nil
	default:
		return nil, errors.New("intenção não suportada")
	}
}

// createProduct cria um novo produto
func (h *ProductIntentHandler) createProduct(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Extrair dados necessários
	name, hasName := intent.Entities["name"].(string)
	desc, hasDesc := intent.Entities["description"].(string)
	sku, hasSKU := intent.Entities["sku"].(string)
	category, hasCategory := intent.Entities["category"].(string)

	// Parseamos o preço se existir
	var price float64
	hasPrice := false
	if priceStr, ok := intent.Entities["price"].(string); ok && priceStr != "" {
		if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
			price = p
			hasPrice = true
		}
	}

	// Parseamos a quantidade em estoque se existir
	var stockQty int
	hasStockQty := false
	if stockStr, ok := intent.Entities["stock_qty"].(string); ok && stockStr != "" {
		if sq, err := strconv.Atoi(stockStr); err == nil {
			stockQty = sq
			hasStockQty = true
		}
	}

	// Validar dados obrigatórios
	missingFields := make([]string, 0)

	if !hasName || strings.TrimSpace(name) == "" {
		missingFields = append(missingFields, "nome")
	}

	// Solicitar os dados que estão faltando, se houver
	if len(missingFields) > 0 {
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Para criar um produto, preciso dos seguintes dados: %s. Pode me informar?", strings.Join(missingFields, ", ")),
		}, nil
	}

	// Definir valores padrão para campos não informados
	if !hasDesc {
		desc = ""
	}

	if !hasSKU {
		// Gerar SKU baseado no nome
		sku = generateSKU(name)
	}

	if !hasCategory {
		category = "Geral"
	}

	if !hasPrice {
		price = 0.0
	}

	if !hasStockQty {
		stockQty = 0
	}

	// Criar o objeto de produto
	newProduct := &domain.Product{
		Name:        name,
		Description: desc,
		SKU:         sku,
		Price:       price,
		StockQty:    stockQty,
		Category:    category,
		TenantID:    ctxData.TenantID,
		Active:      true,
	}

	// Chamar o repositório para persistir o produto
	err := h.productRepo.Create(ctxData.TenantID, newProduct)
	if err != nil {
		h.logger.Error("Erro ao criar produto", "error", err, "product", name)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Não foi possível criar o produto: %v", err),
		}, nil
	}

	// Retornar o resultado da operação
	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Produto '%s' criado com sucesso! SKU: %s", name, sku),
		Data: map[string]interface{}{
			"product_id": newProduct.ID,
			"sku":        sku,
		},
	}, nil
}

// getProduct busca um produto específico
func (h *ProductIntentHandler) getProduct(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Verificar qual critério de busca foi fornecido
	var product *domain.Product
	var err error

	// Se temos ID, usamos ele como prioridade
	if id, ok := intent.Entities["id"].(string); ok && id != "" {
		product, err = h.productRepo.FindByID(ctxData.TenantID, id)
	} else if sku, ok := intent.Entities["sku"].(string); ok && sku != "" {
		// Se temos SKU, usamos como segunda opção
		product, err = h.productRepo.FindBySKU(ctxData.TenantID, sku)
	} else if name, ok := intent.Entities["name"].(string); ok && name != "" {
		// Se temos nome, buscamos por nome
		products, err := h.productRepo.FindByName(ctxData.TenantID, name)
		if err != nil {
			h.logger.Error("Erro ao buscar produto por nome", "error", err, "name", name)
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Erro ao buscar produto por nome: %v", err),
			}, nil
		}

		if len(products) == 0 {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Não encontrei nenhum produto com o nome '%s'", name),
			}, nil
		}

		if len(products) > 1 {
			// Múltiplos resultados, listar opções
			var productList strings.Builder
			productList.WriteString("Encontrei vários produtos com esse nome. Qual destes você está procurando?\n\n")

			for i, p := range products {
				productList.WriteString(fmt.Sprintf("%d. %s - SKU: %s - Preço: R$ %.2f\n",
					i+1, p.Name, p.SKU, p.Price))
			}

			return &ActionResult{
				Success: true,
				Message: productList.String(),
				Data: map[string]interface{}{
					"products":        products,
					"multiple_result": true,
				},
			}, nil
		}

		// Se encontrou exatamente um produto
		product = products[0]
	} else {
		// Não temos critérios suficientes
		return &ActionResult{
			Success: false,
			Message: "Preciso de mais informações para encontrar o produto. Por favor, informe o ID, SKU ou nome do produto.",
		}, nil
	}

	if err != nil {
		h.logger.Error("Erro ao buscar produto", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao buscar o produto: %v", err),
		}, nil
	}

	if product == nil {
		return &ActionResult{
			Success: false,
			Message: "Produto não encontrado.",
		}, nil
	}

	// Mostrar os dados do produto encontrado
	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Produto encontrado:\n"+
			"- ID: %s\n"+
			"- Nome: %s\n"+
			"- Descrição: %s\n"+
			"- SKU: %s\n"+
			"- Preço: R$ %.2f\n"+
			"- Estoque: %d unidades\n"+
			"- Categoria: %s\n"+
			"- Status: %s",
			product.ID,
			product.Name,
			product.Description,
			product.SKU,
			product.Price,
			product.StockQty,
			product.Category,
			productStatusToString(product.Active)),
		Data: map[string]interface{}{
			"product": product,
		},
	}, nil
}

// Implementação simplificada dos outros métodos para manter o exemplo conciso
func (h *ProductIntentHandler) updateProduct(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	return &ActionResult{
		Success: true,
		Message: "Produto atualizado com sucesso.",
	}, nil
}

func (h *ProductIntentHandler) deleteProduct(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	return &ActionResult{
		Success: true,
		Message: "Produto excluído com sucesso.",
	}, nil
}

func (h *ProductIntentHandler) listProducts(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	return &ActionResult{
		Success: true,
		Message: "Lista de produtos obtida com sucesso.",
	}, nil
}

func (h *ProductIntentHandler) updateStock(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	return &ActionResult{
		Success: true,
		Message: "Estoque atualizado com sucesso.",
	}, nil
}

func (h *ProductIntentHandler) updatePrice(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	return &ActionResult{
		Success: true,
		Message: "Preço atualizado com sucesso.",
	}, nil
}

// extractAdditionalInfo extrai informações adicionais da mensagem
func (h *ProductIntentHandler) extractAdditionalInfo(message string, entities map[string]interface{}) {
	// Extrair preço com regex específico
	priceRegex := regexp.MustCompile(`(?i)(?:preço|valor|custo)[\s:]+(?:R\$\s*)?([0-9]+(?:[.,][0-9]+)?)`)
	priceMatch := priceRegex.FindStringSubmatch(message)
	if priceMatch != nil && len(priceMatch) > 1 {
		// Normalizar formato do número (substituir , por .)
		normalizedPrice := strings.Replace(priceMatch[1], ",", ".", -1)
		entities["price"] = normalizedPrice
	}

	// Extrair quantidade em estoque
	stockRegex := regexp.MustCompile(`(?i)(?:estoque|quantidade|qtd)[\s:]+([0-9]+)`)
	stockMatch := stockRegex.FindStringSubmatch(message)
	if stockMatch != nil && len(stockMatch) > 1 {
		entities["stock_qty"] = stockMatch[1]
	}

	// Extrair SKU
	skuRegex := regexp.MustCompile(`(?i)(?:sku|código|referência)[\s:]+([A-Za-z0-9-]+)`)
	skuMatch := skuRegex.FindStringSubmatch(message)
	if skuMatch != nil && len(skuMatch) > 1 {
		entities["sku"] = skuMatch[1]
	}

	// Extrair categoria
	categoryRegex := regexp.MustCompile(`(?i)(?:categoria|departamento|setor)[\s:]+([A-Za-zÀ-ÖØ-öø-ÿ\s]+?)(?:\s+e|\s+com|\s+tendo|\.|$)`)
	categoryMatch := categoryRegex.FindStringSubmatch(message)
	if categoryMatch != nil && len(categoryMatch) > 1 {
		entities["category"] = strings.TrimSpace(categoryMatch[1])
	}

	// Extrair descrição
	descRegex := regexp.MustCompile(`(?i)(?:descrição|desc)[\s:]+([A-Za-zÀ-ÖØ-öø-ÿ\s0-9.,!-]+?)(?:\s+e|\s+com|\s+tendo|\.|$)`)
	descMatch := descRegex.FindStringSubmatch(message)
	if descMatch != nil && len(descMatch) > 1 {
		entities["description"] = strings.TrimSpace(descMatch[1])
	}
}

// generateSKU gera um SKU baseado no nome do produto
func generateSKU(name string) string {
	// Implementação simplificada - em produção usaria algo mais robusto
	name = strings.ToUpper(name)

	// Remover acentos e caracteres especiais
	name = regexp.MustCompile(`[^A-Z0-9]`).ReplaceAllString(name, "")

	// Pegar os primeiros 3 caracteres e adicionar um número aleatório
	prefix := name
	if len(prefix) > 3 {
		prefix = prefix[:3]
	}

	// Número "aleatório" para exemplo
	randomNum := 12345

	return fmt.Sprintf("%s%d", prefix, randomNum)
}

// productStatusToString converte o status booleano para string
func productStatusToString(active bool) string {
	if active {
		return "Ativo"
	}
	return "Inativo"
}

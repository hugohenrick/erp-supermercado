package adapter

import (
	"context"
	"fmt"

	"github.com/hugohenrick/erp-supermercado/internal/domain/customer"
	"github.com/hugohenrick/erp-supermercado/pkg/domain"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// CustomerRepositoryAdapter adapts between the simplified repository interface and the actual implementation
type CustomerRepositoryAdapter struct {
	internalRepo customer.Repository
	logger       logger.Logger
}

// NewCustomerRepositoryAdapter creates a new adapter for the customer repository
func NewCustomerRepositoryAdapter(internalRepo customer.Repository, log logger.Logger) repository.CustomerRepository {
	return &CustomerRepositoryAdapter{
		internalRepo: internalRepo,
		logger:       log,
	}
}

// Create creates a new customer
func (a *CustomerRepositoryAdapter) Create(tenantID string, c *domain.Customer) error {
	a.logger.Info("CustomerRepositoryAdapter.Create called",
		"id", c.ID,
		"name", c.Name,
		"document", c.Document,
		"email", c.Email,
		"tenant_id", tenantID)

	// Convert our simple domain model to the internal domain model
	personType := customer.PersonTypePF
	if c.CustomerType == "PJ" {
		personType = customer.PersonTypePJ
	}

	// Create a basic internal customer
	a.logger.Debug("Creating internal customer with data",
		"tenant_id", tenantID,
		"name", c.Name,
		"document", c.Document,
		"person_type", personType)

	internalCustomer, err := customer.NewCustomer(
		tenantID,
		"", // BranchID - let the repo figure it out
		personType,
		c.Name,
		c.Document,
	)
	if err != nil {
		a.logger.Error("Failed to create internal customer",
			"error", err,
			"name", c.Name,
			"document", c.Document)
		return err
	}

	// Preserve the ID if provided
	if c.ID != "" {
		internalCustomer.ID = c.ID
		a.logger.Debug("Using provided ID for internal customer", "id", c.ID)
	}

	// Add email as a contact if provided
	if c.Email != "" {
		a.logger.Debug("Adding email contact", "email", c.Email)
		contact := customer.Contact{
			Name:        c.Name,
			Email:       c.Email,
			MainContact: true,
		}
		internalCustomer.AddContact(contact)
	}

	// Add phone as a contact if provided
	if c.Phone != "" {
		a.logger.Debug("Adding phone contact", "phone", c.Phone)
		// If we already have a contact with email, add phone to it
		if len(internalCustomer.Contacts) > 0 {
			internalCustomer.Contacts[0].Phone = c.Phone
		} else {
			// Otherwise create a new contact
			contact := customer.Contact{
				Name:        c.Name,
				Phone:       c.Phone,
				MainContact: true,
			}
			internalCustomer.AddContact(contact)
		}
	}

	// Add address if provided
	if c.Address != "" || c.City != "" || c.State != "" || c.ZipCode != "" {
		a.logger.Debug("Adding address",
			"address", c.Address,
			"city", c.City,
			"state", c.State,
			"zip", c.ZipCode)
		address := customer.Address{
			Street:          c.Address,
			City:            c.City,
			State:           c.State,
			ZipCode:         c.ZipCode,
			MainAddress:     true,
			DeliveryAddress: true,
		}
		internalCustomer.AddAddress(address)
	}

	// Set status based on Active flag
	if c.Active {
		internalCustomer.Status = customer.StatusActive
	} else {
		internalCustomer.Status = customer.StatusInactive
	}

	// Call the internal repository
	ctx := context.Background()
	a.logger.Info("Calling internal repository Create method",
		"tenant_id", tenantID,
		"customer_id", internalCustomer.ID,
		"contacts", len(internalCustomer.Contacts),
		"addresses", len(internalCustomer.Addresses))

	err = a.internalRepo.Create(ctx, internalCustomer)
	if err != nil {
		a.logger.Error("Internal repository Create failed",
			"error", err,
			"customer_id", internalCustomer.ID)
		return err
	}

	a.logger.Info("Customer successfully created",
		"id", internalCustomer.ID,
		"name", internalCustomer.Name)
	return nil
}

// Update updates an existing customer
func (a *CustomerRepositoryAdapter) Update(tenantID string, c *domain.Customer) error {
	// For simplicity, we'll fetch the customer first
	ctx := context.Background()
	internalCustomer, err := a.internalRepo.FindByID(ctx, c.ID)
	if err != nil {
		return err
	}

	// Update the fields that can be updated
	internalCustomer.Name = c.Name

	// Update document if provided
	if c.Document != "" {
		internalCustomer.Document = c.Document
		// Update person type based on document
		if len(c.Document) > 11 {
			internalCustomer.PersonType = customer.PersonTypePJ
		} else {
			internalCustomer.PersonType = customer.PersonTypePF
		}
	}

	// Update status
	if c.Active {
		internalCustomer.Status = customer.StatusActive
	} else {
		internalCustomer.Status = customer.StatusInactive
	}

	// Update contact information if exists
	if c.Email != "" || c.Phone != "" {
		if len(internalCustomer.Contacts) > 0 {
			// Update existing contact
			if c.Email != "" {
				internalCustomer.Contacts[0].Email = c.Email
			}
			if c.Phone != "" {
				internalCustomer.Contacts[0].Phone = c.Phone
			}
		} else {
			// Create new contact
			contact := customer.Contact{
				Name:        c.Name,
				Email:       c.Email,
				Phone:       c.Phone,
				MainContact: true,
			}
			internalCustomer.AddContact(contact)
		}
	}

	// Update address information if exists
	if c.Address != "" || c.City != "" || c.State != "" || c.ZipCode != "" {
		mainAddress := internalCustomer.GetMainAddress()
		if mainAddress != nil {
			// Update existing address
			if c.Address != "" {
				mainAddress.Street = c.Address
			}
			if c.City != "" {
				mainAddress.City = c.City
			}
			if c.State != "" {
				mainAddress.State = c.State
			}
			if c.ZipCode != "" {
				mainAddress.ZipCode = c.ZipCode
			}
		} else {
			// Create new address
			address := customer.Address{
				Street:          c.Address,
				City:            c.City,
				State:           c.State,
				ZipCode:         c.ZipCode,
				MainAddress:     true,
				DeliveryAddress: true,
			}
			internalCustomer.AddAddress(address)
		}
	}

	// Call the internal repository
	a.logger.Info("Calling internal repository Update method", "tenant_id", tenantID, "customer_id", internalCustomer.ID)
	return a.internalRepo.Update(ctx, internalCustomer)
}

// Delete deletes a customer by ID
func (a *CustomerRepositoryAdapter) Delete(tenantID string, customerID string) error {
	ctx := context.Background()
	a.logger.Info("Calling internal repository Delete method", "tenant_id", tenantID, "customer_id", customerID)
	return a.internalRepo.Delete(ctx, customerID)
}

// FindByID finds a customer by ID
func (a *CustomerRepositoryAdapter) FindByID(tenantID string, customerID string) (*domain.Customer, error) {
	ctx := context.Background()
	a.logger.Info("Calling internal repository FindByID method", "tenant_id", tenantID, "customer_id", customerID)
	internalCustomer, err := a.internalRepo.FindByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return a.convertToSimpleCustomer(internalCustomer), nil
}

// FindByDocument finds a customer by document (CPF/CNPJ)
func (a *CustomerRepositoryAdapter) FindByDocument(tenantID string, document string) (*domain.Customer, error) {
	ctx := context.Background()
	a.logger.Info("Calling internal repository FindByDocument method", "tenant_id", tenantID, "document", document)
	internalCustomer, err := a.internalRepo.FindByDocument(ctx, tenantID, document)
	if err != nil {
		return nil, err
	}

	return a.convertToSimpleCustomer(internalCustomer), nil
}

// FindByEmail finds a customer by email
func (a *CustomerRepositoryAdapter) FindByEmail(tenantID string, email string) (*domain.Customer, error) {
	ctx := context.Background()

	// This is not directly available in the internal repo, so we do a workaround
	// We'll use a simple text search on the JSON contacts field
	customers, err := a.internalRepo.List(ctx, tenantID, 10, 0)
	if err != nil {
		return nil, err
	}

	// Search through contacts for the email
	for _, c := range customers {
		for _, contact := range c.Contacts {
			if contact.Email == email {
				return a.convertToSimpleCustomer(c), nil
			}
		}
	}

	return nil, fmt.Errorf("customer with email %s not found", email)
}

// FindByName finds customers by name
func (a *CustomerRepositoryAdapter) FindByName(tenantID string, name string) ([]*domain.Customer, error) {
	ctx := context.Background()
	a.logger.Info("Calling internal repository FindByName method", "tenant_id", tenantID, "name", name)
	internalCustomers, err := a.internalRepo.FindByName(ctx, tenantID, name, 10, 0)
	if err != nil {
		return nil, err
	}

	simpleCustomers := make([]*domain.Customer, 0, len(internalCustomers))
	for _, c := range internalCustomers {
		simpleCustomers = append(simpleCustomers, a.convertToSimpleCustomer(c))
	}

	return simpleCustomers, nil
}

// FindAll finds all customers for a tenant
func (a *CustomerRepositoryAdapter) FindAll(tenantID string) ([]*domain.Customer, error) {
	ctx := context.Background()
	a.logger.Info("Calling internal repository List method", "tenant_id", tenantID)
	internalCustomers, err := a.internalRepo.List(ctx, tenantID, 100, 0)
	if err != nil {
		return nil, err
	}

	simpleCustomers := make([]*domain.Customer, 0, len(internalCustomers))
	for _, c := range internalCustomers {
		simpleCustomers = append(simpleCustomers, a.convertToSimpleCustomer(c))
	}

	return simpleCustomers, nil
}

// convertToSimpleCustomer converts from internal domain model to simplified domain model
func (a *CustomerRepositoryAdapter) convertToSimpleCustomer(c *customer.Customer) *domain.Customer {
	simpleCustomer := &domain.Customer{
		ID:           c.ID,
		Name:         c.Name,
		TenantID:     c.TenantID,
		Document:     c.Document,
		Active:       c.Status == customer.StatusActive,
		CustomerType: string(c.CustomerType),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}

	// Extract email from contacts
	if len(c.Contacts) > 0 {
		simpleCustomer.Email = c.Contacts[0].Email
		simpleCustomer.Phone = c.Contacts[0].Phone
	}

	// Extract address information
	mainAddress := c.GetMainAddress()
	if mainAddress != nil {
		simpleCustomer.Address = mainAddress.Street
		simpleCustomer.City = mainAddress.City
		simpleCustomer.State = mainAddress.State
		simpleCustomer.ZipCode = mainAddress.ZipCode
	}

	return simpleCustomer
}

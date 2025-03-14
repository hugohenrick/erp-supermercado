package tenant

// Address representa um endereço no sistema
type Address struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement,omitempty"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`
	Country    string `json:"country"`
}

// NewAddress cria uma nova instância de Address
func NewAddress(street, number, complement, district, city, state, zipCode, country string) Address {
	return Address{
		Street:     street,
		Number:     number,
		Complement: complement,
		District:   district,
		City:       city,
		State:      state,
		ZipCode:    zipCode,
		Country:    country,
	}
}

// IsEmpty verifica se o endereço está vazio
func (a *Address) IsEmpty() bool {
	return a.Street == "" && a.Number == "" && a.District == "" && a.City == "" && a.State == "" && a.ZipCode == ""
}

// Format formata o endereço como string
func (a *Address) Format() string {
	addr := a.Street + ", " + a.Number
	if a.Complement != "" {
		addr += " - " + a.Complement
	}
	addr += " - " + a.District + ", " + a.City + " - " + a.State + ", " + a.ZipCode
	if a.Country != "" {
		addr += ", " + a.Country
	}
	return addr
} 
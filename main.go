package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type CSFloatResponse struct {
	Data []Listing `json:"data"`
}

type Listing struct {
	ID        string `json:"id"`
	Price     int64  `json:"price"`
	Reference struct {
		BasePrice int64 `json:"base_price"`
	} `json:"reference"`
	Item struct {
		MarketHashName string  `json:"market_hash_name"`
		Wear           float64 `json:"float_value"`
	} `json:"item"`
}

type SkinsMonitor struct {
	Client   *http.Client
	BaseURL  string
	APIKey   string
	MinPrice float64
	MaxPrice float64
}

func NewSkinsMonitor(minPrice, maxPrice float64) *SkinsMonitor {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error al cargar el archivo .env:", err)
	}

	key := os.Getenv("CSFLOAT_API_KEY")

	return &SkinsMonitor{
		Client:   &http.Client{Timeout: 15 * time.Second},
		BaseURL:  "https://csfloat.com/api/v1/listings",
		APIKey:   key,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}
}

func (sm *SkinsMonitor) FetchWithAuth() error {
	if sm.APIKey == "" {
		return fmt.Errorf("la variable de entorno CSFLOAT_API_KEY está vacía. Configúrala en tu PowerShell")
	}

	fmt.Println("🔑 Autenticando petición con tu API Key oficial...")

	url := fmt.Sprintf("%s?limit=30&sort_by=most_recent", sm.BaseURL)
	if sm.MinPrice > 0 {
		url = fmt.Sprintf("%s&min_price=%d", url, int64(sm.MinPrice*100))
	}
	if sm.MaxPrice > 0 {
		url = fmt.Sprintf("%s&max_price=%d", url, int64(sm.MaxPrice*100))
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error al crear la petición: %v", err)
	}

	req.Header.Set("Authorization", sm.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CS2-Arbitrage-App/2.0")

	resp, err := sm.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error de conexión: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error en la API de CSFloat. Código de estado: %d", resp.StatusCode)
	}

	var listings CSFloatResponse
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return fmt.Errorf("error al decodificar JSON: %v", err)
	}

	fmt.Printf("✅ Conexión exitosa. Analizando %d skins en tiempo real...\n\n", len(listings.Data))
	fmt.Println("=================================================================")

	for _, list := range listings.Data {
		csFloatPrice := float64(list.Price) / 100.0
		steamPrice := float64(list.Reference.BasePrice) / 100.0

		if steamPrice > 0 {
			discount := ((steamPrice - csFloatPrice) / steamPrice) * 100

			fmt.Printf("🎯 Skin: %s\n", list.Item.MarketHashName)
			fmt.Printf("   ├─ Float/Wear: %.5f\n", list.Item.Wear)
			fmt.Printf("   ├─ Precio en CSFloat: $%.2f USD\n", csFloatPrice)
			fmt.Printf("   ├─ Referencia Steam:  $%.2f USD\n", steamPrice)

			fmt.Printf("   📈 Descuento: %.2f%% más barato en CSFloat\n", discount)

			fmt.Printf("   📉 Sobreprecio: %.2f%% más caro que en Steam\n", -discount)

			fmt.Println("====================================================================================================================")
		}
	}

	return nil
}

func main() {

	var max float64 = 1
	var min float64 = 0.03
	monitor := NewSkinsMonitor(min, max)
	if err := monitor.FetchWithAuth(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	}
}

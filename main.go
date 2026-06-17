package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
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

type HistoryListing struct {
	ID        string `json:"id"`
	Price     int64  `json:"price"`
	CreatedAt string `json:"created_at"`
	Item      struct {
		MarketHashName string  `json:"market_hash_name"`
		Wear           float64 `json:"float_value"`
	} `json:"item"`
}

func (sm *SkinsMonitor) FetchHistory(itemName string) error {
	if sm.APIKey == "" {
		return fmt.Errorf("la variable de entorno CSFLOAT_API_KEY está vacía")
	}

	fmt.Printf("📈 Obteniendo historial de precios para: %s...\n", itemName)

	escapedName := url.PathEscape(itemName)
	apiURL := fmt.Sprintf("https://csfloat.com/api/v1/history/%s/sales", escapedName)

	req, err := http.NewRequest("GET", apiURL, nil)
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

	var history []HistoryListing
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return fmt.Errorf("error al decodificar JSON: %v", err)
	}

	fmt.Printf("✅ Se encontraron %d ventas recientes.\n\n", len(history))
	fmt.Println("====================================================================================================================")
	for i, sale := range history {
		priceUSD := float64(sale.Price) / 100.0
		t, err := time.Parse(time.RFC3339, sale.CreatedAt)
		dateStr := sale.CreatedAt
		if err == nil {
			dateStr = t.Local().Format("2006-01-02 15:04:05")
		}

		fmt.Printf("[%02d] Fecha: %s | Precio: $%.2f USD | Float/Wear: %.5f\n", i+1, dateStr, priceUSD, sale.Item.Wear)
	}
	fmt.Println("====================================================================================================================")

	// Agrupar por día (YYYY-MM-DD) y calcular promedios
	type DailyStats struct {
		TotalPrice int64
		Count      int
	}
	dailyMap := make(map[string]*DailyStats)
	var dates []string

	for _, sale := range history {
		t, err := time.Parse(time.RFC3339, sale.CreatedAt)
		if err != nil {
			continue
		}
		dayStr := t.Local().Format("2006-01-02")
		if stats, exists := dailyMap[dayStr]; exists {
			stats.TotalPrice += sale.Price
			stats.Count++
		} else {
			dailyMap[dayStr] = &DailyStats{TotalPrice: sale.Price, Count: 1}
			dates = append(dates, dayStr)
		}
	}

	// Ordenar las fechas de forma cronológica
	sort.Strings(dates)

	fmt.Println("\n📊 Resumen de variación de precios promedio por día:")
	fmt.Println("====================================================================================================================")
	for _, date := range dates {
		stats := dailyMap[date]
		avgPrice := (float64(stats.TotalPrice) / float64(stats.Count)) / 100.0
		fmt.Printf("📅 Día: %s | Precio Promedio: $%.2f USD | Ventas registradas: %d\n", date, avgPrice, stats.Count)
	}
	fmt.Println("====================================================================================================================")

	return nil
}

type GraphPoint struct {
	Day      string  `json:"day"`
	AvgPrice float64 `json:"avg_price"`
	Count    int     `json:"count"`
}

func (sm *SkinsMonitor) FetchHistoryGraph(itemName string) error {
	if sm.APIKey == "" {
		return fmt.Errorf("la variable de entorno CSFLOAT_API_KEY está vacía")
	}

	fmt.Printf("📊 Obteniendo datos del gráfico de precios para: %s...\n", itemName)

	escapedName := url.PathEscape(itemName)
	apiURL := fmt.Sprintf("https://csfloat.com/api/v1/history/%s/graph", escapedName)

	req, err := http.NewRequest("GET", apiURL, nil)
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

	var graph []GraphPoint
	if err := json.NewDecoder(resp.Body).Decode(&graph); err != nil {
		return fmt.Errorf("error al decodificar JSON: %v", err)
	}

	fmt.Printf("✅ Se obtuvieron %d puntos diarios de historial.\n\n", len(graph))
	fmt.Println("====================================================================================================================")
	for i := len(graph) - 1; i >= 0; i-- {
		point := graph[i]
		priceUSD := point.AvgPrice / 100.0

		t, err := time.Parse(time.RFC3339, point.Day)
		dateStr := point.Day
		if err == nil {
			dateStr = t.Format("2006-01-02")
		}

		fmt.Printf("📅 Día: %s | Precio Promedio: $%.2f USD | Ventas registradas: %d\n", dateStr, priceUSD, point.Count)
	}
	fmt.Println("====================================================================================================================")

	return nil
}

func main() {
	minPrice := flag.Float64("min", 0.0, "Precio mínimo en USD")
	maxPrice := flag.Float64("max", 0.0, "Precio máximo en USD")
	historyItem := flag.String("history", "", "Nombre de la skin para ver sus últimas 40 ventas")
	graphItem := flag.String("graph", "", "Nombre de la skin para ver su historial de precios diario por meses")
	flag.Parse()

	// Si no se especifican flags de precio, se usan los valores por defecto del usuario
	min := *minPrice
	max := *maxPrice
	if *historyItem == "" && *graphItem == "" && *minPrice == 0.0 && *maxPrice == 0.0 {
		min = 0.03
		max = 1.0
	}

	monitor := NewSkinsMonitor(min, max)

	if *graphItem != "" {
		if err := monitor.FetchHistoryGraph(*graphItem); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	} else if *historyItem != "" {
		if err := monitor.FetchHistory(*historyItem); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	} else {
		if err := monitor.FetchWithAuth(); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}
}

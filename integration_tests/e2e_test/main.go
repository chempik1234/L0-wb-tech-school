package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Order generated from json
type Order struct {
	OrderUID          string   `json:"order_uid"`
	TrackNumber       string   `json:"track_number"`
	Entry             string   `json:"entry"`
	Delivery          Delivery `json:"delivery"`
	Payment           Payment  `json:"payment"`
	Items             []Item   `json:"items"`
	Locale            string   `json:"locale"`
	InternalSignature string   `json:"internal_signature"`
	CustomerID        string   `json:"customer_id"`
	DeliveryService   string   `json:"delivery_service"`
	Shardkey          string   `json:"shardkey"`
	SmID              int      `json:"sm_id"`
	DateCreated       string   `json:"date_created"`
	OofShard          string   `json:"oof_shard"`
}

type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDt    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

// TestResult is a result of testing 1 order
type TestResult struct {
	OrderUID      string
	FirstRequest  time.Duration
	SecondRequest time.Duration
	IsCached      bool
	Error         error
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := context.Background()
	baseURL := os.Getenv("INTEGRATION_TESTS_BASE_URL")

	// Генерируем 50 случайных заказов
	orders := generateRandomOrders(50)
	logger.Info("Generated 50 random orders", zap.Int("count", len(orders)))

	// Шаг 1: Отправляем заказы через POST
	if err := sendOrders(ctx, baseURL+"/simulator/create", orders); err != nil {
		logger.Fatal("Failed to send orders", zap.Error(err))
	}
	logger.Info("Successfully sent 50 orders to simulator")

	// Шаг 2: Ждем 3 секунды для обработки
	logger.Info("Waiting 3 seconds for processing...")
	time.Sleep(3 * time.Second)

	// Шаг 3: Опрашиваем эндпоинты и измеряем время
	results := testCachePerformance(ctx, baseURL+"/api/order/", orders)

	// Анализируем результаты
	analyzeResults(results, logger)
}

// generateRandomOrders создает n случайных заказов
func generateRandomOrders(n int) []Order {
	orders := make([]Order, n)

	// Данные для генерации (аналогично вашему JS скрипту)
	names := []string{"Иван", "Петр", "Мария", "Анна", "Сергей", "Ольга", "Дмитрий", "Екатерина"}
	surnames := []string{"Иванов", "Петров", "Сидоров", "Смирнов", "Кузнецов", "Попов", "Васильев"}
	cities := []string{"Москва", "Санкт-Петербург", "Новосибирск", "Екатеринбург", "Казань", "Нижний Новгород"}
	streets := []string{"Ленина", "Пушкина", "Гагарина", "Советская", "Мира", "Центральная"}
	products := []struct {
		Name  string
		Brand string
		Price int
	}{
		{"Смартфон", "Xiaomi", 25000},
		{"Ноутбук", "Lenovo", 45000},
		{"Наушники", "Sony", 8000},
		{"Часы", "Apple", 35000},
		{"Планшет", "Samsung", 28000},
		{"Фотоаппарат", "Canon", 55000},
		{"Телевизор", "LG", 40000},
		{"Игровая консоль", "PlayStation", 30000},
	}
	banks := []string{"alpha", "sber", "tinkoff", "vtb", "gazprom"}
	currencies := []string{"USD", "EUR", "RUB", "GBP"}
	deliveryServices := []string{"meest", "cdek", "dhl", "ups", "fedex"}
	locales := []string{"en", "ru", "de", "fr"}

	for i := 0; i < n; i++ {
		name := names[i%len(names)]
		surname := surnames[i%len(surnames)]
		city := cities[i%len(cities)]
		street := streets[i%len(streets)]

		orderUID := fmt.Sprintf("test_order_%d_%d", time.Now().Unix(), i)
		trackNumber := fmt.Sprintf("WBILTEST%d", i)

		// Генерируем 1-3 товара в заказе
		itemCount := 1 + (i % 3)
		items := make([]Item, itemCount)
		totalGoods := 0

		for j := 0; j < itemCount; j++ {
			product := products[(i+j)%len(products)]
			sale := (i * j) % 50
			totalPrice := product.Price * (100 - sale) / 100

			items[j] = Item{
				ChrtID:      1000000 + i*10 + j,
				TrackNumber: trackNumber,
				Price:       product.Price,
				Rid:         fmt.Sprintf("rid_%d_%d", i, j),
				Name:        product.Name,
				Sale:        sale,
				Size:        fmt.Sprintf("%d", j),
				TotalPrice:  totalPrice,
				NmID:        2000000 + i*10 + j,
				Brand:       product.Brand,
				Status:      202,
			}

			totalGoods += totalPrice
		}

		deliveryCost := 500 + (i % 1500)
		customFee := i % 500
		totalAmount := totalGoods + deliveryCost + customFee

		orders[i] = Order{
			OrderUID:    orderUID,
			TrackNumber: trackNumber,
			Entry:       "WBIL",
			Delivery: Delivery{
				Name:    fmt.Sprintf("%s %s", name, surname),
				Phone:   fmt.Sprintf("+7916%07d", 1000000+i),
				Zip:     fmt.Sprintf("%06d", 100000+i),
				City:    city,
				Address: fmt.Sprintf("ул. %s, д. %d", street, 1+i%100),
				Region:  fmt.Sprintf("%sская область", city),
				Email:   fmt.Sprintf("%s.%s@gmail.com", name, surname),
			},
			Payment: Payment{
				Transaction:  orderUID,
				RequestID:    "",
				Currency:     currencies[i%len(currencies)],
				Provider:     "wbpay",
				Amount:       totalAmount,
				PaymentDt:    time.Now().Unix() - int64(i*100),
				Bank:         banks[i%len(banks)],
				DeliveryCost: deliveryCost,
				GoodsTotal:   totalGoods,
				CustomFee:    customFee,
			},
			Items:             items,
			Locale:            locales[i%len(locales)],
			InternalSignature: "",
			CustomerID:        fmt.Sprintf("user%d", 1000+i),
			DeliveryService:   deliveryServices[i%len(deliveryServices)],
			Shardkey:          fmt.Sprintf("%d", 1+i%9),
			SmID:              1 + i%999,
			DateCreated:       time.Now().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
			OofShard:          fmt.Sprintf("%d", 1+i%3),
		}
	}

	return orders
}

// sendOrders отправляет заказы через POST
func sendOrders(ctx context.Context, url string, orders []Order) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for _, order := range orders {
		jsonData, err := json.Marshal(order)
		if err != nil {
			return fmt.Errorf("failed to marshal order %s: %w", order.OrderUID, err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request for order %s: %w", order.OrderUID, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send order %s: %w", order.OrderUID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status for order %s: %d, body: %s",
				order.OrderUID, resp.StatusCode, string(body))
		}

		// Читаем ответ чтобы connection мог быть переиспользован
		io.Copy(io.Discard, resp.Body)
	}

	return nil
}

// testCachePerformance тестирует производительность кэша
func testCachePerformance(ctx context.Context, baseURL string, orders []Order) []TestResult {
	results := make([]TestResult, len(orders))
	var wg sync.WaitGroup
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for i, order := range orders {
		wg.Add(1)
		go func(idx int, order Order) {
			defer wg.Done()

			url := baseURL + order.OrderUID
			result := TestResult{OrderUID: order.OrderUID}

			// Первый запрос (должен быть медленнее)
			start := time.Now()
			order1, err := getOrder(client, url)
			result.FirstRequest = time.Since(start)

			if err != nil {
				result.Error = err
				results[idx] = result
				return
			}

			// Проверяем, что данные совпадают
			if !compareOrders(order, order1) {
				result.Error = fmt.Errorf("order data mismatch")
				results[idx] = result
				return
			}

			// Второй запрос (должен быть быстрее благодаря кэшу)
			start = time.Now()
			order2, err := getOrder(client, url)
			result.SecondRequest = time.Since(start)

			if err != nil {
				result.Error = err
			} else if !compareOrders(order, order2) {
				result.Error = fmt.Errorf("cached order data mismatch")
			} else {
				// Проверяем что второй запрос был быстрее
				result.IsCached = result.SecondRequest < result.FirstRequest
			}

			results[idx] = result
		}(i, order)
	}

	wg.Wait()
	return results
}

// getOrder выполняет GET запрос для получения заказа
func getOrder(client *http.Client, url string) (Order, error) {
	var order Order

	resp, err := client.Get(url)
	if err != nil {
		return order, fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return order, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return order, fmt.Errorf("failed to decode response: %w", err)
	}

	return order, nil
}

// compareOrders сравнивает два заказа на идентичность
func compareOrders(a, b Order) bool {
	// Сравниваем только ключевые поля для производительности
	return a.OrderUID == b.OrderUID &&
		a.TrackNumber == b.TrackNumber &&
		a.Payment.Transaction == b.Payment.Transaction &&
		a.Payment.Amount == b.Payment.Amount &&
		len(a.Items) == len(b.Items)
}

// analyzeResults анализирует и выводит результаты тестирования
func analyzeResults(results []TestResult, logger *zap.Logger) {
	total := len(results)
	successful := 0
	cached := 0
	var totalFirstTime, totalSecondTime time.Duration

	for _, result := range results {
		if result.Error == nil {
			successful++
			if result.IsCached {
				cached++
			}
			totalFirstTime += result.FirstRequest
			totalSecondTime += result.SecondRequest
		} else {
			logger.Error("Test failed for order",
				zap.String("order_uid", result.OrderUID),
				zap.Error(result.Error))
		}
	}

	avgFirst := time.Duration(0)
	avgSecond := time.Duration(0)
	if successful > 0 {
		avgFirst = totalFirstTime / time.Duration(successful)
		avgSecond = totalSecondTime / time.Duration(successful)
	}

	cacheEfficiency := 0.0
	if avgFirst > 0 {
		cacheEfficiency = (1 - float64(avgSecond)/float64(avgFirst)) * 100
	}

	logger.Info("Integration Test Results",
		zap.Int("total_orders", total),
		zap.Int("successful", successful),
		zap.Int("cached_improvement", cached),
		zap.Duration("avg_first_request", avgFirst),
		zap.Duration("avg_second_request", avgSecond),
		zap.Float64("cache_efficiency_percent", cacheEfficiency),
		zap.Int("failures", total-successful))

	// Сохраняем результаты в файл для дальнейшего анализа
	saveResultsToFile(results, "integration_test_results.json")
}

// saveResultsToFile сохраняет результаты в JSON файл
func saveResultsToFile(results []TestResult, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Failed to create results file: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		log.Printf("Failed to encode results: %v", err)
	}
}

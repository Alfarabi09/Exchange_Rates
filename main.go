package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

// ValCurs представляет корневой элемент XML от ЦБ РФ с информацией о курсах валют
type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    string   `xml:"Date,attr"` // Дата курса валют
	Valutes []Valute `xml:"Valute"`    // Список валют
}

// Valute содержит информацию о конкретной валюте
type Valute struct {
	ID       string `xml:"ID,attr"`  // ID валюты
	NumCode  string `xml:"NumCode"`  // Цифровой код валюты
	CharCode string `xml:"CharCode"` // Символьный код валюты
	Nominal  int    `xml:"Nominal"`  // Номинал валюты
	Name     string `xml:"Name"`     // Название валюты
	Value    string `xml:"Value"`    // Значение курса валюты
}

// CurrencyStats хранит статистику по курсам валюты
type CurrencyStats struct {
	MaxValue     float64 // Максимальное значение курса
	MinValue     float64 // Минимальное значение курса
	MaxDate      string  // Дата максимального курса
	MinDate      string  // Дата минимального курса
	TotalValue   float64 // Суммарное значение курса для расчета среднего
	Count        int     // Количество записей для расчета среднего
	Average      float64 // Среднее значение курса
	Nominal      int     // Номинал валюты
	CurrencyName string  // Название валюты
	NumCode      string  // Цифровой код валюты
	CharCode     string  // Символьный код валюты
}

var globalStats = make(map[string]*CurrencyStats) // Глобальный map для хранения статистики по валютам

// fetchCurrencyRates выполняет запрос к API ЦБ РФ и возвращает XML с данными о курсах валют
func fetchCurrencyRates(url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("Ошибка при создании запроса: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ошибка при запросе к API: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Ошибка при чтении ответа: %w", err)
	}

	return string(body), nil
}

// parseXML анализирует XML и возвращает структуру ValCurs с данными о курсах валют
func parseXML(data string) (ValCurs, error) {
	var valCurs ValCurs
	reader := bytes.NewReader([]byte(data))
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel // Для обработки кодировки windows-1251

	err := decoder.Decode(&valCurs)
	if err != nil {
		return ValCurs{}, err
	}

	return valCurs, nil
}

// analyzeData анализирует данные о курсах валют и обновляет статистику в globalStats
func analyzeData(valCurs ValCurs) {
	for _, valute := range valCurs.Valutes {
		valueStr := strings.Replace(valute.Value, ",", ".", -1) // Заменяем запятую на точку для преобразования в float
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			fmt.Printf("Ошибка при преобразовании курса валюты %s: %v\n", valute.CharCode, err)
			continue
		}

		// Добавление или обновление статистики по валюте в globalStats
		stats, ok := globalStats[valute.CharCode]
		if !ok {
			globalStats[valute.CharCode] = &CurrencyStats{
				MaxValue:     value,
				MinValue:     value,
				MaxDate:      valCurs.Date,
				MinDate:      valCurs.Date,
				TotalValue:   value,
				Count:        1,
				Nominal:      valute.Nominal,
				CurrencyName: valute.Name,
				NumCode:      valute.NumCode,
				CharCode:     valute.CharCode,
			}
		} else {
			stats.TotalValue += value
			stats.Count++
			if value > stats.MaxValue {
				stats.MaxValue = value
				stats.MaxDate = valCurs.Date
			}
			if value < stats.MinValue {
				stats.MinValue = value
				stats.MinDate = valCurs.Date
			}
		}
	}
}

func main() {
	baseUrl := "http://www.cbr.ru/scripts/XML_daily_eng.asp?date_req=%s"
	startDate := time.Now().AddDate(0, 0, -90)

	for d := startDate; d.Before(time.Now()); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("08/03/2024") // Форматирование даты для запроса
		url := fmt.Sprintf(baseUrl, dateStr)

		xmlData, err := fetchCurrencyRates(url) // Получение данных о курсах валют
		if err != nil {
			fmt.Println(err)
			continue
		}

		valCurs, err := parseXML(xmlData) // Разбор полученных данных
		if err != nil {
			fmt.Printf("Ошибка при разборе XML для даты %s: %v\n", dateStr, err)
			continue
		}

		analyzeData(valCurs) // Анализ данных и обновление статистики
	}

	// Вывод собранной статистики по каждой валюте
	for _, stats := range globalStats {
		stats.Average = stats.TotalValue / float64(stats.Count) // Расчёт среднего значения курса
		fmt.Printf("%s (%s, %s) - Nominal: %d, Max: %f (%s), Min: %f (%s), Average: %f\n",
			stats.CurrencyName, stats.CharCode, stats.NumCode, stats.Nominal,
			stats.MaxValue, stats.MaxDate, stats.MinValue, stats.MinDate, stats.Average)
	}
}

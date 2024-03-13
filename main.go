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

type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    string   `xml:"Date,attr"`
	Valutes []Valute `xml:"Valute"`
}

type Valute struct {
	ID       string `xml:"ID,attr"`
	NumCode  string `xml:"NumCode"`
	CharCode string `xml:"CharCode"`
	Nominal  int    `xml:"Nominal"`
	Name     string `xml:"Name"`
	Value    string `xml:"Value"`
}

type CurrencyStats struct {
	MaxValue     float64
	MinValue     float64
	MaxDate      string
	MinDate      string
	TotalValue   float64
	Count        int
	Average      float64
	CurrencyName string
}

var globalStats = make(map[string]*CurrencyStats)

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

func parseXML(data string) (ValCurs, error) {
	var valCurs ValCurs
	reader := bytes.NewReader([]byte(data))
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel // Для обработки windows-1251

	err := decoder.Decode(&valCurs)
	if err != nil {
		return ValCurs{}, err
	}

	return valCurs, nil
}

func analyzeData(valCurs ValCurs) {
	for _, valute := range valCurs.Valutes {
		valueStr := strings.Replace(valute.Value, ",", ".", -1)
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			fmt.Printf("Ошибка при преобразовании курса валюты %s: %v\n", valute.CharCode, err)
			continue
		}

		if stats, ok := globalStats[valute.CharCode]; !ok {
			globalStats[valute.CharCode] = &CurrencyStats{
				MaxValue:     value,
				MinValue:     value,
				MaxDate:      valCurs.Date,
				MinDate:      valCurs.Date,
				TotalValue:   value,
				Count:        1,
				CurrencyName: valute.Name,
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
	totalDays := 0

	for d := startDate; d.Before(time.Now()); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("02/01/2006")
		url := fmt.Sprintf(baseUrl, dateStr)

		xmlData, err := fetchCurrencyRates(url)
		if err != nil {
			fmt.Println(err)
			continue
		}

		valCurs, err := parseXML(xmlData)
		if err != nil {
			fmt.Printf("Ошибка при разборе XML для даты %s: %v\n", dateStr, err)
			continue
		}

		analyzeData(valCurs)
		totalDays++
	}

	for _, stats := range globalStats {
		stats.Average = stats.TotalValue / float64(stats.Count)
		fmt.Printf("%s - Макс: %f (%s), Мин: %f (%s), Среднее: %f\n", stats.CurrencyName, stats.MaxValue, stats.MaxDate, stats.MinValue, stats.MinDate, stats.Average)

	}
}

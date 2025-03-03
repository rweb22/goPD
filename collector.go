package main

import (
	"encoding/json"
	"bufio"
	"fmt"
	"os"
	"time"
	"net/http"
	"io"
	"sort"
	"strconv"
)

const(
	MarketOpen = "09:15"
	MarketClose = "15:30"
)

var loc *time.Location

func tyme() time.Time {
	return time.Now().In(loc)
}

func loadHolidays() map[string]bool {
	holidays := make(map[string]bool)
	file, err := os.Open("holidays.txt")
	if err != nil {
		fmt.Println("Unable to load Holidays file.")
		return holidays
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		holidays[scanner.Text()] = true
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading holidays file: ", err)
	}

	return holidays
}

func isMarketOpen() bool {
	holidays := loadHolidays()
	now := tyme()
	marketOpen, _ := time.Parse("15:04", MarketOpen)
	marketClose, _ := time.Parse("15:04", MarketClose)
	currentTime, _ := time.Parse("15:04", now.Format("15:04"))

	return currentTime.After(marketOpen) && currentTime.Before(marketClose) && (now.Weekday() != time.Saturday) && (now.Weekday() != time.Sunday) && (!holidays[now.Format("2006-01-02")])
}

func getNextMarketOpen() time.Time {
	holidays := loadHolidays()
	now := tyme()
	nextTradeDay := now

	if now.Format("15:04") >= MarketClose {
		nextTradeDay = now.Add(24 * time.Hour)
	}

	for nextTradeDay.Weekday() == time.Saturday || nextTradeDay.Weekday() == time.Sunday || holidays[nextTradeDay.Format("2006-01-02")] {
		nextTradeDay = nextTradeDay.Add(24 * time.Hour)
	}

	nextOpen := time.Date(nextTradeDay.Year(), nextTradeDay.Month(), nextTradeDay.Day(), 9, 15, 0, 0, loc)
	return nextOpen
}

func fetchPD(optionsData []interface{}, callToken int, putToken int) (int, int) {
	callDone, putDone := false, false
	pd, pdv := 0, 0

	for _, opt := range optionsData {
		option := opt.(map[string]interface{})
		if callToken == int(option["token"].(float64)) {
			ltp := option["last_price"].(float64)
			oi := option["oi"].(float64)
			volume := option["volume"].(float64)
			pd -= int(ltp * oi)
			pdv -= int(ltp * volume)
			callDone = true
			if putDone {
				break
			}
		} else if putToken == int(option["token"].(float64)) {
			ltp := option["last_price"].(float64)
			oi := option["oi"].(float64)
			volume := option["volume"].(float64)
			pd += int(ltp * oi)
			pdv += int(ltp * volume)
			putDone = true
			if callDone {
				break
			}
		}
	}

	return pd, pdv
}

func fetchStockData(pdDatas map[string]interface{}, metacacheData map[int]map[string]int, expiryDate string, stockToken int, strikeDiff int) error {
	url := fmt.Sprintf("https://oxide.sensibull.com/v1/compute/cache/live_derivative_prices/%d", stockToken)
	retries := 15
	data := make(map[string]interface{})
	var optionsData []interface{}
	var atm_strike int
	var ctime string

	for i:=1; i<= retries; i++ {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error fetching stock data: ", err)
			time.Sleep(2 * time.Second)
			continue
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error getting stock body: ", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Println("Error parsing StockData JSON: ", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if _, ok := data["data"]; ok {
			if dataMap, ok := data["data"].(map[string]interface{}); ok {
				t, err := time.Parse(time.RFC3339, dataMap["last_updated_at"].(string))
				if err != nil {
					fmt.Println("Error parsing time:", err)
				} else {
					ctime = t.Format("15:04")
				}

				stockPrice := dataMap["underlying_price"].(float64)
				pdDatas["stock_price"].(map[string]float64)[ctime] = stockPrice

				if expiryData, ok := dataMap["per_expiry_data"].(map[string]interface{}); ok {
					if expiryEntry, ok := expiryData[expiryDate].(map[string]interface{}); ok {
						atm_strike = int(expiryEntry["atm_strike"].(float64))
						if options, ok := expiryEntry["options"].([]interface{}); ok {
							optionsData = options
						}
					}
				}
			}
			pd, pdv := fetchPD(optionsData, metacacheData[atm_strike]["CE"], metacacheData[atm_strike]["PE"])
			pdDatas["PD"].(map[int]map[string]int)[1][ctime] = pd
			pdDatas["PDV"].(map[int]map[string]int)[1][ctime] = pdv
			for i:=1; i<=10; i++ {
				pdga, pdvga := fetchPD(optionsData, metacacheData[atm_strike - i*strikeDiff]["CE"], metacacheData[atm_strike - i*strikeDiff]["PE"])
				pdgb, pdvgb := fetchPD(optionsData, metacacheData[atm_strike + i*strikeDiff]["CE"], metacacheData[atm_strike + i*strikeDiff]["PE"])
				pd += pdga + pdgb
				pdv += pdvga + pdvgb
				pdDatas["PD"].(map[int]map[string]int)[2*i+1][ctime] = pd
				pdDatas["PDV"].(map[int]map[string]int)[2*i+1][ctime] = pdv
			}
			return nil
		}
	}
	errr := fmt.Errorf("Failed to access Stock Data.")
	return errr
}

func fetchOptionsData(pdDatas map[string]map[string]interface{}, metacacheData map[string]map[int]map[string]int, stockTokens map[string]int, expiryDates map[string]string, strikeDiffs map[string]int) error {
	for key, value := range stockTokens {
		err := fetchStockData(pdDatas[key], metacacheData[key], expiryDates[key], value, strikeDiffs[key])
		if err != nil {
			return err
		}
	}

	now := tyme()
	dateString := now.Format("2006-01-02")
	fileName := fmt.Sprintf("%s.json", dateString)

	jsonData, err := json.MarshalIndent(pdDatas, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		panic(err)
	}

	println("Today's JSON file has been updated.")

	return nil
}

func extractMetacacheData(data map[string]interface{}, metacacheData map[int]map[string]int, expiryDates map[string]string, stock string) bool {
	var dateKeys []string

	for key := range data {
		_, err := time.Parse("2006-01-02", key)
		if err == nil {
			dateKeys = append(dateKeys, key)
		}
	}

	if len(dateKeys) == 0 {
		fmt.Println("Error Parsing Expiry Dates.")
		return false
	}

	sort.Strings(dateKeys)

	var k string = dateKeys[0]
	expiryDates[stock] = k

	iData := data[k].(map[string]interface{})["options"]
	insData := iData.(map[string]interface{})

	for strike := range insData {
		strikeFloat, err := strconv.ParseFloat(strike, 64)
		if err != nil {
			fmt.Println("Error Parsing Strike Price.")
		}
		str := int(strikeFloat)
		callData := insData[strike].(map[string]interface{})["CE"]
		CallData := callData.(map[string]interface{})
		putData := insData[strike].(map[string]interface{})["PE"]
		PutData := putData.(map[string]interface{})
		metacacheData[str] = map[string]int {}
		metacacheData[str]["CE"] = int(CallData["instrument_token"].(float64))
		metacacheData[str]["PE"] = int(PutData["instrument_token"].(float64))
	}
	return true
}

func fetchDataLoop(pdDatas map[string]map[string]interface{}, stopChan <-chan os.Signal) error {
	metacacheURL := "https://oxide.sensibull.com/v1/compute/cache/instrument_metacache/2"
	var data map[string]interface{}
	mc := false
	st := 0
	metacacheData := make(map[string]map[int]map[string]int)
	stockTokens := make(map[string]int)
	expiryDates := make(map[string]string)
	strikeDiffs := make(map[string]int)
	strikeDiffs["BANKNIFTY"] = 100
	strikeDiffs["NIFTYNXT50"] = 100
	strikeDiffs["RELIANCE"] = 10
	var retries int = 15

	for i:=1; i<=retries; i++ {
		mc = false
		st = 0
		fmt.Printf("Attempt %d...\n", i)

		resp, err := http.Get(metacacheURL)
		if err != nil {
			fmt.Println("Metacache Request Error: ", err)
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading Metacache response: ", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Println("Error parsing Metacache JSON: ", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if _, ok := data["derivatives"]; ok {
			fmt.Println("✅ Successfully retrieved valid data!")
			metacacheData["BANKNIFTY"] = make(map[int]map[string]int)
			metacacheData["NIFTYNXT50"] = map[int]map[string]int {}
			metacacheData["RELIANCE"] = map[int]map[string]int {}
			var b bool = true
			if derivatives, ok := data["derivatives"].(map[string]interface{}); ok {
				for _, key := range []string{"BANKNIFTY", "NIFTYNXT50", "RELIANCE"} {
					if symbolData, ok := derivatives[key].(map[string]interface{}); ok {
						if symbolDerivatives, ok := symbolData["derivatives"].(map[string]interface{}); ok {
							b = b && extractMetacacheData(symbolDerivatives, metacacheData[key], expiryDates, key)
						}
					}
				}
			}

			if b == false {
				fmt.Println("Error extracting Instrument Tokens.")
			} else {
				mc = true
				nse_list := data["underlyer_list"].(map[string]interface{})["NSE"]
				if nse, ok := nse_list.(map[string]interface{}); ok {
					if nseIndices, ok := nse["NSE-INDICES"].(map[string]interface{}); ok {
						if eq, ok := nseIndices["EQ"].(map[string]interface{}); ok {
							for _, stock := range []string{"BANKNIFTY", "NIFTYNXT50"} {
								if stockData, ok := eq[stock].(map[string]interface{}); ok {
									if token, ok := stockData["instrument_token"].(float64); ok {
										stockTokens[stock] = int(token)
										st++
									}
								}
							}
						}
					}

					if nseInner, ok := nse["NSE"].(map[string]interface{}); ok {
						if eq, ok := nseInner["EQ"].(map[string]interface{}); ok {
							if reliance, ok := eq["RELIANCE"].(map[string]interface{}); ok {
								if token, ok := reliance["instrument_token"].(float64); ok {
									stockTokens["RELIANCE"] = int(token)
									st++
								}
							}
						}
					}
				}
				break
			}
		}

		fmt.Println("Received empty or invalid data, retrying...")
		time.Sleep(2 * time.Second)
	}

	if !mc || (st < 3) {
		fmt.Println("❌ Failed to fetch valid data after", retries, "attempts.")
		return nil
	}

	fmt.Println("Tokens Collected. Launching the next phase...")

	tym := tyme()

	endTime := time.Date(tym.Year(), tym.Month(), tym.Day(), 16, 25, 0, 0, loc)
	for {
		now := tyme()
		if now.After(endTime) {
			fmt.Println("Data fetching completed for the day.")
			return nil
		}

		select {
		case <- stopChan:
			fmt.Println("Stopping Data Collector...")
			return fmt.Errorf("Shutdown signal received...")
		case <-time.After(1 * time.Minute):
			err := fetchOptionsData(pdDatas, metacacheData, stockTokens, expiryDates, strikeDiffs)
			if err != nil {
				fmt.Printf("Error fetching data: %v\n", err)
			}
		}
	}
}

func StartCollector(stopChan <-chan os.Signal) (time.Time, error) {
	var err error
	loc, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		fmt.Println("Error loading location:", err)
	}

	var nextOpen time.Time
	if (!isMarketOpen()) {
		nextOpen = getNextMarketOpen()
		return nextOpen, nil
	}

	pdDatas := make(map[string]map[string]interface{})
	stocks := []string{"BANKNIFTY", "NIFTYNXT50", "RELIANCE"}

	for _, stock := range stocks {
		pdDatas[stock] = make(map[string]interface{})
		pdDatas[stock]["stock_price"] = make(map[string]float64)
		pdDatas[stock]["PD"] = make(map[int]map[string]int)
		pdDatas[stock]["PDV"] = make(map[int]map[string]int)
	}

	for i:=0; i<=10; i++ {
		j := 2*i+1
		for _, stock := range stocks {
			pdDatas[stock]["PD"].(map[int]map[string]int)[j] = make(map[string]int)
			pdDatas[stock]["PDV"].(map[int]map[string]int)[j] = make(map[string]int)
		}
	}

	if err = fetchDataLoop(pdDatas, stopChan); err != nil {
		fmt.Println(err)
		return tyme(), err
	}

	nextOpen = getNextMarketOpen()
	return nextOpen, nil
}

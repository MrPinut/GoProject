package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	_ "github.com/lib/pq"
)

const (
	KrakenStatusApi  = "https://api.kraken.com/0/public/SystemStatus"
	KrakenPairsData  = "https://api.kraken.com/0/public/AssetPairs"
	KrakenPairsPrice = "https://api.kraken.com/0/public/Ticker?pair="

	host     = "localhost"
	port     = "5432"
	user     = "toto"
	password = "mysecretpassword"
	dbname   = "mydatabase"
	schema   = "public"
)

type PairData struct {
	Altname    string `json:"altname"`
	PairStatus string `json:"status"`
	Token1     string `json:"base"`
	Token2     string `json:"quote"`
	Volume     string `json:"fee_volume_currency"`
}

type MyStruct struct {
	FieldMap map[string]PairData `json:"result"`
}

type Status struct {
	Result struct {
		ServeurStatus string `json:"status"`
		TimeStamp     string `json:"timestamp"`
	} `json:"result"`
}

type Prices struct {
	FieldMap map[string]Price `json:"result"`
}

type Price struct {
	ActualPrice  []string `json:"c"`
	Volume       []string `json:"v"`
	High         []string `json:"h"`
	Low          []string `json:"l"`
	OpeningPrice string   `json:"o"`
}

func getKrakenStatus() Status {
	// Requête de l'API de Kraken pour récupérer les informations sur les serveurs
	krakenStatus, err := http.Get(KrakenStatusApi)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(krakenStatus.Body)
	if err != nil {
		// Handle the error
		fmt.Printf("Error: %s", err)
		panic(err)
	}

	// Parse the JSON string
	var state Status
	errors := json.Unmarshal(body, &state)
	if errors != nil {
		// Handle the error
		fmt.Printf("Error: %s", errors)
		panic(err)
	}

	fmt.Println("Le cours actuel du Bitcoin en dollars est de : ", state.Result.ServeurStatus)

	return state
}

func getKrakenPair() (MyStruct, []string) {

	krakenData, err := http.Get(KrakenPairsData)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(krakenData.Body)
	if err != nil {
		// Handle the error
		fmt.Printf("Error: %s", err)
		panic(err)
	}

	var pairTest MyStruct
	errors := json.Unmarshal(body, &pairTest)
	if errors != nil {
		// Handle the error
		fmt.Printf("Error: %s", errors)
		panic(errors)
	}

	translationKeys := make([]string, 0, len(pairTest.FieldMap))

	// We only need the keys
	for key := range pairTest.FieldMap {
		translationKeys = append(translationKeys, key)
	}

	//fmt.Println(translationKeys)
	//fmt.Println(pairTest.FieldMap[translationKeys[3]].PairStatus)

	return pairTest, translationKeys

}

func getPairPrice(pairName string, ch chan Prices, wg *sync.WaitGroup) {

	krakenData, err := http.Get(KrakenPairsPrice + pairName)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(krakenData.Body)
	if err != nil {
		// Handle the error
		fmt.Printf("Error: %s", err)
		panic(err)
	}

	var price Prices
	errors := json.Unmarshal(body, &price)
	if errors != nil {
		// Handle the error
		fmt.Printf("Error: %s", errors)
		panic(errors)
	}
	ch <- price

	//fmt.Println(price.FieldMap[pairName].OpeningPrice)

	wg.Done()

}

func getAllPairs(mapKey []string) chan Prices {
	var wg sync.WaitGroup
	len := len(mapKey)
	wg.Add(len)
	ch := make(chan Prices, len)
	for x := 0; x < len; x++ {

		go getPairPrice(mapKey[x], ch, &wg)
	}
	wg.Wait()

	return ch
}

func main() {

	// Connexion à la base de données
	connectionString := "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	db, err := sql.Open("postgres", connectionString)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	sqlStat := "DROP TABLE coins"
	_, error := db.Exec(sqlStat)

	if error != nil {
		fmt.Println(error)
	}

	// Création de la table si elle n'existe pas
	sqlStat = "CREATE TABLE IF NOT EXISTS public.coins ( id SERIAL NOT NULL, altname character varying NOT NULL, status character varying, token1 character varying, token2 character varying, ActualPrice real, Volume real, High real, Low real, Open real,  PRIMARY KEY (id) ); ALTER TABLE IF EXISTS public.coins OWNER to toto;"
	_, errors := db.Exec(sqlStat)

	if errors != nil {
		fmt.Println(errors)
	}

	structure, list := getKrakenPair()

	ch := getAllPairs(list)

	lenght := len(ch)

	for a := 1; a <= lenght; a++ {
		tempo := <-ch

		translationKeys := make([]string, 0, len(tempo.FieldMap))

		// We only need the keys
		for key := range tempo.FieldMap {
			translationKeys = append(translationKeys, key)

		}

		//structure.FieldMap[translationKeys[0]].
		val1, _ := strconv.Atoi(tempo.FieldMap[translationKeys[0]].ActualPrice[0])
		val2, _ := strconv.Atoi(tempo.FieldMap[translationKeys[0]].Volume[0])
		val3, _ := strconv.Atoi(tempo.FieldMap[translationKeys[0]].High[0])
		val4, _ := strconv.Atoi(tempo.FieldMap[translationKeys[0]].Low[0])
		val5, _ := strconv.Atoi(tempo.FieldMap[translationKeys[0]].OpeningPrice)
		sqlStatement := fmt.Sprintf("INSERT INTO coins (id, altname, status,token1,token2,ActualPrice,Volume,High, Low, Open) VALUES (%d, '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d)", a, structure.FieldMap[translationKeys[0]].Altname, structure.FieldMap[translationKeys[0]].PairStatus, structure.FieldMap[translationKeys[0]].Token1, structure.FieldMap[translationKeys[0]].Token2, val1, val2, strconv.Atoi(tempo.FieldMap[translationKeys[0]].High[0]), val3, val4)
		_, err := db.Exec(sqlStatement)
		if err != nil {
			fmt.Println(err)
		}
		//fmt.Println(tempo.FieldMap[translationKeys[0]].OpeningPrice)
	}

	status := getKrakenStatus()

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, status.Result.ServeurStatus)
	})

	http.HandleFunc("/table", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT * FROM coins ORDER BY volume DESC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//sort.Slice(rows, func(i, j int) bool { return rows[i].price < rows[j].price })
		for rows.Next() {
			var col1, col2, col3, col4, col5, col6, col7, col8, col9, col10 string
			if err := rows.Scan(&col1, &col2, &col3, &col4, &col5, &col6, &col7, &col8, &col9, &col10); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "%s %s %s %s %s %s %s %s %s %s\n", col1, col2, col3, col4, col5, col6, col7, col8, col9, col10)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	//http.HandleFunc("/upload", uploadHandler)
	http.ListenAndServe(":8080", nil)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}

}

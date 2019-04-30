package main

import (
	"bufio"
	"os"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

var (
	fromStdin   = flag.Bool("s", true, "Read Stats JSON from STDIN")
	ankiConnect = flag.String("a", "http://127.0.0.1:8765/", "Host running AnkiConnect")
	cred        = flag.String("c", "path/to/serviceAccount.json", "Path to the credential")
)

func reqToAnkiConnect(ankiConnect string, body string) (*http.Response, error) {
	b := strings.NewReader(body)
	req, err := http.NewRequest("POST", ankiConnect, b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type TodayStats struct {
	Cards int `json:"cards"`
	Time  int `json:"time"`
}

func getTodayStatsFromAnkiConnect(ankiConnect string) (TodayStats, error) {
	// iOS アプリ等と Sync する
	r1, err := reqToAnkiConnect(ankiConnect, `{"action": "sync", "version": 6}`)
	if err != nil {
		return TodayStats{}, err
	}
	defer r1.Body.Close()

	r2, err := reqToAnkiConnect(ankiConnect, `{"action": "todayStats", "version": 6}`)
	if err != nil {
		return TodayStats{}, err
	}
	defer r2.Body.Close()

	var res struct {
		Result TodayStats `json:"result"`
		Error  string     `json:"error"`
	}

	if err := json.NewDecoder(r2.Body).Decode(&res); err != nil {
		return TodayStats{}, err
	}

	return res.Result, nil
}

func readTodayStatsFromStdin() (TodayStats, error) {
  stdin := bufio.NewScanner(os.Stdin)
  stdin.Scan()

  var stats TodayStats

	if err := json.Unmarshal(stdin.Bytes(), &stats); err != nil {
		return TodayStats{}, err
	}

	return stats, nil
}

func add(ctx context.Context, credFilePath string, stats TodayStats) error {
	sa := option.WithCredentialsFile(credFilePath)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	t := time.Now()
	docKey := t.Format("2006-01-02")
	collection := client.Collection("StudyRecords")
  doc := collection.Doc(docKey)
	if _, err := doc.Set(ctx, map[string]interface{}{
		"cards": stats.Cards,
		"time":  stats.Time,
	}); err != nil {
		return err
	}
	return nil
}

func getTodayStats() (TodayStats, error) {
  if *fromStdin {
	  return readTodayStatsFromStdin()
  } else {
	  return getTodayStatsFromAnkiConnect(*ankiConnect)
  }
}

func main() {
	flag.Parse()
  stats, err := getTodayStats()
	log.Println("Get a study record from Anki.")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Got: %v", stats)
	if err := add(context.Background(), *cred, stats); err != nil {
		log.Fatalln(err)
	}
}

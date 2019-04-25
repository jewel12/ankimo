package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// FirestoreEvent is the payload of a Firestore event.
type FirestoreEvent struct {
	OldValue   FirestoreValue `json:"oldValue"`
	Value      FirestoreValue `json:"value"`
	UpdateMask struct {
		FieldPaths []string `json:"fieldPaths"`
	} `json:"updateMask"`
}

// FirestoreValue holds Firestore fields.
type FirestoreValue struct {
	CreateTime time.Time `json:"createTime"`
	// Fields is the data for this value. The type depends on the format of your
	// database. Log an interface{} value and inspect the result to see a JSON
	// representation of your database fields.
	Fields     TodayStats `json:"fields"`
	Name       string     `json:"name"`
	UpdateTime time.Time  `json:"updateTime"`
}

type TodayStats struct {
	Cards struct {
		IntegerValue string `json:"integerValue"`
	} `json:"cards"`
	Time struct {
		IntegerValue string `json:"integerValue"`
	} `json:"time"`
}

var leastCards = 230

func Notify(ctx context.Context, e FirestoreEvent) error {
	log.Printf("ev: %v", e.Value.Fields.Cards)
	cards, err := strconv.Atoi(e.Value.Fields.Cards.IntegerValue)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	time, err := strconv.Atoi(e.Value.Fields.Time.IntegerValue)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	min := time / 60

	var msg string
	if cards < leastCards {
		msg = fmt.Sprintf(
			"今日は%d枚のカードしか勉強してないパカ！勉強時間は%d分パカ！しっかりするパカ", cards, min)
	} else {
		msg = fmt.Sprintf(
			"今日は%d枚のカードを勉強してるパカ。勉強時間は%d分パカ！", cards, min)
	}

	log.Printf("msg: %v", msg)
	if err := post(WebHookBody{msg}); err != nil {
	  log.Printf("post err: %v", err)
		log.Fatalln(err)
		return err
	}
	return nil
}

// for IFTTT webhook
type WebHookBody struct {
	Value1 string `json:"value1"`
}

func post(body WebHookBody) error {
	webhook := os.Getenv("ANKIMO_WEBHOOK")
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	b := strings.NewReader(string(jsonBody))
	req, err := http.NewRequest("POST", webhook, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if _, err = http.DefaultClient.Do(req); err != nil {
		return err
	}
	return nil
}

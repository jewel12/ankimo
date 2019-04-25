package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"log"
	"net/http"
	"os"
	"strings"
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

func getDeckNames(ankiConnect string) ([]string, error) {
	r, err := reqToAnkiConnect(ankiConnect, `{"action": "deckNames", "version": 6}`)
	if err != nil {
		return []string{}, err
	}
	defer r.Body.Close()

	var res struct {
		Names []string    `json:"result"`
		Error  string     `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return []string{}, err
	}

	return res.Names, nil
}

func findCardIDs(ankiConnect string, deck string) ([]int, error) {
	req := fmt.Sprintf(`{"action": "findCards", "version": 6, "params": {"query": "deck:%s"}}`, deck)
	r, err := reqToAnkiConnect(ankiConnect, req)
	if err != nil {
		return []int{}, err
	}
	defer r.Body.Close()

	var res struct {
		IDs   []int `json:"result"`
		Error []int `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return []int{}, err
	}

	return res.IDs, nil
}

type f struct {
	Value string `json:"value"`
	Order int    `json:"order"`
}
type CardInfo struct {
	Answer     string `json:"answer"`
	Question   string `json:"question"`
	DeckName   string `json:"deckName"`
	ModelName  string `json:"modelName"`
	FieldOrder int    `json:"fieldOrder"`
	Css        string `json:"css"`
	Interval   int    `json:"interval"`
	CardID     int    `json:"cardId"`
	Note       int 	  `json:"note"`
	Fields     struct {
		Front f `json:"Front"`
		Back  f `json:"Back"`
	} `json:"fields"`
}

func getCardsInfo(ankiConnect string, cards []int) ([]CardInfo, error) {
	cs, err := json.Marshal(cards)
	if err != nil {
		return []CardInfo{}, err
	}

	req := fmt.Sprintf(`{"action": "cardsInfo", "version": 6, "params": {"cards": %s}}`, string(cs))

	r, err := reqToAnkiConnect(ankiConnect, req)
	if err != nil {
		return []CardInfo{}, err
	}
	defer r.Body.Close()

	var res struct {
		Cards  []CardInfo `json:"result"`
		Error  string     `json:"error"`
	}

	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return []CardInfo{}, err
	}

	return res.Cards, nil
}

type phoneticSyms map[string]string

func readPhoneticSymbols(tsv string) (phoneticSyms, error) {
	f, err := os.Open(tsv)
	syms := phoneticSyms{}
	if err != nil {
		return syms, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		splited := strings.Split(scanner.Text(), "\t")
		if len(splited) != 2 {
			return syms, errors.New("invalid format error")
		}
		syms[splited[0]] = splited[1]
	}

	return syms, nil
}

func createCardPhonetics(syms phoneticSyms, c CardInfo) (string, error) {
	splited := strings.Split(c.Fields.Front.Value, " ")
	var cardPs []string
	for _, s := range splited {
		v, ok := syms[s]
		if ok {
			cardPs = append(cardPs, v)
		}
	}
	if len(cardPs) == 0 {
		return "", fmt.Errorf("the card has no phonetics data")
	}
	return strings.Join(cardPs, " "), nil
}

func updateFront(ankiConnect string, c CardInfo, newFront string) error {
	var s struct {
		Action  string `json:"action"`
		Version int    `json:"version"`
		Params struct {
			Notes struct {
				Id int `json:"id"`
				Fields struct {
					Front string `json:"Front"`
					Back  string `json:"Back"`
				} `json:"fields"`
			} `json:"note"`
		} `json:"params"`
	}
	s.Action = "updateNoteFields"
	s.Version = 6
	s.Params.Notes.Id = c.Note
	s.Params.Notes.Fields.Front = newFront
	s.Params.Notes.Fields.Back = c.Fields.Back.Value
	q, err := json.Marshal(s)
	if err != nil {
		return err
	}
	r, err := reqToAnkiConnect(ankiConnect, string(q))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return nil
}

func addPhoneticSymbols(ankiConnect string, deck string, tsv string) error {
	log.Println("Reading phonetic symbols.")
	syms, err := readPhoneticSymbols(tsv)
	if err != nil {
		return err
	}

	log.Println("Finding card-ids in the deck.")
	cardIDs, err := findCardIDs(ankiConnect, deck)
	if err != nil {
		return err
	}
	log.Println(fmt.Sprintf("%d card(s) found.", len(cardIDs)))
	log.Println("Getting cards information.")
	info, err := getCardsInfo(ankiConnect, cardIDs)
	if err != nil {
		return err
	}

	for _, c := range info {
		log.Printf("updating a front of the card (%d)", c.CardID)
		cardPs, err := createCardPhonetics(syms, c)
		if err != nil {
			log.Printf("card (%d): %s", c.CardID, err)
		}
		front := strings.Join([]string{c.Fields.Front.Value, cardPs}, "<br><br>")
		if err := updateFront(ankiConnect, c, front); err != nil {
			log.Printf("updating a front of the card (%d) has error: %s", c.CardID, err)
		}
	}

	return err
}

func addImage(ankiConnect string) error {

}

func main() {
	app := cli.NewApp()
	app.Name = "cards"

	ankiConnectFlg := cli.StringFlag{
		Name:  "a",
		Value: "http://localhost:8765/",
		Usage: "URL of Anki-Connect",
	}

	app.Commands = []cli.Command{
		{
			Name:  "decks",
			Flags: []cli.Flag{
				ankiConnectFlg,
			},
			Action: func(c *cli.Context) error {
				r, err := getDeckNames(c.String("a"))
				if err != nil {
					return err
				}
				fmt.Sprintf("%s", r)
				return nil
			},
		},

		{
			Name:  "phonetic_symbols",
			Flags: []cli.Flag{
				ankiConnectFlg,

				cli.StringFlag{
					Name:  "deck",
				},

				cli.StringFlag{
					Name:  "tsv",
				},
			},
			Action: func(c *cli.Context) error {
				err := addPhoneticSymbols(c.String("a"), c.String("deck"), c.String("tsv"))
				if err != nil {
					return err
				}
				return nil
			},
		},

		{
			Name:  "image",
			Flags: []cli.Flag{
				ankiConnectFlg,

				cli.StringFlag{
					Name:  "deck",
				},
			},
			Action: func(c *cli.Context) error {
				err := addImage(c.String("a"), c.String("deck"), c.String("tsv"))
				if err != nil {
					return err
				}
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
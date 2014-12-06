package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"bytes"

	"github.com/gorilla/schema"
)

type SlashCommand struct {
	Token        string  `schema:"token"`
	Team_ID      string  `schema:"team_id"`
	Channel_ID   string  `schema:"channel_id"`
	Channel_Name string  `schema:"channel_name"`
	User_ID      string  `schema:"user_id"`
	User_Name    string  `schema:"user_name"`
	Command      string  `schema:"command"`
	Text         string  `schema:"text,omitempty"`
	Trigger_Word string  `schema:"trigger_word,omitempty"`
	Team_Domain  string  `schema:"team_domain,omitempty"`
	Service_ID   string  `schema:"service_id,omitempty"`
	Timestamp    float64 `schema:"timestamp,omitempty"`
}

var webhooks map[string]string

func main() {
	webhooks = map[string]string {
		"rancher-bot-test": "The world can't see this dumbass!",
	}
	http.HandleFunc("/slack", CommandHandler)
	http.HandleFunc("/slack_hook", CommandHandler)
	StartServer()
}
func CommandHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err == nil {
		decoder := schema.NewDecoder()
		command := new(SlashCommand)
		err := decoder.Decode(command, r.PostForm)
		if err != nil {
			log.Println("Couldn't parse post request:", err)
		}
		if len(command.Text) == 0 {
			log.Println("no command specified")
		} else {
			c := strings.Split(command.Text, " ")
			command.Command = c[0]
			command.Text = strings.Join(c[1:], " ")
			if command.Command == "help" {
				plainResp(w, "/rancher help \"displays this help message\"\n/rancher echo msg \"prints msg\"\n/rancher image me [query] \"gets an image matching a topic\"")
			}
			if command.Command == "echo" {
				plainResp(w, command.Text)
			}
			if command.Command == "image" {
				log.Println("got an image query!")
				go getImage(command)
			}
		}
		if len(command.Trigger_Word) > 0 {
			if command.Trigger_Word == "sorry" {
				JSONResp(w, "chill on your oops, " + command.User_Name)
			}
			if command.Trigger_Word == "lol" {
				JSONResp(w, "this is a work environment, please keep it down! " + command.User_Name)
			}
		}
	}
}

func getImage(command *SlashCommand) {
	webhook := webhooks[command.Channel_Name]

	log.Println(webhook)

	payload := getGiphyImageByTag(strings.Split(command.Text, " ")[1:], command.User_Name)	

	payloadMap := make(map[string]string)
	
	payloadMap["text"] = payload	

	bodyContent, err := json.Marshal(payloadMap)
	if err != nil {
		panic(err.Error())
	}

	req, err := http.NewRequest("POST", webhook, bytes.NewBuffer(bodyContent))
	if err != nil {
		panic(err.Error())
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())	
	}
	log.Println(resp.Status)
}

func getGiphyImageByTag(tags[] string, username string) string {
	u, err := url.Parse("http://api.giphy.com/v1/gifs/random")
	if err != nil {
		panic(err.Error())
	}

	q := u.Query()
	q.Add("tag", strings.Join(tags, "+"))
	q.Add("api_key", "dc6zaTOxFJmzC")

	u.RawQuery = q.Encode()
		
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic(err.Error())
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())	
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		panic(fmt.Sprintf("Bad response from [%s], go [%d]", u.String(), resp.StatusCode))
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())	
	}
	returnData := make(map[string]interface{})
	err = json.Unmarshal(bytes, &returnData)
	if err != nil {
		panic(err.Error())
	}
	if returnDataMap,ok  := returnData["data"]; ok {
		var retUrl interface{}
		switch returnDataMap.(type) {
			case map[string]interface{}:
				retUrl = returnDataMap.(map[string]interface{})["image_url"]
			default:
				return "sorry " + username + ", no matching results for " + strings.Join(tags, " ")
		}
		if retUrl == nil {
			return "sorry " + username + ", no matching results for " + strings.Join(tags, " ")
		}	
		return username + " images matching query: " + strings.Join(tags, " ")  + "\n" + retUrl.(string)
	}
	return "sorry " + username + ", no matching results for " + strings.Join(tags, " ")
}

func JSONResp(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := map[string]string{"text": msg}
	r, err := json.Marshal(resp)
	if err != nil {
		log.Println("Couldn't marshal hook response:", err)
	} else {
		io.WriteString(w, string(r))
	}
}

func plainResp(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, msg)
}

func StartServer() {
	port := 8888
	log.Printf("Starting HTTP server on %d", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		log.Fatal("Server start error: ", err)
	}
}


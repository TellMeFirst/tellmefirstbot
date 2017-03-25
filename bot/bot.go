package bot

import (
    "bytes"
    "encoding/json"
    "log"
    "net/http"
    "mime/multipart"
    "time"
    "github.com/AntonioLangiu/tellmefirstbot/common"
    "gopkg.in/telegram-bot-api.v4"
)


func LoadBot(configuration *common.Configuration) {
	// Start the Bot
	bot, err := tgbotapi.NewBotAPI(configuration.TelegramAPI)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)

    // Start a custom http client
    var httpclient = &http.Client{
        Timeout: time.Second * 15,
    }

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			var command string = update.Message.Command()
			log.Printf("Command is %s\n\n",command)

            out := ""
            switch command {
                case "start": {
                    out = "Welcome to TellMeFirstBot!\nWith this bot you can..."
                }
                case "help": {
                    out = "Send me some text and you will receive related informations scraped from Wikipedia\n"
                }
            }
            if out != "" {
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, out)
                bot.Send(msg)
            }
		}

        // Classify the received text
        response := classifyText(update.Message.Text, httpclient)

        for _,item := range response.Resources {
            imageUri := getImageUri(item.Uri, item.Label, httpclient)
            //imageUri := ""

            output := "<b>"+item.Title+"</b>\n"
            output += "Link: "+item.Uri+"\n"
            if imageUri != "" {
                output += "<a href=\""+imageUri+"\">\xF0\x9F\x93\xB7</a>"
            }
            msg := tgbotapi.NewMessage(update.Message.Chat.ID, output)
            msg.ParseMode = "HTML"
            bot.Send(msg)
        }
	}
}

type tmfImageUri struct {
    Uri string `json:"@imageURL"`
}

func getImageUri(uri string, label string, client *http.Client) string {
    log.Printf("start getting the image")
    req, err := http.NewRequest("GET", "http://tellmefirst.polito.it:2222/rest/getImage", nil)
    if err != nil {
        log.Print(err)
        return ""
    }
    query := req.URL.Query()
    query.Add("uri", uri)
    query.Add("label", label)
    req.URL.RawQuery = query.Encode()
    req.Header.Add("Accept", "application/json")

    log.Printf("performing request");
    resp, err := client.Do(req)
    log.Printf("performed");

    if err != nil {
        log.Print(err)
        return ""
    }
    if resp.StatusCode != 200 {
        log.Printf("Status code is %d", resp.StatusCode)
        return ""
    }

    defer resp.Body.Close()
    decoder := json.NewDecoder(resp.Body)
    var r []tmfImageUri
    err = decoder.Decode(&r)
    if err != nil {
        log.Printf("Error parsing reponse %s\n", err)
        return ""
    }
    return r[0].Uri
}

type tmfResource struct {
    Uri string `json:"@uri"`
    Label string `json:"@label"`
    Title string `json:"@title"`
    Score string `json:"@score"`
    MergedTypes string `json:"@mergedTypes"`
    Image string `json:"@image"`
}

type tmfResponse struct {
    Service string `json:"@service"`
    Resources []tmfResource `json:"Resources"`
}

func classifyText (text string, client *http.Client) *tmfResponse {
    log.Printf("Classifying string: %s", text)

    // compose request
    body := new(bytes.Buffer)
    writer := multipart.NewWriter(body)
    writer.CreateFormField("classifyText")
    if writer.WriteField("text", text) != nil {
        return nil
    }
    if writer.WriteField("lang", "english") != nil {
        return nil
    }
    if writer.WriteField("numTopics", "3") != nil {
        return nil
    }
    writer.Close()

    req, err := http.NewRequest("POST", "http://tellmefirst.polito.it:2222/rest/classify", body)
    if err != nil {
        log.Printf("Error creating request\n")
        return nil
    }
    req.Header.Add("Content-Type", writer.FormDataContentType())
    req.Header.Add("Accept", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Error performing request %s\n", err)
        return nil
    }

    if resp.StatusCode != http.StatusOK {
        log.Printf("request invalid, status code: %d", resp.StatusCode)
        return nil
    }

    defer resp.Body.Close()
    // parse the response
    decoder := json.NewDecoder(resp.Body)
    var r tmfResponse
    err = decoder.Decode(&r)
    if err != nil {
        log.Printf("Error parsing reponse\n")
        return nil
    }
    log.Printf("Classify end\n");
    return &r
}

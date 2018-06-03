package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/ephraimkunz/dialogflow"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

const (
	// Required in json response
	AlexaVersion = "1.0"

	// Alexa intent types
	AlexaIntentTypeIntent       = "IntentRequest"
	AlexaIntentTypeLaunch       = "LaunchRequest"
	AlexaIntentTypeSessionEnded = "SessionEndedRequest"

	// Force a link account card to appear in the Alexa app
	AlexaCardTypeLink = "LinkAccount"

	// Story intents
	RandomStoryIntent = "random_story"

	// Alexa built-in intents we must handle
	AlexaHelpIntent   = "AMAZON.HelpIntent"
	AlexaCancelIntent = "AMAZON.CancelIntent"
	AlexaStopIntent   = "AMAZON.StopIntent"

	// SSML speech constants
	HelpText         = "<speak>Ask to hear a family story.</speak>"
	WelcomeText      = "<speak>Welcome to Family Stories! Ask to hear a family story.</speak>"
	AuthRequiredText = "<speak>This task requires linking your FamilySearch account to this skill.</speak>"

	baseUrl = "https://api-integ.familysearch.org/platform"
)

type AlexaRequest struct {
	Session AlexaSession        `json:"session,omitempty"`
	Request AlexaRequestDetails `json:"request,omitempty"`
}

type AlexaRequestDetails struct {
	Type   string      `json:"type,omitempty"`
	Intent AlexaIntent `json:"intent,omitempty"`
}

type AlexaIntent struct {
	Name  string     `json:"name,omitempty"`
	Slots AlexaSlots `json:"slots,omitempty"`
}

type AlexaSlots struct {
	Number AlexaSlot `json:"number,omitempty"`
	Lang   AlexaSlot `json:"lang,omitempty"`
}

type AlexaSlot struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type AlexaSession struct {
	User AlexaUser `json:"user,omitempty"`
}

type AlexaUser struct {
	AccessToken string `json:"accessToken,omitempty"`
}

type AlexaResponse struct {
	Version  string               `json:"version,omitempty"`
	Response AlexaResponseDetails `json:"response,omitempty"`
}

type AlexaResponseDetails struct {
	OutputSpeech     AlexaOutputSpeech `json:"outputSpeech,omitempty"`
	Card             *AlexaCard        `json:"card,omitempty"` // Pointer here to omit "card:{}" when empty struct
	ShouldEndSession bool              `json:"shouldEndSession"`
}

type AlexaCard struct {
	Type string `json:"type,omitempty"`
}

type AlexaOutputSpeech struct {
	Type string `json:"type"`
	SSML string `json:"ssml"`
}

type Person struct {
	Name    string   `json:"name,omitempty"`
	ID      string   `json:"ID,omitempty"`
	Stories []string `json:"stories,omitempty"`
}

var (
	random = rand.New(rand.NewSource(time.Now().Unix()))
)

func init() {
	http.HandleFunc("/familystories", rootHandler)
	http.HandleFunc("/familystoriesalexa", alexaHandler)
}

func ancestorRequest(token, personId string) *http.Request {
	newReq, _ := http.NewRequest("GET", baseUrl+"/tree/ancestry", nil)

	newReq.Header.Add("Accept", "application/x-fs-v1+json")
	newReq.Header.Add("Authorization", "Bearer "+token)

	q := newReq.URL.Query()
	q.Add("person", personId)
	q.Add("generations", "5")
	newReq.URL.RawQuery = q.Encode()

	return newReq
}

func storyMemoriesRequest(pID, token string) *http.Request {
	newReq, _ := http.NewRequest("GET", baseUrl+"/tree/persons/"+pID+"/memories", nil)

	newReq.Header.Add("Accept", "application/x-fs-v1+json")
	newReq.Header.Add("Authorization", "Bearer "+token)

	q := newReq.URL.Query()
	q.Add("type", "Story")
	newReq.URL.RawQuery = q.Encode()

	return newReq
}

func currentPersonRequest(token string) *http.Request {
	newReq, _ := http.NewRequest("GET", baseUrl+"/tree/current-person", nil)

	newReq.Header.Add("Accept", "application/x-fs-v1+json")
	newReq.Header.Add("Authorization", "Bearer "+token)
	return newReq
}

func getStory(ctx context.Context, token string) (*Person, string, error) {
	client := urlfetch.Client(ctx)

	// Get myself
	req := currentPersonRequest(token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}

	bytes, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	myself := getPeopleFromJSON(bytes)
	if len(myself) != 1 {
		return nil, "", errors.New("Couldn't find logged in user")
	}

	// Get the list of ancestors
	req = ancestorRequest(token, myself[0].ID)

	resp, err = client.Do(req)
	if err != nil {
		return nil, "", err
	}

	bytes, _ = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	people := getPeopleFromJSON(bytes)
	shuffled := shufflePeople(people, random)
	personStories := make(chan Person, 1)

	// Get the first memory from shuffled people that comes back
	for _, person := range shuffled {
		go func(person Person) {
			req := storyMemoriesRequest(person.ID, token)

			resp, err := client.Do(req)
			if err != nil {
				log.Debugf(ctx, "Error fetching memories: %s", err.Error())
				return
			}

			bytes, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()

			if len(bytes) > 0 {
				stories := getStoriesFromJSON(bytes)
				shuffledStories := shuffleStrings(stories, random)
				person.Stories = shuffledStories

				// Non blocking send
				select {
				case personStories <- person:
				default:
				}
			}
		}(person)
	}

	chosenPerson := <-personStories // Wait for first one

	resp, err = client.Get(chosenPerson.Stories[0])

	if err != nil {
		return nil, "", err
	}

	story, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return &chosenPerson, string(story), nil
}

func getPeopleFromJSON(bytes []byte) []Person {
	var mapJson map[string]interface{}
	json.Unmarshal(bytes, &mapJson)

	persons := mapJson["persons"].([]interface{})
	results := make([]Person, 0)

	for _, person := range persons {
		mapPerson := person.(map[string]interface{})

		name := mapPerson["display"]
		mapName := name.(map[string]interface{})

		newPerson := Person{
			ID:   mapPerson["id"].(string),
			Name: mapName["name"].(string),
		}

		results = append(results, newPerson)
	}
	return results
}

func getStoriesFromJSON(bytes []byte) []string {
	var mapJson map[string]interface{}
	json.Unmarshal(bytes, &mapJson)

	descriptions := mapJson["sourceDescriptions"].([]interface{})
	results := make([]string, 0)

	for _, desc := range descriptions {
		mapDesc := desc.(map[string]interface{})

		link := mapDesc["about"].(string)

		results = append(results, link)
	}
	return results
}

func debug(ctx context.Context, data []byte, err error) {
	if err == nil {
		log.Debugf(ctx, "Request", string(data))
	} else {
		log.Debugf(ctx, err.Error())
	}
}

func NewAlexaResponse(ssml string) AlexaResponse {
	ar := AlexaResponse{}
	ar.Version = AlexaVersion
	ar.Response.OutputSpeech.Type = "SSML"
	ar.Response.OutputSpeech.SSML = ssml
	ar.Response.ShouldEndSession = true
	return ar
}

func alexaHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	success := validateRequest(ctx, w, r)
	if !success {
		return
	}

	b, err := httputil.DumpRequest(r, true)
	debug(ctx, b, err)

	decoder := json.NewDecoder(r.Body)
	alexaReq := &AlexaRequest{}
	err = decoder.Decode(alexaReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// This is how Alexa handles the "welcome" intent
	if alexaReq.Request.Type == AlexaIntentTypeLaunch {
		resp := NewAlexaResponse(WelcomeText)
		resp.Response.ShouldEndSession = false
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonResp)
		return
	}

	// Make sure that access_token is valid if invoking an intent requiring an access token
	if alexaReq.Session.User.AccessToken == "" && alexaReq.Request.Intent.Name == RandomStoryIntent {
		resp := NewAlexaResponse(AuthRequiredText)
		card := AlexaCard{AlexaCardTypeLink}
		resp.Response.Card = &card
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonResp)
		return
	}

	var resp string

	switch alexaReq.Request.Intent.Name {
	case RandomStoryIntent:
		person, story, err := getStory(ctx, alexaReq.Session.User.AccessToken)
		if err != nil {
			resp = "Error getting the person"
		} else {
			resp = wrapStoryWithContext(person, story)
		}
	case AlexaHelpIntent:
		resp = HelpText
	case AlexaCancelIntent, AlexaStopIntent:
		resp = "" // Just stop whatever is going on
	default:
		http.Error(w, "Incorrect fullfillment action", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	str := "<speak>" + resp + "</speak>"
	str = strings.Replace(str, "&", "and", -1) // Alexa won't read ssml with '&' in it

	alexaResp := NewAlexaResponse(str)
	if alexaReq.Request.Intent.Name == AlexaHelpIntent {
		alexaResp.Response.ShouldEndSession = false // Session does not end on Launch intent or Help intent
	}

	byteResp, err := json.Marshal(alexaResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(byteResp)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	var request dialogflow.Request
	if err := json.Unmarshal(buf.Bytes(), &request); err != nil {
		log.Errorf(ctx, err.Error())
		resp := dialogflow.NewSSMLResponse(
			err.Error(),
			err.Error(),
			false,
		)
		data, _ := resp.ToJSON()
		w.Write(data)
		return
	}
	log.Warningf(ctx, "%+v", request)

	token := request.OriginalRequest.Data.User.AccessToken
	if token == "" {
		resp := dialogflow.NewSSMLResponse(
			"Sign in before trying that",
			"Sign in before trying that",
			false,
		)
		data, _ := resp.ToJSON()
		w.Write(data)
		return
	}

	person, story, err := getStory(ctx, token)
	if err != nil {
		log.Errorf(ctx, err.Error())
		resp := dialogflow.NewSSMLResponse(
			"Error getting a story",
			"Error getting a story",
			false,
		)
		data, _ := resp.ToJSON()
		w.Write(data)
		return
	}

	readableSpeech := wrapStoryWithContext(person, story)
	displaySpeech := fmt.Sprintf("Listen to the story about %s or read it here: %s", person.Name, person.Stories[0])
	resp := dialogflow.NewSSMLResponse(readableSpeech, displaySpeech, false)
	data, _ := resp.ToJSON()
	log.Infof(ctx, "Returned to client: %s", string(data))
	w.Write(data)
}

func wrapStoryWithContext(p *Person, story string) string {
	return fmt.Sprintf("Here's a story about %s: %s", p.Name, story)
}

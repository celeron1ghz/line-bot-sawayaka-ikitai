package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/line/line-bot-sdk-go/linebot"
)

type SawayakaResult struct {
	InnerDto SawayakatDto `json:"innerDto"`
}

type SawayakatDto struct {
	Stores []SawayakaStore `json:"stores"`
}

type SawayakaStore struct {
	AreaCode  string `json:"areaCode"`
	StoreName string `json:"storeName"`
	WaitCount string `json:"waitCount"`
	WaitTime  string `json:"waitTime"`
}

type LineResult struct {
	Events []LineResultEvent `json:"events"`
}

type LineResultEvent struct {
	Message    LineResultEventMessage `json:"message"`
	ReplyToken string                 `json:"replyToken"`
}

type LineResultEventMessage struct {
	Type string `json:"type"`
	Id   string `json:"id"`
	Text string `json:"text"`
}

func GetSawayakaStoreStatuses() ([]SawayakaStore, error) {
	sawayakaStores := []string{
		"KR00398061", // 函南
		"KR00299583", // 沼津学園通り
		"KR00299563", // 御殿場
		"KR00299582", // 長泉
	}

	q := url.Values{}
	q.Add("limit", "50")
	q.Add("key", "UZTa9O6QvHM1vtyLpxcqNyUlbfuT0DYJ")
	q.Add("domain", "www.genkotsu-hb.com")
	q.Add("storeId", strings.Join(sawayakaStores, ","))
	q.Add("timestamp", time.Now().String())

	uri, _ := url.Parse("https://airwait.jp/WCSP/api/external/stateless/store/getWaitInfo")
	uri.RawQuery = q.Encode()

	client := &http.Client{}
	req, _ := http.NewRequest("GET", uri.String(), nil)
	req.Header.Add("Origin", "https://www.genkotsu-hb.com")

	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	result := &SawayakaResult{}

	b, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(b, result)

	if err != nil {
		return nil, err
	}

	return result.InnerDto.Stores, nil
}

func ParseLineRequest(req events.APIGatewayProxyRequest) (LineResult, error) {
	var result LineResult
	err := json.Unmarshal([]byte(req.Body), &result)
	return result, err
}

// func handler(ctx context.Context, event linebot.Event) (interface{}, error) {
func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	lineParam, err := ParseLineRequest(req)

	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{Body: "OK", StatusCode: 200}, nil
	}

	if lineParam.Events[0].Message.Text != "ゅびぃ、さわやかいきたい" {
		// no-op
		return events.APIGatewayProxyResponse{Body: "OK", StatusCode: 200}, nil
	}

	client, err := linebot.New(os.Getenv("LINE_CHANNEL_SECRET"), os.Getenv("LINE_ACCESS_TOKEN"))

	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{Body: "OK", StatusCode: 200}, nil
	}

	stores, err := GetSawayakaStoreStatuses()

	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{Body: "OK", StatusCode: 200}, nil
	}

	messages := []string{}
	notInBusinessHourStoresCnt := 0

	for _, s := range stores {
		if s.WaitCount == "-" && s.WaitTime == "-" {
			notInBusinessHourStoresCnt++
		}
	}

	if len(stores) == notInBusinessHourStoresCnt {
		messages = append(messages, "沼津近辺のさわやかは全て営業時間外ですわ。")
	} else {
		messages = append(messages, "ルビィのために沼津近辺のさわやかの混雑状況を調べてきましたわ。", "")

		for _, s := range stores {
			name := strings.ReplaceAll(s.StoreName, "さわやか", "")

			if s.WaitCount == "-" && s.WaitTime == "-" {
				messages = append(messages, fmt.Sprintf("%sは 営業時間外", name))
			} else if s.WaitCount == "0" && s.WaitTime == "約0分" {
				messages = append(messages, fmt.Sprintf("%sは 待ち無し", name))
			} else {
				messages = append(messages, fmt.Sprintf("%sは %s組で%s待ち", name, s.WaitCount, s.WaitTime))
			}
		}

		messages = append(messages, "ですわ。")
	}

	_, err = client.ReplyMessage(lineParam.Events[0].ReplyToken, linebot.NewTextMessage(strings.Join(messages, "\n"))).Do()

	if err != nil {
		fmt.Println(err)
	}

	return events.APIGatewayProxyResponse{Body: "OK", StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}

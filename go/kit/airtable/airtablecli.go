package airtable

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/monorepo/go/kit/logger"
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var Log *logrus.Logger

type cmdPayload struct {
	Base    string              `json:"base"`
	Table   string              `json:"table"`
	Records []map[string]string `json:"records"`
}

func init() {
	Log = logger.GetLogger()
}

var RequestFailedErr = errors.New("Request Failed")

type AirtableRecord struct {
	ID          string            `json:"id,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
	CreatedTime string            `json:"createdTime,omitempty"`
}

type AirtableRecordsRes struct {
	Records []AirtableRecord `json:"records,omitempty"`
	Offset  string           `json:"offset,omitempty"`
}

type Client struct {
	apiKey      string
	baseUrl     string
	rateLimiter chan bool
	limiter     ratelimit.Limiter
}

func (a *Client) getRateToken() time.Time {
	if a.limiter == nil {
		a.limiter = ratelimit.New(10) // req / sec
	}
	return a.limiter.Take()
}

func (a *Client) prepRequest(req *http.Request) error {
	bearer := fmt.Sprintf("Bearer %v", a.apiKey)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", bearer)
	return nil
}

func (a *Client) doAuthed(method, url string, data []byte) (*http.Response, error) {
	retries := 0
	maxRetries := 20
	backoffMultiple := 10
	retryIntervalMs := 200
	maxRetryInterval := 300000
	var resp *http.Response
	var err error

	for {
		Log.Info("Making request for ", url)
		var dataReader io.Reader

		if data == nil {
			dataReader = nil
		} else {
			dataReader = bytes.NewReader(data)
		}

		req, err := http.NewRequest(method, url, dataReader)

		if err != nil {
			return resp, err
		}

		err = a.prepRequest(req)
		if err != nil {
			return resp, err
		}

		cli := &http.Client{}
		if retries > maxRetries {
			msg := fmt.Sprintf("Hit max retries when attempting %v", req.RequestURI)
			return resp, errors.New(msg)
		}

		if retryIntervalMs > maxRetryInterval {
			retryIntervalMs = maxRetryInterval
		}

		a.getRateToken()

		resp, err = cli.Do(req)

		if err != nil {
			return resp, err
		}

		Log.Infof("Airtable StatusCode %v", resp.StatusCode)
		if resp.StatusCode != 200 {
			rawRes, _ := ioutil.ReadAll(resp.Body)
			Log.Errorf("%s", rawRes)
		}

		if resp.StatusCode == 422 {
			blk := time.NewTimer(time.Duration(retryIntervalMs) * time.Millisecond)
			Log.Info("Retrying Airtable UpdateRecords in", retryIntervalMs, "ms")
			<-blk.C
			retryIntervalMs = retryIntervalMs * backoffMultiple
			continue
		}

		break
	}

	return resp, err
}

func (a *Client) UpdateRecords(base, table string, records []AirtableRecord) error {

	urlStr := strings.Join([]string{a.baseUrl, base, table}, "/")

	baseUrl, err := url.Parse(urlStr)

	if err != nil {
		return err
	}

	var recordBatches [][]AirtableRecord
	var currentBatch []AirtableRecord

	for i, rec := range records {
		if len(currentBatch) == 10 {
			recordBatches = append(recordBatches, currentBatch)
			currentBatch = []AirtableRecord{}
		}

		currentBatch = append(currentBatch, rec)

		if i == len(records)-1 {
			recordBatches = append(recordBatches, currentBatch)
			currentBatch = []AirtableRecord{}
			break
		}
	}

	for _, batch := range recordBatches {
		payload := AirtableRecordsRes{
			Records: batch,
		}

		data, err := json.Marshal(payload)

		if err != nil {
			return err
		}

		resp, err := a.doAuthed("PATCH", baseUrl.String(), data)

		if err != nil {
			return err
		}

		defer resp.Body.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Client) GetRecords(base, table, filter string) ([]AirtableRecord, error) {
	var records []AirtableRecord

	params := url.Values{}
	if filter != "" {
		params.Add("filterByFormula", filter)
	}

	urlStr := strings.Join([]string{a.baseUrl, base, table}, "/")

	baseUrl, err := url.Parse(urlStr)
	baseUrl.RawQuery = params.Encode()

	resp, err := a.doAuthed("GET", baseUrl.String(), nil)

	if err != nil {
		return records, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return records, errors.New("Invalid StatusCode GetRecords")
	}

	rawRes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return records, err
	}

	serialized := &AirtableRecordsRes{}

	err = json.Unmarshal(rawRes, serialized)

	if err != nil {
		return records, err
	}

	records = serialized.Records
	return records, nil
}

func (a *Client) CreateRecords(base, table string, records []map[string]string) error {

	var airRecs []AirtableRecord
	for _, record := range records {
		airRec := AirtableRecord{
			Fields: record,
		}
		airRecs = append(airRecs, airRec)
	}

	//TODO: Check to see if a record exists, if it does then move on
	data := &AirtableRecordsRes{
		Records: airRecs,
	}

	dataBytes, err := json.Marshal(data)

	if err != nil {
		return err
	}

	url := strings.Join([]string{a.baseUrl, base, table}, "/")

	Log.Infof("Sending to airtable %v", url)

	resp, err := a.doAuthed("POST", url, dataBytes)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	return nil
}

func NewClient(apiKey, baseUrl string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseUrl: baseUrl,
	}
}

func NewClientFromEnv() (*Client, error) {
	key := os.Getenv("AIRTABLE_API_KEY")
	if key == "" {
		return &Client{}, errors.New("Empty AIRTABLE_API_KEY")
	}

	baseUrl := "https://api.airtable.com/v0"
	return NewClient(key, baseUrl), nil
}

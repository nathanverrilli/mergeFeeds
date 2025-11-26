package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	misc "github.com/nathanverrilli/nlvMisc"
)

const SLOWDOWNSECONDS = 1
const HTTPTRYCOUNT = 3

var headers = map[string]string{
	"Content-Type": "application/json",
	"Accept":       "application/json",
}

var hc *http.Client
var httpMutex sync.Mutex
var rxNextLinkExtractor *regexp.Regexp

func init() {
	hc = newHttpClient()
	rxNextLinkExtractor = regexp.MustCompile("<([^>]*)>")
}

func newHttpClient() (hc *http.Client) {
	var tr *http.Transport
	if FlagDestInsecure {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		tr = &http.Transport{}
	}
	return &http.Client{Transport: tr, Timeout: 350 * time.Second}
}

// requestJsonObject sends an HTTP GET request to the provided URL with authorization and
// retrieves the response as JSON. requestJsonObject utilizes backoff and retries on failures,
// with headers defined globally. Returns the response body as a byte slice, the next link if present,
// an X-Total-Count header value, and any error encountered.
// note that the http client is thread-safe, so this function is safe to call concurrently.
// The mutex causes the HTTP requests to single-thread for debugging; not for use otherwise
func requestJsonObject(requestUrl string, authorization string) (body []byte, next string, xCount int, err error) {
	var backoffDelay int64 = 0
	var httpAttempt = 0
	var httpErr error = nil
	var resp *http.Response = nil
	var ctx context.Context
	var cancelFunc context.CancelFunc = nil

	// explicitly zero-value return vars
	body = nil
	next = ""
	xCount = 0

	if FlagDebugger {
		httpMutex.Lock()
		defer httpMutex.Unlock()
	} else if FlagSlow {
		time.Sleep(time.Second * time.Duration(SLOWDOWNSECONDS))
	}

	for nil == resp || resp.StatusCode != http.StatusOK {
		if httpAttempt >= HTTPTRYCOUNT {
			err = errors.New("HTTP request failed after " + strconv.Itoa(httpAttempt) + " attempts")
			if nil != cancelFunc {
				cancelFunc()
			}
			return body, next, xCount, err
		}
		httpAttempt++

		if backoffDelay > 0 {
			xLog.Printf("error recovery: backing off http request for %d milliseconds", backoffDelay)
			time.Sleep(time.Duration(backoffDelay) * time.Millisecond)
		}

		ctx, cancelFunc = context.WithTimeout(context.Background(), 2*time.Minute)
		// defer cancelFunc() <-- not here, loop error exit or loop exit

		hReq, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, bytes.NewBuffer([]byte("")))
		if nil != err {
			cancelFunc()
			xLog.Printf("Error creating HTTP request: %s", err.Error())
			return body, next, xCount, err
		}

		for key, val := range headers {
			hReq.Header.Set(key, val)
		}
		hReq.Header.Set("Authorization", authorization)

		resp, httpErr = hc.Do(hReq)
		if nil != httpErr || resp.StatusCode != http.StatusOK {
			cancelFunc()
			if nil != httpErr {
				xLog.Printf("Error performing HTTP request on [%s] because: %s", requestUrl, httpErr.Error())
			} else {
				xLog.Printf("HTTP request [%s] failed with status code %d", requestUrl, resp.StatusCode)
				if resp.StatusCode >= 400 {
					return body, next, resp.StatusCode,
						fmt.Errorf("HTTP request [%s] failed with status code %d",
							requestUrl, resp.StatusCode)
				}
			}
			backoffDelay += int64(250 * httpAttempt)
			continue
		}
	}
	defer cancelFunc()

	body, err = io.ReadAll(resp.Body)
	if nil != err {
		xLog.Printf("Error reading HTTP response body: %s", err.Error())
		return body, next, xCount, err
	}
	defer misc.DeferError(resp.Body.Close)

	xCountHeader := resp.Header.Get("X-Total-Count")
	if misc.IsStringSet(&xCountHeader) {
		xCount, err = strconv.Atoi(xCountHeader)
		if nil != err {
			xLog.Printf("Error parsing X-Total-Count header: %s", err.Error())
		}
	}

	nextHeader := resp.Header.Get("Link")
	if misc.IsStringSet(&nextHeader) {
		// regexp to extract URL link
		next = rxNextLinkExtractor.FindStringSubmatch(nextHeader)[1]
		if FlagDebug || FlagVerbose {
			xLog.Printf("next header: %s", next)
		}
	}

	return body, next, xCount, nil
}

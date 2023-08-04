package client

import (
    "context"
    "encoding/json"
    "github.com/stretchr/testify/require"
    "io"
    "net/http"
    "net/http/httptest"
    "net/url"
    "os"
    "reflect"
    "testing"
)

type Expectation struct {
    HttpRequest struct {
        Method                string `json:"method"`
        Path                  string `json:"path"`
        QueryStringParameters struct {
            CartId string `json:"cartId"`
        } `json:"queryStringParameters"`
        Cookies struct {
            Session string `json:"session"`
        } `json:"cookies"`
    } `json:"httpRequest"`
    HttpResponse struct {
        Body string `json:"body"`
    } `json:"httpResponse"`
}

func TestClient(t *testing.T) {

    rawExpectation, err := os.ReadFile("testdata/expectation.json")
    require.NoError(t, err)

    createExpectationMethod := reflect.ValueOf((*Client).CreateExpectation)

    expectationId := "055CA455-1DF7-45BB-8535-4F83E7266092"

    verifyRequestBody := VerifyRequestBody{
        ExpectationId: ExpectationId{
            Id: expectationId,
        },
    }

    verifyRequestMethod := reflect.ValueOf((*Client).VerifyRequest)

    tests := []struct {
        name          string
        clientMethod  reflect.Value
        reqBody       any
        reqValidator  func([]byte)
        respCode      int
        expectedError error
    }{
        {
            "create expectation succeed",
            createExpectationMethod,
            rawExpectation,
            func(body []byte) {
                var expectation Expectation
                err = json.Unmarshal(body, &expectation)
                require.NoError(t, err)
                require.Equal(t, "GET", expectation.HttpRequest.Method)
                require.Equal(t, "055CA455-1DF7-45BB-8535-4F83E7266092", expectation.HttpRequest.QueryStringParameters.CartId)
            },
            http.StatusCreated,
            nil,
        },
        {
            "create expectation fails with 400",
            createExpectationMethod,
            rawExpectation,
            nil,
            http.StatusBadRequest,
            &IncorrectRequestFormat{},
        },
        {
            "create expectation fails with 406",
            createExpectationMethod,
            rawExpectation,
            nil,
            http.StatusNotAcceptable,
            &InvalidExpectation{},
        },
        {
            "verify request succeed",
            verifyRequestMethod,
            verifyRequestBody,
            func(body []byte) {
                var verify VerifyRequestBody
                err = json.Unmarshal(body, &verify)
                require.NoError(t, err)
                require.Equal(t, expectationId, verify.ExpectationId.Id)
            },
            http.StatusAccepted,
            nil,
        },
        {
            "verify request failed with 400",
            verifyRequestMethod,
            verifyRequestBody,
            nil,
            http.StatusBadRequest,
            &IncorrectRequestFormat{},
        },
        {
            "verify request failed with 406",
            verifyRequestMethod,
            verifyRequestBody,
            nil,
            http.StatusNotAcceptable,
            &RequestHasNotBeenReceived{},
        },
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            ts := httptest.NewServer(
                http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    if test.reqValidator != nil {
                        body, err := io.ReadAll(r.Body)
                        require.NoError(t, err)
                        test.reqValidator(body)
                    }
                    w.WriteHeader(test.respCode)
                }),
            )
            defer ts.Close()

            parsedUrl, err := url.Parse(ts.URL)
            require.NoError(t, err)

            client := NewClient(parsedUrl, http.DefaultClient)
            callResult := test.clientMethod.Call([]reflect.Value{
                reflect.ValueOf(client),
                reflect.ValueOf(context.Background()),
                reflect.ValueOf(test.reqBody),
            })

            var clientMethodErr any
            if callResult != nil && len(callResult) > 0 {
                clientMethodErr = callResult[0].Interface()
            }

            if test.expectedError != nil {
                err, ok := clientMethodErr.(error)
                require.True(t, ok, "%v cannot be cast into error", clientMethodErr)
                require.ErrorIs(t, test.expectedError, err)
            } else {
                require.Nil(t, clientMethodErr)
            }
        })
    }
}

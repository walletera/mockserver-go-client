package main

import (
    "context"
    "fmt"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
    "github.com/walletera/mockserver-go-client/pkg/client"
    "io"
    "net/http"
    "net/url"
    "os"
)

const mockserverPort = "1090"

var expectation = []byte(`
{
  "id": "successfully get cart",
  "httpRequest": {
    "method": "GET",
    "path": "/view/cart"
  },
  "httpResponse": {
    "body": "some_response_body"
  }
}`)

func main() {
    ctx := context.Background()
    stopMockserver := runMockserver(ctx)
    defer stopMockserver(ctx)

    url, err := url.Parse(fmt.Sprintf("http://localhost:%s", mockserverPort))
    if err != nil {
        fmt.Println(err.Error())
        return
    }

    httpClient := http.DefaultClient
    mockServerClient := client.NewClient(url, httpClient)

    err = mockServerClient.CreateExpectation(ctx, expectation)
    if err != nil {
        fmt.Printf("failed creating expectation: %s\n", err.Error())
        return
    }

    resp, err := httpClient.Get(fmt.Sprintf("http://localhost:%s/view/cart", mockserverPort))
    if err != nil {
        fmt.Printf("get /view/cart request failed: %s", err.Error())
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Println(err.Error())
    }
    fmt.Printf("GET /view/cart response: code %d - body %s\n", resp.StatusCode, body)

    err = mockServerClient.VerifyRequest(ctx, client.VerifyRequestBody{
        ExpectationId: client.ExpectationId{
            Id: "successfully get cart",
        },
    })
    if err != nil {
        fmt.Printf("request verification failed: %s\n", err.Error())
        return
    }

    err = mockServerClient.Clear(ctx)
    if err != nil {
        fmt.Printf("clear request failed: %s\n", err.Error())
        return
    }
}

func runMockserver(ctx context.Context) func(ctx context.Context) {
    os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
    req := testcontainers.ContainerRequest{
        Image: "mockserver/mockserver",
        Name:  "mockserver",
        Env: map[string]string{
            "MOCKSERVER_SERVER_PORT": mockserverPort,
        },
        ExposedPorts: []string{fmt.Sprintf("%s:%s", mockserverPort, mockserverPort)},
        WaitingFor:   wait.ForHTTP("/mockserver/status").WithMethod(http.MethodPut).WithPort(mockserverPort),
    }
    mockserverC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        fmt.Printf("failed starting mockserver container: %s\n", err.Error())
        os.Exit(1)
    }

    return func(ctx context.Context) {
        if err := mockserverC.Terminate(ctx); err != nil {
            fmt.Printf("failed to terminate container: %s\n", err.Error())
        }
    }
}

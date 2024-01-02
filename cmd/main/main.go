package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	servicePort string = `:8080`
	proxyPort   string = `:8081`
)

func getHttpClient() *http.Client {
	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			// Modify the time to wait for a connection to establish
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := http.Client{
		Transport: &transport,
		Timeout:   3 * time.Second,
	}

	return &client
}

func getTargetUrl(url string) string {
	urlParts := strings.Split(url, `/`)

	// return urlParts[1] + servicePort + `/` + strings.Join(urlParts[2:], `/`)

	return fmt.Sprintf("%s://%s%s", "http", urlParts[1]+servicePort+"/", strings.Join(urlParts[2:], `/`))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Create a new HTTP request with the same method, URL, and body as the original request
	targetURL := r.URL.String()
	serviceEndpoint := getTargetUrl(targetURL)
	proxyReq, err := http.NewRequest(r.Method, serviceEndpoint, r.Body)
	if err != nil {
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	// Copy the headers from the original request to the proxy request
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Send the proxy request using the custom transport
	resp, err := getHttpClient().Transport.RoundTrip(proxyReq)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error sending proxy request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the headers from the proxy response to the original response
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	w.WriteHeader(resp.StatusCode)

	// Copy the body of the proxy response to the original response
	io.Copy(w, resp.Body)
}

func main() {
	router := mux.NewRouter()

	router.PathPrefix("/").Handler(http.HandlerFunc(handleRequest))

	// Create a new HTTP server with the handleRequest function as the handler
	server := http.Server{
		Addr:    proxyPort,
		Handler: router,
	}

	// Start the server and log any errors
	log.Println("Starting proxy server on " + proxyPort)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Error starting proxy server: ", err)
	}
}

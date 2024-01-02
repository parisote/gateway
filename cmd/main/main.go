package main

import (
	"github.com/go-chi/chi/v5"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func main() {
	r := chi.NewRouter()

	r.HandleFunc("/*", func(res http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		qs := req.URL.Query()
		body := req.Body
		log.Printf("Incoming request: %s", req.URL.String())
		targetURL, isPublic := lookupTargetURL(path, qs, body)
		if targetURL == "" {
			http.Error(res, "Not Found", 404)
			return
		}
		if !isPublic {
			//if (!myAuth.authenticate(req.Header.Get("Authorization"))) {
			http.Error(res, "Unauthorized", 401)
			return
			//}
		}
		client := &http.Client{Timeout: 10 * time.Second}
		proxy(client, targetURL, res, req)
	})

	log.Println("El gateway está corriendo en el puerto 8080")
	http.ListenAndServe(":8080", r)
}

func lookupTargetURL(path string, qs url.Values, body io.ReadCloser) (string, bool) {
	log.Println("WEAS")
	log.Println(path)
	log.Println(qs)

	bodyBytes, err := ioutil.ReadAll(body)
	defer body.Close()
	if err != nil {
		return "", false
	}

	bodyString := string(bodyBytes)
	log.Println(bodyString)

	return "http://localhost:3000/ping", true
}

func proxy(client *http.Client, targetURL string, w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid URL", 500)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Director = func(req *http.Request) {
		for key, value := range req.Header {
			req.Header[key] = value
		}
		req.Header.Add("my-header", "value")

		req.URL.Scheme = url.Scheme
		req.URL.Host = url.Host
		req.URL.Path = url.Path
	}
	proxy.Transport = &http.Transport{
		DialContext:           (&net.Dialer{Timeout: client.Timeout}).DialContext,
		ResponseHeaderTimeout: client.Timeout,
		ExpectContinueTimeout: client.Timeout,
	}
	proxy.ServeHTTP(w, r)
}

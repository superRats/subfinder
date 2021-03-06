//
// Written By : @ice3man (Nizamul Rana)
//
// Distributed Under MIT License
// Copyrights (C) 2018 Ice3man
//

// Package netcraft is a Netcraft Scraping Engine in Golang
package netcraft

import (
	"crypto/sha1" // Required for netcraft challenge response
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/subfinder/subfinder/libsubfinder/helper"
)

// Contains all subdomains found
var globalSubdomains []string

// Global Holder for Netcraft cookies
var gCookies []*http.Cookie

// Local function to recursively enumerate subdomains until no subdomains
// are left
func enumerate(state *helper.State, baseURL string) (err error) {

	// Make a http request to Netcraft
	resp, gCookies, err := helper.GetHTTPCookieResponse(baseURL, gCookies, state.Timeout)
	if err != nil {
		if !state.Silent {
			fmt.Printf("\nnetcraft: %v\n", err)
		}
		return
	}

	// Check all cookies for netcraft_js_verification_challenge
	for i := 0; i < len(gCookies); i++ {
		var curCookie = gCookies[i]
		if curCookie.Name == "netcraft_js_verification_challenge" {
			// Get the current challenge string
			challenge := url.QueryEscape(curCookie.Value)

			// Create a sha1 hash as response
			h := sha1.New()
			io.WriteString(h, challenge)
			response := fmt.Sprintf("%x", h.Sum(nil))

			respCookie := &http.Cookie{
				Name:   "netcraft_js_verification_response",
				Value:  response,
				Domain: ".netcraft.com",
			}

			gCookies = append(gCookies, respCookie)
		}
	}

	// Get the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if !state.Silent {
			fmt.Printf("\nnetcraft: %v\n", err)
		}
		return
	}

	src := string(body)

	re := regexp.MustCompile("<a href=\"http://toolbar.netcraft.com/site_report\\?url=(.*)\">")
	match := re.FindAllStringSubmatch(src, -1)

	for _, subdomain := range match {
		// Dirty Logic
		finishedSub := strings.Split(subdomain[1], "//")[1]

		if state.Verbose == true {
			if state.Color == true {
				fmt.Printf("\n[%sNETCRAFT%s] %s", helper.Red, helper.Reset, finishedSub)
			} else {
				fmt.Printf("\n[NETCRAFT] %s", finishedSub)
			}
		}

		globalSubdomains = append(globalSubdomains, finishedSub)
	}

	// we have another page full of juicy subdomains
	if strings.Contains(src, "Next page") {
		// Checkout the link for the next page
		reNext := regexp.MustCompile("<A href=\"(.*?)\"><b>Next page</b></a>")
		match := reNext.FindStringSubmatch(src)

		// Replace spaces with + characters in URL Query since they don't allow request to happen
		finalQuery := strings.Replace(match[1], " ", "+", -1)
		enumerate(state, "https://searchdns.netcraft.com"+finalQuery)
	}

	// Finally, all subdomains found :-)
	return nil
}

// Query function returns all subdomains found using the service.
func Query(args ...interface{}) interface{} {

	domain := args[0].(string)
	state := args[1].(*helper.State)

	// Initialize global cookie holder
	gCookies = nil

	// Query using first page. Everything from there would be recursive
	err := enumerate(state, "https://searchdns.netcraft.com/?restriction=site+ends+with&host="+domain+"&lookup=wait..&position=limited")
	if err != nil {
		if !state.Silent {
			fmt.Printf("\nerror: %v\n", err)
		}
		return globalSubdomains
	}

	return globalSubdomains
}

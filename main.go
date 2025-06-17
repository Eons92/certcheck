package main

import (
	"fmt"
	//"strconv"
	s "strings"

	"github.com/bitfield/script"

	//"github.com/imroc/req/v3"
	"crypto/tls"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/hackebrot/turtle"
	"github.com/hako/durafmt"
	"github.com/jedib0t/go-pretty/v6/table"
)

var certState = ""

func main() {
	urlList, err := script.File("url1.csv").Slice()
	if err != nil {
		panic(err)
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Validity", "URL", "Remaining days", "Expiration Date"})
	/*
			t.AppendRows([]table.Row{
		        {1, "Arya", "Stark", 3000},
		        {20, "Jon", "Snow", 2000, "You know nothing, Jon Snow!"},
		    })
		    t.AppendSeparator()
		    t.AppendRow([]interface{}{300, "Tyrion", "Lannister", 5000})
		    t.AppendFooter(table.Row{"", "", "Total", 10000})
		    t.Render()
	*/
	for i := 0; i < len(urlList); i++ {
		hasExpired, daysLeft, expirationDate, validURL := SslExpiry(s.Join([]string{"https://", urlList[i]}, ""))

		if hasExpired == true {
			daysLeft = s.Join([]string{turtle.Emojis["red_circle"].Char, daysLeft}, " ")
		}

		t.AppendRow(table.Row{turtle.Emojis[statusEmoji(validURL)].Char, urlList[i], daysLeft, expirationDate.Local()})
	}
	t.SetStyle(table.StyleColoredBright)
	t.Render()
	//fmt.Println(turtle.Emojis["white_check_mark"].Char)

}

// NetworkConnector is an interface to abstract network connections.
type NetworkConnector interface {
	Dial(network, address string) (net.Conn, error)
}

// RealNetworkConnector implements NetworkConnector using the real net.Dialer.
type RealNetworkConnector struct{}

func (rnc RealNetworkConnector) Dial(network, address string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	return tls.DialWithDialer(dialer, network, address, tlsConfig)
}

func SslExpiry(targetURL string) (isExpired bool, remainingDays string, expirationDate time.Time, validURL bool) {
	validURL = true
	formattedURL := FormatURL(targetURL)
	conn, err := DialNetwork(formattedURL, RealNetworkConnector{})
	if err != nil {
		fmt.Println("Error:", err)
		validURL = false
		return
	}
	defer conn.Close()

	expiryDate, err := GetCertificateExpiryDate(conn)
	expirationDate = expiryDate
	if err != nil {
		fmt.Println("Error:", err)
		validURL = false
		return
	}

	if expiryDate.Before(time.Now()) {
		isExpired = true
	}

	remainingDays = CalculateRemainingDays(expiryDate)

	//fmt.Printf("Certificate Expiry Date: %s\n", expiryDate)

	return
}

// FormatURL formats the given URL to the desired format.
// If the URL starts with "https://", this function removes the prefix,
// trims any trailing "/" character, and appends ":443" to indicate the
// default HTTPS port. The formatted URL is returned as the result.
// for example https://example.com/ is going to return example.com:443
func FormatURL(targetURL string) string {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal(parsedURL)
	}
	if parsedURL.Scheme == "https" {
		targetURL = parsedURL.Host + ":443"
	}
	return targetURL
}

// DialNetwork uses the provided NetworkConnector to establish a network connection.
func DialNetwork(targetURL string, connector NetworkConnector) (net.Conn, error) {
	return connector.Dial("tcp", targetURL)
}

// GetCertificateExpiryDate retrieves the expiry date of the peer certificate.
func GetCertificateExpiryDate(conn net.Conn) (time.Time, error) {
	certChain := conn.(*tls.Conn).ConnectionState().PeerCertificates

	if len(certChain) == 0 {
		return time.Time{}, fmt.Errorf("no certificate found")
	}

	return certChain[0].NotAfter, nil
}

// CalculateRemainingDays calculates the remaining days until the given date.
func CalculateRemainingDays(expiryDate time.Time) string {
	duration := durafmt.Parse(expiryDate.Sub(time.Now())).LimitFirstN(2)
	return string(duration.String())
}

func statusEmoji(status bool) string {
	if status {
		return "white_check_mark"
	}
	return "red_circle"
}

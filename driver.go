package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ChrisTrenkamp/goxpath"
	"github.com/ChrisTrenkamp/goxpath/tree"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree"
	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/go-ini/ini"
)

type ZoneRecord struct {
	Host string
	Type string
	Data string
	TTL  string
}

type Credentials struct {
	Username string
	Password string
}

func getCurrentExternalIP() string {
	res, err := http.Get("https://api.ipify.org")

	if err != nil {
		return "error"
	}

	defer res.Body.Close()
	contents, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "boom"
	}

	return string(contents)
}

func getZonesFromLiveDns(username string, password string, zone string) string {

	u, _ := url.Parse("https://domains.livedns.co.il/API/DomainsAPI.asmx/GetZoneRecords")
	q := u.Query()
	q.Set("Username", username)
	q.Set("Password", password)
	q.Set("DomainName", zone)
	u.RawQuery = q.Encode()
	res, err := http.Get(u.String())

	if err != nil {
		return ""
	}

	defer res.Body.Close()
	contents, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return ""
	}

	return string(contents)
}

func getZones(cred *Credentials) []ZoneRecord {

	xml := getZonesFromLiveDns(cred.Username, cred.Password, "krapivin.co.il")
	fmt.Printf(xml)
	xp := goxpath.MustParse(`//LiveDnsResult/ZoneRecord`)
	t, _ := xmltree.ParseXML(bytes.NewBufferString(xml))
	zones, _ := xp.ExecNode(t)

	zonesArray := make([]ZoneRecord, 0)
	for _, zone := range zones {
		host, _ := goxpath.MustParse(`/Host`).Exec(zone)
		recType, _ := goxpath.MustParse(`/Type`).Exec(zone)
		data, _ := goxpath.MustParse(`/Data`).Exec(zone)
		ttl, _ := goxpath.MustParse(`/TTL`).Exec(zone)

		zonesArray = append(zonesArray, ZoneRecord{
			Host: host.(tree.Elem).ResValue(),
			Type: recType.(tree.Elem).ResValue(),
			Data: data.(tree.Elem).ResValue(),
			TTL:  ttl.(tree.Elem).ResValue(),
		})
	}

	res := From(zonesArray).Where(func(z interface{}) bool {
		return strings.Compare(z.(ZoneRecord).Type, "Host (A)") == 0
	}).Results()

	result := make([]ZoneRecord, len(res))
	for i := range res {
		result[i] = res[i].(ZoneRecord)
	}
	return result
}

func updateZone(zone *ZoneRecord, newIP string) bool {
	return false
}

func updateLiveDnsIP(credentials Credentials, ip string) bool {
	zones := getZones(&credentials)
	for _, zone := range zones {
		suc := updateZone(&zone, ip)
		if suc {
			fmt.Println("Zone: " + zone.Data + " updated")

		} else {
			fmt.Println("Zone: " + zone.Data + " failed to update")
		}
	}

	return true
}

func main() {
	cred := new(Credentials)

	cfg, err := ini.Load("settings.ini")
	if err != nil {
		fmt.Printf("An error occured while loading settings.ini. loading credentials from env. var.")
		cred.Password = os.Getenv("live_pass")
		cred.Username = os.Getenv("live_user")
	}

	credSection, _ := cfg.GetSection("credentials") // return error too
	cred.Username = credSection.Key("username").String()
	cred.Password = credSection.Key("password").String()

	ticker := time.NewTicker(time.Second)

	ip := "127.0.0.1" // keep it in a file and read the value on start

	for t := range ticker.C {
		curIp := getCurrentExternalIP()
		if ip != curIp {
			fmt.Printf("%s > Differents IPs, \n\tCur.: %s, \n\tPrev.: %s\n", t.UTC().Format(time.RFC3339), curIp, ip)
			fmt.Printf("%s > Updaing LiveDNS via DomainAPI...\n", time.Now().UTC().Format(time.RFC3339))

			success := updateLiveDnsIP(*cred, curIp)
			if success {
				ip = curIp
				fmt.Printf("%s > Done.\n", time.Now().UTC().Format(time.RFC3339))
			} else {
				fmt.Printf("%s > ERROR ERROR ERROR.\n", time.Now().UTC().Format(time.RFC3339))
			}

		} else {
			fmt.Printf("%s > A-OK.\n", time.Now().UTC().Format(time.RFC3339))
		}
	}
}

package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"time"
	"bytes"
	"strings"
	
	"github.com/go-ini/ini"
	"github.com/ChrisTrenkamp/goxpath/goxpath"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree"

	. "github.com/ahmetalpbalkan/go-linq"
)

type ZoneRecord struct {
	Host string
	Type string
	Data string
	TTL string
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

	res, err := http.Get(fmt.Sprintf("https://domains.livedns.co.il/API/DomainsAPI.asmx/GetZoneRecords?UserName=%s&Password=%s&DomainName=%s", username, password, zone))

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

func getZones() []ZoneRecord {

	xml := getZonesFromLiveDns("", "", "")
	xp := goxpath.MustParse(`//LiveDnsResult/ZoneRecord`)
	t := xmltree.MustParseXML(bytes.NewBufferString(xml))
	zones := xmltree.Exec(xp, t, nil)

	zonesArray := make([]ZoneRecord,0)
	for _, zone := range zones {
		zonesArray = append(zonesArray, ZoneRecord{
			Host:xmltree.Exec(goxpath.MustParse(`/Host`), zone, nil)[0].String(),
			Type:xmltree.Exec(goxpath.MustParse(`/Type`), zone, nil)[0].String(),
			Data:xmltree.Exec(goxpath.MustParse(`/Data`), zone, nil)[0].String(),
			TTL:xmltree.Exec(goxpath.MustParse(`/TTL`), zone, nil)[0].String(),
		})
	}
	getARecords := func (in T) (bool, error) {
		return strings.Compare(in.(ZoneRecord).Type, "Host (A)") == 0, nil
	}

	res, _ := From(zonesArray).Where(getARecords).Results()

	result := make([]ZoneRecord,len(res))
	for i := range res {
		result[i] = res[i].(ZoneRecord)
	}
	return result
}

func updateZone(zone *ZoneRecord, newIP string) bool {
	return false
}

func updateLiveDnsIP(credentials Credentials, ip string) bool{
	zones := getZones()
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

func main()
{
	cfg, err := ini.Load("settings.ini")

	ticker := time.NewTicker(time.Second)

	ip := "127.0.0.1" // keep it in a file and read the value on start

	for t := range ticker.C {
		curIp := getCurrentExternalIP()
		if ip != curIp {
			fmt.Printf("%s > Differents IPs, \n\tCur.: %s, \n\tPrev.: %s\n", t.UTC().Format(time.RFC3339), curIp, ip)
			fmt.Printf("%s > Updaing LiveDNS via DomainAPI...\n", time.Now().UTC().Format(time.RFC3339))

			success := updateLiveDnsIP(curIp)
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

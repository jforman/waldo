// https://godoc.org/golang.org/x/oauth2/google
// https://cloud.google.com/dns/api/v1/changes/create
//
// https://github.com/kelseyhightower/dns01-exec-plugins/blob/master/googledns/client.go

package main

import (
	"flag"
	"fmt"
	"github.com/rdegges/go-ipify"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/api/dns/v1"
)

var currentExternalIP string
var credentialsPath string
var ctx context.Context
var errorCount int
var externalIP string
var managedZone string
var recordName string
var recordType string
var recordTTL int64
var project string
var waitDuration time.Duration

func init() {
	flag.DurationVar(&waitDuration, "waitDuration", 300*time.Second,
		 		"Interval in seconds to check public IP address.")
	flag.StringVar(&credentialsPath, "credentialsPath", "",
				"Path to JSON credentials file for updating DNS.")
	flag.StringVar(&recordName, "recordName", "",
				"DNS Host Resource Record to Update in Cloud DNS.")
	flag.StringVar(&managedZone, "managedZone", "",
				"Zone name in Google Cloud DNS.")
	flag.StringVar(&project, "project", "",
				"Project name within Google Cloud associated with Managed Zone.")
	flag.StringVar(&recordType, "recordType", "A",
				"RR Datatype for the DNS record.")
	flag.Int64Var(&recordTTL, "recordttl", 60,
				"TTL (minutes) for the DNS record TTL")
}

func getFormattedRecordName(recordName string) string {
	return fmt.Sprintf("%v.", recordName)
}

func getHttpClient() (*dns.Service, error) {
	authJson, err := ioutil.ReadFile(credentialsPath)
	if err != nil {
		return nil, err
	}

	conf, err := google.JWTConfigFromJSON(authJson, dns.CloudPlatformScope)

	if err != nil {
		return nil, err
	}

	ctx := cloud.NewContext(project, conf.Client(oauth2.NoContext))

	hc, err := google.DefaultClient(ctx, dns.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	c, err := dns.New(hc)
	if err != nil {
		return nil, err
	}

	return c, err
}

func getDNSRecord(dnsClient *dns.Service, recordName string, ipAddress string) (record *dns.ResourceRecordSet, doesIpMatch bool, err error) {
	records, err := dnsClient.ResourceRecordSets.List(project, managedZone).Do()
	if err != nil {
		return nil, false, err
	}

	for _, record := range records.Rrsets {
		if (record.Name == getFormattedRecordName(recordName)) &&
		 	(record.Type == recordType) {
			if (record.Rrdatas[0] == ipAddress) {
				// A fully matching record was found.
				return record, true, nil
			}
			// A record was found, but the IPs do not match.
			return record, false, nil
		}
	}
	// No matching records were found, no errors.
	return nil, false, nil
}

func deleteDNSRecord(dnsClient *dns.Service, resourceRecordSet *dns.ResourceRecordSet) error {
	fmt.Printf("Attempting to delete record %v.\n", resourceRecordSet.Name)
	change := &dns.Change{
		Deletions: []*dns.ResourceRecordSet{resourceRecordSet},
	}

	_, err := dnsClient.Changes.Create(project, managedZone, change).Do()
	if err != nil {
		return err
	}
	return nil
}

func addDNSRecord(dnsClient *dns.Service, ipAddress string) error {
	record := &dns.ResourceRecordSet{
		Name:    getFormattedRecordName(recordName),
		Rrdatas: []string{ipAddress},
		Ttl:     recordTTL,
		Type:    recordType,
	}

	change := &dns.Change{
		Additions: []*dns.ResourceRecordSet{record},
	}

	_, err := dnsClient.Changes.Create(project, managedZone, change).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Record %v updated to point to %v.\n", recordName, ipAddress)
	return nil
}

func main() {
	flag.Parse()
	var err error

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsPath)
	if err != nil {
		fmt.Printf("Error parsing application credentials: %v.\n", err)
		os.Exit(1)
	}

	dnsClient, err := getHttpClient()
	if err != nil {
		fmt.Printf("Error getting dnsClient: %v.\n", err)
		os.Exit(1)
	}

	// Create function to die when receiving SIGINT/SIGTERM.
	go func() {
		sig := <- sigs
		fmt.Printf("Received signal! %v.\n", sig)
		os.Exit(0)
	}()


	for {
		externalIP, err := ipify.GetIp()

		if err != nil {
			fmt.Printf("Unable to determine external IP address: %v. No update performed. \n", err)
			time.Sleep(waitDuration)
			continue
		}

		record, doesIpMatch, err := getDNSRecord(dnsClient, recordName, externalIP)
		if err != nil {
			fmt.Printf("Error getting DNS record: %v. Skipping evaluation.\n", err)
			time.Sleep(waitDuration)
			continue
		}

		if (record != nil) {
			// Record with matching name was found, but it does not match old entry.
			if doesIpMatch {
				time.Sleep(waitDuration)
				continue
			}
			// Record exists, but IP does not match. So we need to delete the old one.
			err := deleteDNSRecord(dnsClient, record)
			if err != nil {
				fmt.Printf("Error in deleting record: %v.\n", err)
			}
		}

		// No previous record was found. So let's add it.
		err = addDNSRecord(dnsClient, externalIP)
		if err != nil {
			fmt.Printf("Error in adding DNS record: %v.\n", err)
		}
		time.Sleep(waitDuration)
	}
}

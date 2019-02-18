package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rdegges/go-ipify"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v2beta1"
)

var currentExternalIP string
var credentialsPath string
var ctx context.Context
var externalIP string
var managedZone string
var oneShot bool
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
	flag.BoolVar(&oneShot, "oneShot", false,
		"Attempt to perform one update and then quit.")
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

func getDNSClient() (*dns.Service, error) {
	// https://github.com/golang/oauth2/blob/master/google/example_test.go
	data, err := ioutil.ReadFile(credentialsPath)
	if err != nil {
		return nil, err
	}

	conf, err := google.JWTConfigFromJSON(data, dns.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	client := conf.Client(oauth2.NoContext)
	dnsService, err := dns.New(client)
	if err != nil {
		return nil, err
	}

	return dnsService, err
}

func getDNSRecord(dnsClient *dns.Service, recordName string, cloudIpAddress string) (record *dns.ResourceRecordSet, doesIpMatch bool, err error) {
	records, err := dnsClient.ResourceRecordSets.List(project, managedZone).Do()
	if err != nil {
		return nil, false, err
	}

	for _, record := range records.Rrsets {
		if (record.Name == getFormattedRecordName(recordName)) &&
			(record.Type == recordType) {
			if record.Rrdatas[0] == cloudIpAddress {
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

	dnsClient, err := getDNSClient()
	if err != nil {
		fmt.Printf("Error getting dnsClient: %v.\n", err)
		os.Exit(1)
	}

	// Create function to die when receiving SIGINT/SIGTERM.
	go func() {
		sig := <-sigs
		fmt.Printf("Received signal! %v.\n", sig)
		os.Exit(0)
	}()

	fmt.Printf("Waldo started. waitDuration: %v, oneShot: %v, credentialsPath: %v, recordName: %v, managedZone: %v, project: %v, recordType: %v, recordTTL: %v.\n", waitDuration, oneShot, credentialsPath, recordName, managedZone, project, recordType, recordTTL)

	for {
		externalIP, err := ipify.GetIp()

		if err != nil {
			fmt.Printf("Unable to determine external IP address: %v. No update performed. \n", err)
			time.Sleep(waitDuration)
			if oneShot {
				os.Exit(1)
			}
			continue
		}

		record, doesIpMatch, err := getDNSRecord(dnsClient, recordName, externalIP)
		if err != nil {
			fmt.Printf("Error getting DNS record: %v. Skipping evaluation.\n", err)
			if oneShot {
				os.Exit(1)
			}
			time.Sleep(waitDuration)
			continue
		}

		if record != nil {
			if doesIpMatch {
				if oneShot {
					fmt.Println("IP addresses match. Nothing to do in oneShot mode.")
					os.Exit(0)
				}
				time.Sleep(waitDuration)
				continue
			}
			// Record exists, but IP does not match. So we need to delete the old one.
			fmt.Println("IP address of record does not match record.")
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
		if oneShot {
			os.Exit(0)
		}
		time.Sleep(waitDuration)
	}
}

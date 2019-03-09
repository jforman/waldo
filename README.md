## Waldo

Detect external Internet IP address changes from any machine, and update a DNS entry in a Google Cloud DNS Managed Zone with that new IP address.

## Problem Statement

I moved from having a static IPv4 public IP address on my home Internet connection,
to having a dynamic IPv4 address. How was I going to be able to figure out the
IP address to SSH into if it could possibly change?

There must be a way to detect if has changed, and update my DNS records accordingly.

## Run Locally
1. Configure [Application Default Credentials](https://developers.google.com/identity/protocols/application-default-credentials) for your Google Cloud Project.
2. Download the resulting JSON file to your local machine.
3. Determine a DNS entry that will be updated in the managed zone.
4. Run binary.
```
go run waldo.go --credentialsPath adc.json --managedZone $ZONEFROMCLOUDDNS --project $PROJECTNAME --recordName fqdn.to.keep.updated.tld
```

## Run On Kubernetes

First, you need to create a secret containing your service account credentials.

```
kubectl create secret generic waldo-creds --from-file=waldo-creds.json
```

Edit waldo-deployment.yaml to contain your zone name, record name. etc.

Then create the deployment.

```
kubectl apply -f waldo-deployment.yaml
```

## Command Line Flags

```
-credentialsPath string
  Path to JSON credentials file for updating DNS.
-managedZone string
  Zone name in Google Cloud DNS.
-project string
  Project name within Google Cloud associated with Managed Zone.
-recordName string
  DNS Host Resource Record to Update in Cloud DNS.
-recordType string
  RR Datatype for the DNS record. (default "A")
-recordttl int
  TTL (minutes) for the DNS record TTL (default 60)
-waitDuration duration
  Interval in seconds to check public IP address. (default 5m0s)
```

## Future plans when time permits.

* Dry run support for Record Adds and Deletes.
* Handle both IPv4 and IPv6.
* Notification Upon IP Change: Slack, Email, IRC, SMS

## References Used in Development

* https://godoc.org/golang.org/x/oauth2/google
* https://cloud.google.com/dns/api/v1/changes/create
* https://github.com/kelseyhightower/dns01-exec-plugins/blob/master/googledns/client.go

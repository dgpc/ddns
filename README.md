# DDNS

Dynamic DNS server for Google Cloud DNS managed zones on Cloud Run.

This is not an officially supported Google product. This project is maintained
soley by David Claridge, and not in association with his employment at Google.

## Set-up

1. Clone this GitHub repository.
1. Change the `project` and `zone` constants in `ddns/google.go`.
1. Build using `docker build -t gcr.io/<project>/ddns-server:latest .
1. Push the image to GCR.
1. Create the Cloud Run applicable.

## TODO

This project is a work in progress. It is *definitely not production-ready*.
Unfinished work includes:

* Improving the Set-up instructions.
* Implementing authentication.
* Moving project/zone configuration to environment variables.
* Supporting multiple users/zones.

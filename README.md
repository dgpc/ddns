# DDNS

Dynamic DNS server for Google Cloud DNS managed zones on Cloud Run.

This is not an officially supported Google product. This project is maintained
soley by David Claridge, and not in association with his employment at Google.

## Set-up

1. Create a GCP project and enable Datastore, Cloud DNS & Cloud Run.
1. Create a Zone in Cloud DNS.
1. Clone this GitHub repository.
1. Change the `Project` and `zone` constants in `ddns/google.go`.
1. Build using `docker build -t gcr.io/<project>/ddns-server:latest .`.
1. Push the image to GCR using `docker push gcr.io/<project>/ddns-server:latest`.
1. (optional) Create a service account with the roles "Cloud Datastore User" and "DNS Administrator". Pass that service account to the next command as `--service-account <custom-account>`.
1. Deploy the Cloud Run application `gcloud run deploy --image gcr.io/<project>/ddns-server:latest --project <project> --platform managed --allow-unauthenticated`.
1. Create tokens for your domains using: `go run ./cmd/adddomain`. The created token corresponds to the password to use in the next step.
1. Configure the dynamic DNS client on your router to use the Cloud Run app's address. Specify any value as the username, if required, and the created token as the password.

## TODO

This project is a work in progress. Unfinished work includes:

* UI for managing domains (instead of `adddomain` CLI utility).
* Moving project/zone configuration to environment variables.
* Listening based on the `PORT` environment variable, rather than a constant `8080`.

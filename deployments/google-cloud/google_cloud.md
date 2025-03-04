# Deploy to Google Cloud

## Install Google Cloud SDK

Install the Google Cloud SDK CLI following [these instructions](https://cloud.google.com/sdk/docs/install) or with 
[Homebrew](https://formulae.brew.sh/cask/google-cloud-sdk).

## Google Cloud Project Setup

Log in to Google Cloud.

```bash
gcloud auth login
```

Create a `.env_google_cloud.sh` file to store your google cloud environment variables.

```bash
export GCP_PROJECT_NAME=<gcp project name>
export GCP_PROJECT_ID=<gcp project id>
export GCP_PROJECT_REGION=<gcp region>
export GCP_BILLING_ACCOUNT=<billing account id>
```

A project folder may also be needed.

```bash
export GCP_PROJECT_FOLDER=<gcp project folder>
```

Source the `.env_google_cloud.sh` file.

```bash
source .env_google_cloud.sh
```

Create a new GCP project.

```bash
gcloud projects create ${GCP_PROJECT_ID} \
  --name ${GCP_PROJECT_NAME} \
  --folder=${GCP_PROJECT_FOLDER} \
  --quiet
```

View the newly created project.

```bash
gcloud projects describe ${GCP_PROJECT_ID}
```

Configure the Google Cloud CLI to use your new project.

```bash
gcloud config set project ${GCP_PROJECT_ID}
```

Ensure billing is enabled.

```bash
gcloud services enable cloudbilling.googleapis.com
gcloud alpha billing projects link ${GCP_PROJECT_ID} --billing-account ${GCP_BILLING_ACCOUNT}
```

Enable other supporting APIs.

```bash
gcloud services enable cloudbuild.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable compute.googleapis.com
gcloud services enable vpcaccess.googleapis.com
```

## Build via Cloud Build

Build Image via Cloud Build

```bash
gcloud builds submit --pack image=gcr.io/${GCP_PROJECT_ID}/${GCP_PROJECT_NAME}:tag1,builder=heroku/buildpacks:20
```

## Deploy via Cloud Run

Deploy the demo application.

 ```bash
gcloud run deploy ${GCP_PROJECT_NAME}-demo --command="demo" --allow-unauthenticated --region=${GCP_PROJECT_REGION} --image=gcr.io/${GCP_PROJECT_ID}/${GCP_PROJECT_NAME}:tag1
 ```

Build an OPA server with configuration via Docker.

From the `./deployments/opa-server` directory run the below commands.

```bash
docker pull openpolicyagent/opa:latest
docker build -t ${GCP_PROJECT_NAME}-opa-server:latest .
```

```bash
docker tag ${GCP_PROJECT_NAME}-opa-server:latest gcr.io/${GCP_PROJECT_ID}/hexa-opa-server:latest
docker push gcr.io/${GCP_PROJECT_ID}/hexa-opa-server:latest
```

Deploy via Cloud Run.

```bash
gcloud beta run deploy ${GCP_PROJECT_NAME}-opa-server --allow-unauthenticated \
  --region=${GCP_PROJECT_REGION} --image=gcr.io/${GCP_PROJECT_ID}/hexa-opa-server:latest \
  --port=8887 --args='--server,--addr,0.0.0.0:8887,--config-file,/config.yaml'
```

Edit and deploy a new revision of the demo application with the `OPA_SERVER_URL` environment variable.

```bash
gcloud run services update ${GCP_PROJECT_NAME}-demo --region=${GCP_PROJECT_REGION} \
  --update-env-vars OPA_SERVER_URL='https://<opa-server-url>/v1/data/authz/allow'
```

Edit and deploy a new revision of the opa-server application with the `HEXA_DEMO_URL` environment variable.

```bash
gcloud run services update ${GCP_PROJECT_NAME}-demo --region=${GCP_PROJECT_REGION} \
  --update-env-vars HEXA_DEMO_URL='https://<hexa-demo-url>'
```

## Deploy via Kubernetes

Deploy the Demo app and the OPA Agent to Kubernetes.

Make sure you have sourced your environment file and set the project.

Enable the Kubernetes Engine API.

```bash
gcloud services enable container.googleapis.com
```

Create a GKE cluster.

```bash
gcloud beta container --project "${GCP_PROJECT_ID}" \
  clusters create-auto hexa-demo \
  --region ${GCP_PROJECT_REGION} \
  --release-channel "regular" \
  --network "projects/${GCP_PROJECT_ID}/global/networks/default" \
  --subnetwork "projects/${GCP_PROJECT_ID}/regions/${GCP_PROJECT_REGION}/subnetworks/default" \
  --cluster-ipv4-cidr "/17" \
  --services-ipv4-cidr "/22"
```

Configure kubectl for newly created cluster.

```bash
gcloud container clusters get-credentials hexa-demo --region ${GCP_PROJECT_REGION} --project ${GCP_PROJECT_ID}
```

Create IP addresses for the Demo app.

```bash
gcloud compute addresses create hexa-demo-app-static-ip --global --ip-version IPV4
```

Deploy demo app objects.

```bash
envsubst < kubernetes/demo/config.yaml | kubectl apply -f -
envsubst < kubernetes/demo/managed-certificate.yaml | kubectl apply -f -
envsubst < kubernetes/demo/deployment.yaml | kubectl apply -f -
envsubst < kubernetes/demo/service.yaml | kubectl apply -f - 
envsubst < kubernetes/demo/ingress.yaml | kubectl apply -f -
```

Deploy OPA Agent objects.

```bash
envsubst < kubernetes/opa-server/deployment.yaml | kubectl apply -f -
envsubst < kubernetes/opa-server/service.yaml | kubectl apply -f - 
```

For orchestrating policy, you'll need to set up Google's Identity Aware Proxy. For Cloud Run, you'll need a load balancer
and backend resource. See this [gcloud reference](https://cloud.google.com/load-balancing/docs/https/setup-global-ext-https-serverless#gcloud_1) 
for more information.

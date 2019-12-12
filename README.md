### Google sheet to Json

http://blog.pamelafox.org/2013/06/exporting-google-spreadsheet-as-json.html

(but use export_to_json.js as the script)

### Switch to project

```
gcloud config set project youtube-tool-260519
```

### Reconnect

```
ssh 167.71.128.132
screen -r
```

### Disconnect

```
Ctrl+a then d
```

### Setup

```
# Add the Cloud SDK distribution URI as a package source
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list

# Import the Google Cloud Platform public key
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -

# Update the package list and install the Cloud SDK
sudo apt-get update && sudo apt-get install google-cloud-sdk

# Initialise
gcloud init

# Install Go
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get update
sudo apt-get install golang-go

# Copy keys
mkdir ~/.credentials
pico ~/.credentials/drive_secret.json
pico ~/.credentials/youtube_secret.json

# Initialise project
cd ~/go/src/youtube
go get
go run *.go

```




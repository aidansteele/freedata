# `freedata`

This is a silly little project that gives you free egress (I think? Someone fact-check
that, please) from your webapp-hosting EC2 instances via AWS Systems Manager
Session Manager and Tailscale. Session Manager streams are limited to about
10mbit.

## Usage

Why are you using this? Don't.

```
# run this somewhere with cheap egress, e.g. outside AWS
go build
./freedata i-instanceid 8080 # if your web app is listening on port 8080
```

## How it works

Here's a really half-baked sequence diagram. 

```mermaid
sequenceDiagram
participant B as Browser
participant T as Tailscale
participant O as Outside machine
participant S as Session Manager
participant E as EC2 Instance
E->>S: Register instance at launch
O->>T: Launch Tailscale Funnel
B->>T: Browse to public Tailscale funnel domain
T->>O: Receive Accept()'d net.Conn from browser
O->>S: Dial port 8080 on EC2 Instance
S->>E: Send port-forwarded packets
```

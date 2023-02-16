# RTP to WebRTC (broadcast)

Small utility to broadcast an RTP stream to multiple peers with WebRTC

# Usage

Download the latest release and run the binary or build from source using `go build`.

## Arguemnts

* `-i <url>`: Set URL as the source RTP stream to `<url>`
* `-o <url>`: Set URL for signaling to `ws://<url>/signal`
* `-tid <trackID>`: Set the track ID to `<trackID>`
* `-sid <streamID>`: Set the stream ID to `<streamID>`
* `-mtu <mtu>`: Manually set the MTU of the interface, defaults to 1500
* `-config <config-file>`: Specify the config path, by default `./config.json`

## Config

The configuration file if a json file with the same fields and data specified on the [pion webrtc documentation](https://pkg.go.dev/github.com/pion/webrtc/v3#Configuration)

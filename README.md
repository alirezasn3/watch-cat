# WatchCat

WatchCat is a tool for monitoring network latency by pinging specified IP addresses and providing a web interface to view the results. It can be run as a standalone application or installed as a systemd service for continuous monitoring. The application is configured via a config.json file

## Installation

### Download the binary

Download the latest release binary for Linux using the following command:

```bash
curl -L https://github.com/alirezasn3/watchcat/releases/latest/download/watchcat_linux_amd64 -o watchcat
```

### Make the downloaded binary executable

```bash
chmod +x watchcat
```

### Create config file

Create a config.json file in the same directory as the executable with the following structure:

```json
{
  "listenAddress": "0.0.0.0",
  "destinations": ["8.8.8.8", "1.1.1.1"],
  "monitorAddress": ":8080"
}
```

1. listenAddress:
   • Type: String
   • Description: This field specifies the IP address that the ICMP (ping) packets will be sent from. In the provided example, "0.0.0.0" means that the application will listen on all available network interfaces.
2. destinations:
   • Type: Array of Strings
   • Description: This field contains a list of IP addresses that the application will ping. Each IP address in the array represents a destination that the WatchCat tool will monitor for network latency.
3. monitorAddress:
   • Type: String
   • Description: This field specifies the address and port on which the web server will listen for incoming HTTP requests. In the provided example, ":8080" means that the server will listen on port 8080 on all available network interfaces.

## Running WatchCat

To run WatchCat, simply execute the binary:

```bash
watchcat
```

## Installing as a systemd service

To install WatchCat as a systemd service:

```bash
watchcat --install
```

This will create a systemd service named "watchcat" that will start automatically on system boot.

## Uninstalling the systemd service

To remove the WatchCat systemd service:

```bash
watchcat --uninstall
```

## Accessing the Web Interface

Once WatchCat is running, you can access the web interface by opening a web browser and navigating to http://localhost:8080. Replace localhost with the appropriate IP address if accessing from a different machine.

# mcrouter

`mcrouter` is a Go-based application that provides a proxy server for Minecraft and SSH connections. It allows for domain-based routing and supports features like proxy protocol, IP banning, and domain whitelisting/blacklisting.

## Features

- **SSH and Minecraft Proxy**: Listens for SSH and Minecraft connections and routes them based on domain patterns.
- **Domain-based Routing**: Supports complex domain matching patterns for routing.
- **Proxy Protocol**: Optionally enables proxy protocol for connections.
- **IP Banning**: Automatically bans IPs that attempt to connect directly to the Minecraft server.
- **Domain Whitelisting/Blacklisting**: Allows or denies connections based on domain patterns.

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/yourusername/mcrouter.git
    cd mcrouter
    ```

2. Build the project:
    ```sh
    go build
    ```

## Usage

Run the application with the required flags:
```sh
./mcrouter -k /path/to/ssh/private/key -a /path/to/auth/directory
```

### Command Line Options

- `-S, --ssh`: SSH listen address (default: `127.0.0.1:2222`)
- `-M, --minecraft`: Minecraft listen address (default: `127.0.0.1:25565`)
- `-k, --key`: SSH Server private key file (required)
- `-a, --auth`: SSH Server auth directories (default: `users`)
- `-I, --ban-ip`: Ban IP addresses that tried to ping Minecraft server directly
- `-D, --ban-duration`: Ban duration in hours (default: `48`)
- `-R, --rejected`: Log rejected connections
- `-w, --whitelist`: Domain names allowed to connect
- `-b, --blacklist`: Domain names denied to connect

## Configuration

User configurations are stored in YAML files within the auth directory. Each user has a separate YAML file named after their username. Example configuration:

```yaml
password: "userpassword"
authorized_keys:
  - "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAr..."
allowed_bindings:
  - "example.com"
  - "*.example.com"
```

## Docker

### Building the Docker Image

Build the Docker image with:

```sh
docker build -t mcrouter .
```

### Running with Docker

#### Basic Usage

Run the container with default settings:

```sh
docker run -d -p 2222:2222 -p 25565:25565 \
  -v /path/to/keys:/app/keys \
  -v /path/to/users:/app/users \
  --name mcrouter mcrouter
```

#### Advanced Usage

Run with custom configuration using environment variables:

```sh
docker run -d -p 2222:2222 -p 25565:25565 \
  -v /path/to/keys:/app/keys \
  -v /path/to/users:/app/users \
  -e SSH_LISTEN=0.0.0.0:2222 \
  -e MINECRAFT_LISTEN=0.0.0.0:25565 \
  -e SSH_KEY_PATH=/app/keys/id_rsa \
  -e AUTH_DIR=/app/users \
  -e BAN_IP=true \
  -e BAN_DURATION=48 \
  -e LOG_REJECTED=true \
  -e WHITELIST_DOMAINS="example.com *.example.com" \
  -e BLACKLIST_DOMAINS="bad.com *.bad.com" \
  --name mcrouter mcrouter
```

### Environment Variables

The following environment variables can be configured:

- `SSH_LISTEN`: SSH listen address (default: `0.0.0.0:2222`)
- `MINECRAFT_LISTEN`: Minecraft listen address (default: `0.0.0.0:25565`)
- `SSH_KEY_PATH`: Path to SSH server private key (default: `/app/keys/id_rsa`)
- `AUTH_DIR`: Path to authentication directory (default: `/app/users`)
- `BAN_IP`: Whether to ban IPs that try to connect directly (default: `false`)
- `BAN_DURATION`: Ban duration in hours (default: `48`)
- `LOG_REJECTED`: Whether to log rejected connections (default: `false`)
- `WHITELIST_DOMAINS`: Space-separated list of allowed domains
- `BLACKLIST_DOMAINS`: Space-separated list of denied domains

### Volumes

The container uses two volumes for persistent data:

- `/app/keys`: Directory for SSH private keys
- `/app/users`: Directory for user authentication files

### Notes

- An SSH private key will be automatically generated if not found at the specified path
- The auth directory should contain YAML files for user authentication as described in the Configuration section

## Development

### Prerequisites

- Go 1.20 or later

### Building

```sh
go build
```

### Running Tests

```sh
go test ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/fooBar`)
3. Commit your changes (`git commit -am 'Add some fooBar'`)
4. Push to the branch (`git push origin feature/fooBar`)
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

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

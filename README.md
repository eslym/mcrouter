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

For detailed usage instructions and command line options, please refer to the [USAGE.md](USAGE.md) file.

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

For Docker installation and usage instructions, please refer to the [USAGE.md](USAGE.md#docker) file.

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

### GitHub Actions

The repository includes a GitHub Actions workflow for building and publishing the Docker image to Docker Hub. For this workflow to function properly, the following secrets need to be configured in the GitHub repository:

- `DOCKERHUB_USERNAME`: Your Docker Hub username
- `DOCKERHUB_TOKEN`: A Docker Hub access token with permissions to push to the repository

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

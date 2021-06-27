[![Go Report Card](https://goreportcard.com/badge/github.com/pomo-mondreganto/neo)](https://goreportcard.com/report/github.com/pomo-mondreganto/neo)
[![tests](https://github.com/pomo-mondreganto/neo/actions/workflows/tests.yml/badge.svg)](https://github.com/pomo-mondreganto/neo/actions/workflows/tests.yml)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/pomo-mondreganto/neo)

# neo

Client + server for exploit distribution during Attack & Defence CTF competitions.

## Why

Usually during large CTFs, where the regular laptop can't run the exploits on all participants, teams start some cloud
servers to run these exploits on, but the administration and access management can be a pain and wastes time. **Neo**
can solve this problem by doing 2 things:

1. Every player can start an instance of Neo client and attack a proportional part of the whole team pool automatically.

2. Exploit writers don't copy the newly-created exploits to the exploit server, or manage the distribution by hand, but
   rather submit them to the **Neo server** using the same client, and the server does all the work, running the exploit
   on available clients.

## Usage

To start Neo, you'll need to start one server instance, DestructiveFarm and run one or more client instances.

### Farm

Neo uses the exploit farm to acquire the team list and submit the flags. The protocol is the following
(compatible with the [DestructiveFarm](https://github.com/DestructiveVoice/DestructiveFarm)):

- `GET /api/get_config` must return the configuration with keys `FLAG_FORMAT` and `TEAMS`. The first is the regex of the
  flag, and the second is the mapping `map(string -> string)`, where the key is the team name, and the value is the ip.

- `POST /api/post_flags` will receive an array of mappings with keys `flag`, `sploit` and `team`. `sploit` is the
  exploit name for statistics, and `team` is the team name.

Farm password will be passed in `Authorization` and `X-Token` headers, so the protocol is compatible with **
DestructiveFarm**.

### Server

Server coordinates the clients and distributes teams to attack among them. It must have access to the farm, and all
clients must have access to the server, so you might want to start it somewhere with the public IP address.

To start the server:

1. Download the latest server release (`neo_server_...`) from
   the [Releases](https://github.com/pomo-mondreganto/neo/releases)
   page for your platform (64-bit amd linux and macOS are supported).

2. Edit the configuration in `server_config.yml` file. Edit the `grpc_auth_key` (as it's the password required to
   connect to the server), `farm.url` and `farm.password`. You can also add some environment variables for all exploits
   in the `env` section

3. Start the server by simply running `./neo_server`

### Client

Client has a full-featured CLI and the single binary performs all operations required during the CTF. Client is
distributed as a docker image with a lot of useful python packages preinstalled, see the full list in `requirements.txt`
file (located at [client_env/requirements.txt](./client_env/requirements.txt) in the repository).

Download the latest client release (named `neo_{version}.zip`) from the
[Releases](https://github.com/pomo-mondreganto/neo/releases) page. The `start.sh` file starts the docker container with
the environment if one has not already been run and passes all arguments inside. For example, to get a shell inside the
container, one can run

```shell
./start.sh bash
```

The environment also contains neo binary:

```shell
./start.sh neo --help

Neo client

Usage:
  client [command]

Available Commands:
  add         Add an exploit
  broadcast   Run a command on all connected clients
  disable     Disable an exploit by id
  enable      Enable a disabled exploit by id
  help        Help about any command
  info        Print current state
  run         Start Neo client
  single      Run an exploit once on all teams immediately

Flags:
  -c, --config string   config file (default "client_config.yml")
  -h, --help            help for client
      --host string     server host (default "127.0.0.1")
  -v, --verbose         enable debug logging (default true)

Use "client [command] --help" for more information about a command.
```

As you can see, the binary has a nice help message. Each subcommand has a help message too, for example `add`:

```shell
./start.sh neo add --help

Add an exploit

Usage:
  client add [flags]

Flags:
  -d, --dir                 add exploit as a directory
  -e, --endless             mark script as endless
  -h, --help                help for add
      --id string           exploit name
  -i, --interval duration   run interval (default 15s)
  -t, --timeout duration    timeout for a single run (default 15s)

Global Flags:
  -c, --config string   config file (default "client_config.yml")
      --host string     server host (default "127.0.0.1")
  -v, --verbose         enable debug logging (default true)
```

Each exploit is identified by its file name, and if you try to add the same file again, Neo can replace the exploit with
its newer version.

Neo client only has access to the directory where the `start.sh` file is located, so to add a new exploit, you'll need
to put it somewhere next to `start.sh` (`exploits` directory might be a good place).

## Development notice

Neo is very green and was only tested on a few CTFs by our team. Feel free to open issues and contribute in any way.

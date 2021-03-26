# Mole

[![PkgGoDev](https://pkg.go.dev/badge/github.com/penguinpowernz/mole)](https://pkg.go.dev/github.com/penguinpowernz/mole)

A tunneling daemon and client that allows for local and remote port 
forwarding via the SSH protocol using public key authentication. Because
fuck setting up TLS and certificate authorities for inter-server communication.

This acts like a VPN on the port level.

It has a minimal SSH server (moled) that only handles tunneling, not shells,
however the client will work with a standard SSH server.

The client is made to be run as a daemon with a config file to connect to
preconfigured ports at preconfigured servers using preconfigured keys.  It
can also handle ad hoc port forwarding at the command line however.

## Usage

There are two binaries:

* `mole`
* `moled`

The latter is the minimal SSH server.  They both support config file generation:

    mole -g mole.yml
    moled -g mole.yml

They both accept a config file in the argument:

    mole -c mole.yml
    moled -c mole.yml

The server will allow you to override the port number in the config file
in the command line argument:

    moled -p :222

The client can specify the tunnel to run:

    mole -r 3000 -l 3000 -a 192.168.1.100:222 -i ~/.ssh/id_rsa
    mole -r 3000 -l 3000 -a 192.168.1.100:22 -i ~/.ssh/id_rsa             // connect to a normal SSH server
    mole -c mole.yml                                                      // TODO: use the tunnels and private key from config file
    mole -lt 3000:localhost:3000 -a 192.168.1.100:222  -i ~/.ssh/id_rsa
    mole -rt :22:localhost:33066 -a 192.168.1.100:222  -i ~/.ssh/id_rsa   // TODO: remote port forward using the typical SSH format

You can dump the currently connected tunnels by calling kill on the process ID like so: `kill -USR1 <pid>`:

                                     192.168.1.100:222 [                 127.0.0.1:4222 --> 127.0.0.1:4222                 ]
                                     192.168.1.100:222 [                   localhost:80 <-- 172.31.1.1:80                  ]
                                     192.168.1.100:222 [                 localhost:8080 <-- 172.31.1.1:8080                ]
                               jumpbox2.example.com:22 [                   localhost:22 <-- 0.0.0.0:2222                   ]

## Config File

### Server

In here we specify the public keys who are allow to connect to the server and
the host key.  As well as the listen port and if the server should run or not.

    listen_port: :8022
    run_server: true
    authorized_keys:
      - ssh-rsa AAAAB...snip...9xWs7+Dx
    host_key: |
      -----BEGIN RSA PRIVATE KEY-----
      ...snip...
      -----END RSA PRIVATE KEY-----

### Client

In here we have the public and private key for connecting with the server as well
as any tunnels that should be connected.

    ---
    - address: "*"
      private_key: |
          -----BEGIN RSA PRIVATE KEY-----
          ...snip...
          -----END RSA PRIVATE KEY-----
      public_key: ssh-rsa AAAA...snip...JR7btF0hDw== robert@behemoth
    - address: "192.168.1.100:222"
      # no key fields specified so it will use the keys from the address "*" (default)
      # no host key specified so it will ignore host key (INSECURE!!)
      tunnels:
        - local:    "4222"                             # pretend you're running NATS locally by connecting your local port to the remote NATS server
          remote:   "4222"
        - local:    "80"                               # serve your local webserver on the specific interface on the remote host
          remote:   "172.31.1.1:80"
          reverse:  true
        - L:        "172.31.1.1:8080:localhost:8080"   # the same but using the SSH port forward definition
    - address: "jumpbox.example.com:22"
      private_key: |
          -----BEGIN RSA PRIVATE KEY-----
          ...snip...
          -----END RSA PRIVATE KEY-----
      public_key: ssh-rsa AAAA...snip...JR7btF0hDw== robert@behemoth
      host_key: ssh-rsa ZZZZ...snip...65ASdw0AWsfa==
      tunnels:
        - local:    22                     # poor mans dyndns, turn your cloud server into a jumpbox for your home machine
          remote:   0.0.0.0:2222
          reverse:  true
          disabled: true
        - R: "0.0.0.0:2222:localhost:22"   # poor mans dyndns, but using the reverse port forward definition

So in order to connect the client to a normal SSH server, simply copy your public key
into your `~/.ssh/authorized_keys` file on that server.

## Todo

- [ ] add debian package for armhf
- [ ] add debian package for amd64
- [x] simplify client config
- [ ] interactive acceptance on the remote side, saves host key on the local side
- [x] allow using config file in the mole client
- [x] specify a tunnel with the standard SSH format (e.g. `3344:localhost:3301`)
- [x] specify a tunnel with the standard SSH format at command line (e.g. `3344:localhost:3301`)
- [x] for single port forward command, use default SSH key if none specified (e.g. `~/.ssh/id_rsa`)
- [ ] clean up logging
- [x] add interactive client acceptance
- [x] add host key checking for clients
- [ ] allow generating server config in a client config file and vice versa
- [x] add remote port forwarding
- [x] some kind of statistics or status for the client
- [ ] some kind of statistics or status for the server
- [ ] test that gateway ports actually work by specifying 0.0.0.0 as the bind address
- [ ] use moled to configure the local users `~/.ssh` directory
- [ ] add some persistent retrying for temporary connectivity issues
- [ ] allow changing the user instead of just getting process user

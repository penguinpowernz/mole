# Mole

[![PkgGoDev](https://pkg.go.dev/badge/github.com/penguinpowernz/mole)](https://pkg.go.dev/github.com/penguinpowernz/mole)

A tunneling server that allows for local port forwarding via the SSH
protocol using public key authentication. It has a minimal tunnel 
server that only does tunneling. There is also a tunneling client that
is compatible with normal SSH servers, so the server daemon is not
neccesarily needed.  Designed to be run as a long running service under
systemd or whatever.

## Usage

There are two binaries:

* `mole`
* `moled`

The latter is the daemon.  They both support config file generation:

    mole -g mole.yml
    moled -g mole.yml

They both accept a config file in the argument:

    mole -c mole.yml
    moled -c mole.yml

The server will allow you to override the port number in the config file
in the command line argument:

    moled -p :8023

The client can specify the tunnel to run:

    mole -r 3000 -l 3000 -a 192.168.1.100:8022 -i ~/.ssh/id_rsa
    mole -r 3000 -l 3000 -a 192.168.1.100:22 -i ~/.ssh/id_rsa              // connect to a normal SSH server
    mole -c mole.yml                                                       // TODO: use the tunnels and private key from config file
    mole -lt 3000:localhost:3000 -a 192.168.1.100:8022  -i ~/.ssh/id_rsa   // TODO: local port forward using the typical SSH format
    mole -rt :22:localhost:33066 -a 192.168.1.100:8022  -i ~/.ssh/id_rsa   // TODO: remote port forward using the typical SSH format

## Config File

### Server

In here we specify the public keys who are allow to connect to the server and
the host key.  As well as the listen port and if the server should run or not.

    listen_port: :8022
    run_server: true
    authorized_keys:
      - |
      ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDYC6xU+lOMFRxcCxxJMRiZy2RH28boCy7br6nrNn6TDXcPN972BVWkJMFBiBANSlZuzLcMxtk/PlYz56eejaCft77bRQTjGNiJDbda3ncoJB/umBQiJ  +dTN7iEopaGJ3+Uv/ukC+kgKJGLIGa9698KJHLKJGTLX0ITV2uTKxvFFKWRNgHIPu569Vj/XSsj/+9ww5c4ksal/OVIZ6WpcoIgjNGBr7cspmMJASGrTeDVGFbNiU2kULrqJLZl37t6SinKk4DlodrOjaSKsa/  B2ty19n/iAQx+PpxL8VrWnme/IuLByq2mxQRirWoQrHhHt9xWs7+Dx
    host_key: |
      -----BEGIN RSA PRIVATE KEY-----
      MIICXgIBAAKBgQC+SBJnM94o7iKLep5+h8mLSDazpRasRRiE7zLYBFP9Ea9dTewX
      ZifvikTxE4wST5aid9V1gE72q0Gbc4fERZXJeQ3rQBJiKYdeUKq6WG2QY6uYx91i
      +Vkq9uJb46kxicIf4MEpI4qrFNubzw6HBAJ8soNv85ZOJMe1RVFnkq7uRwIDAQAB
      AoGAP6JgrSzWZf/Fg7m9GXmVuEOtL4TNQU1WNta7vSwtXlu0ttJhWy3pux0Vkz3D
      QThmmuzScRo4zhtVtIP9anEO9x2+1FEUxDc/aFn+lDgkRk/matJcMtPoaAgjdnh+
      m2MHKa8ytS9g6606iLA6iTiEPvARwHUZPjfQj2i2UaR4XyECQQD9iDrfaY7SWH2u
      EqQBJVzeJM/Dm6t3AfjyRzv9FanY2BXHJbXRvSOZyQpOs/AWHYAdPuqZbh2KIUWB
      FzFr+8dxAkEAwCI6uL2aAEv6ZcBk2S+ezebQM+Om8e7F5caIncJzo9lc9+2PNh52
      WARgqxQI1hKX6t2QPXYwJdwv9ADnN5/lNwJBAJ/xzrpdPKYE/5zO07qJWLIoZQ5R
      afXVP7mRKQ48GX/cqriNWMwt14TQaPlH2WIKUGWi6JvM9UPMQ63x9NLb73ECQQCL
      Czp735rHhDSd1nIlSvUeBV+/bYyvoSDOfLL5mHOfq/o/4ke13q2+XMyogkMyyRnv
      +pAcKqAFhied6dlqw+hZAkEAj8wSjZgtTy4D0bWLJxDUaHzVacvTOIgly9e6UBqE
      79QCQnOUsj6NNL0Ln5KZuRrfB5Nl7iBaDlxsLegvP3IuSA==
      -----END RSA PRIVATE KEY-----

### Client

In here we have the public and private key for connecting with the server as well
as any tunnels that should be connected.

    private_key: |
      -----BEGIN RSA PRIVATE KEY-----
      MIICWwIBAAKBgQDJPiD1u/3xk3LBl8MxWrnoh8Jgtb15gheLaS4IJeExNcnqHVkC
      o3rXMoN1vErqMafzSCJeON/D7N0t++gXrUZHPlZlP0NpZA7FIjGP3bZCVZI94f28
      MarHGD5FbMN7eBYbt+ztane5B7uFB4i7GPE73hry3p9uyoZIJR7btF0hDwIDAQAB
      AoGAdfA7UMiD4vgO4PYYJuyM14H4oMTh7jwXoFRb7dqFR1nGo7XfXHSCoWuxL2bS
      YL4JN8KmoaGjQiem2DQxqO6bqEqN5+DlpNBByXIofmSFc1Mp/t8nAJO8jFmUGvMF
      8l4JSMo//OyJQCSeeymmL29BLjQ1yH3n7j0xzFzuIvK4zsECQQDVgzjorPRBDAoc
      aokuZq+NJf4AWq6i9vqLianamnv87MagLxB5YnBcpgjQehQKWbajHEb3W+Hlv1yF
      b2btZ82bAkEA8UnaLTnnyyUcrr4eHPgSl5j5KyEOIC89SQlcH5CRLycnbOReATLY
      tLDrw6ixeM4++BoniHRCXkxSfaYtzLErnQJAT9dZIZEDaYuSAFxKXiKiBPsvB3zh
      jykiOanJ7WgVc1grUl0nIO0RrWOdKjBsbA5uQIJjez5Ns/ciJveomqBVfwJADjYf
      V5KViG2DJvejpmkmDy+/XT7xKgweO/MFLgbBxlk0BUHeF4v7H4lcGYYSDd937fz8
      XxkZ35v3L9dd0zSMMQJAIpnhKW4Ggz0kAGJYvGH4Uz4mr/70mExGGlck5vYRuW2z
      Mqw1OD4t3HMyhqvwItcllju9GGTZjdhOZipbzNicSg==
      -----END RSA PRIVATE KEY-----
    public_key: |
      ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDJPiD1u/3xk3LBl8MxWrnoh8Jgtb15gheLaS4IJeExNcnqHVkCo3rXMoN1vErqMafzSCJeON/D7N0t+  +gXrUZHPlZlP0NpZA7FIjGP3bZCVZI94f28MarHGD5FbMN7eBYbt+ztane5B7uFB4i7GPE73hry3p9uyoZIJR7btF0hDw== robert@behemoth
    tunnels:
      - address: 127.0.0.1:8022
        enabled: true
        local_port: "1234"
        remote_port: "12345"
      - address: 127.0.0.1:8022
        enabled: true
        local_port: "1345"
        remote_port: "1234"

So in order to connect the client to a normal SSH server, simply copy your public key
into your `~/.ssh/authorized_keys` file on that server.

## Todo

- [ ] allow using config file in the mole client
- [ ] specify a tunnel with the standard SSH format (e.g. `3344:localhost:3301`)
- [x] for single port forward command, use default SSH key if none specified (e.g. `~/.ssh/id_rsa`)
- [ ] clean up logging
- [ ] add interactive client acceptance
- [ ] add host key checking for clients
- [ ] allow generating server config in a client config file and vice versa
- [ ] add remote port forwarding
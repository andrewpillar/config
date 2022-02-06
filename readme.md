# Config

>**Note:** This library is still a work in progress, and is very rudimentary
in its implementation. As of now the library does not support folding of
parameter names for example, contrary to what the code examples may imply.

Config is a library for working with structured configuration files in Go. This
library defines its own structured configuration language which allows for the
organization of configuration into *parameters* and *blocks*. Below is an
example,

    $ cat server.conf
    net {
        listen "localhost:443"

        tls {
            cert "/var/lib/ssl/server.crt"
            key  "/var/lib/ssl/server.key"
        }
    }

    database "/var/lib/db.sqlite"

    store files {
        path  "/var/lib/files"

        # Support for sizes using the suffix B, KB, MB, GB, TB, PB, EB, and ZB.
        limit 50MB
    }

then, to make use of the above file in your application you would write,

    package main

    import (
        "fmt"
        "os"

        "github.com/andrewpillar/config"
    )

    type Config struct {
        Net struct {
            Listen string

            TLS struct {
                Cert string
                Key  string
            }
        }

        Database string

        Store map[string]struct {
            Path  string
            Limit int64
        }
    }

    // errh is used for handling any errors that may arise during parsing of
    // configuration file.
    func errh(pos config.Pos, msg string) {
        fmt.Fprintf(os.Stderr, "%s - %s\n", pos, msg)
    }

    func main() {
        var cfg Config

        if err := config.Decode(&cfg, "server.conf", errh); err != nil {
            fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
            os.Exit(1)
        }
    }

The configuration language is a plain text file containing a list of parameters.
Comments in the language start with `#` and end with a newline.

    # Full-line comment.
    log "/dev/stdout" # Inline comment.

A parameter is just a named value, where the value is either a literal, an
array, or a block. A literal can be one of the following,

* `string` - A sequence of bytes wrapped between a pair of `"`.
* `int` - A numeric value that is a whole number.
* `float` - A numeric value that is a float.
* `bool` - A value that is either `true` or `false`.
* `duration` - A numeric value suffixed with with `s`, `m`, `h`, or `d` for
second, minue, hour, or day respectively. This will be converted to a
`time.Duration` when decoded.
* `size` - A numeric value suffixed with `B`, `KB`, `MB`, `GB`, `TB`, `PB`,
`EB`, or `ZB`. This will be converted to `int64` when decoded.

an array is a list of values wrapped between a pair of `[ ]`,

    strings ["one", "two", "three"]
    numbers [1, 2, 3]

a block is a list of parameters wrapped between a pair of `{ }`,

    block {
        string "string"
        number 10
        duration 10h
        size 10GB

        array [true, false]

        block2 {
            # More parameters here.
        }
    }

parameters can be labeled. Labelling a parameter allows for multiple
parameters to be grouped together. For example, assume your application allows
authenticating against multiple authentication backends, then you may want
them to group them by the backend,

    auth ldap {
        addr "ldap://example.com"

        tls {
            cert "/var/lib/ssl/client.crt"
            key  "/var/lib/ssl/client.key"
        }
    }

    auth internal {
        addr "postgres://localhost:5432/db"
    }

    auth saml {
        addr "https://idp.example.com"

        tls {
            ca "/var/lib/ssl/ca.crt"
        }
    }

then in your Go code you would define a map of structs where the key would
be the mechanism,

    type Config struct {
        Auth map[string]struct {
            Addr string

            TLS struct {
                CA   string
                Cert string
                Key  string
            }
        }
    }

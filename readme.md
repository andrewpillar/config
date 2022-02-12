# Config

* [Overview](#overview)
* [Syntax](#syntax)
  * [Comments](#comments)
  * [String](#string)
  * [Number](#number)
  * [Bool](#bool)
  * [Duration](#duration)
  * [Size](#size)
  * [Array](#array)
  * [Block](#block)
  * [Label](#label)

Config is a library for working with structured configuration files in Go. This
library defines its own minimal structured configuration language.

## Overview

The language organizes configuration into a list of parameters. Below is an
example,

    # Example configuration file.

    net {
        listen ":https"

        tls {
            cert "/var/lib/ssl/server.crt"
            key  "/var/lib/ssl/server.key"

            ciphers ["AES-128SHA256", "AES-256SHA256"]
        }
    }

    log access {
        level "info"
        file  "/var/log/http/access.log"
    }

    body_limt 50MB

    timeout {
        read  10m
        write 10m
    }

The above file would then be decoded like so in your Go program,

    package main

    import (
        "fmt"
        "os"
        "time"

        "github.com/andrewpillar/config"
    )

    type Config struct {
        Net struct {
            Listen string

            TLS struct {
                Cert    string
                Key     string
                Ciphers []string
            }
        }

        Log map[string] struct {
            Level string
            File  string
       }

        BodyLimit int64 `config:"body_limit"`

        Timeout struct {
            Read  time.Duration
            Write time.Duration
        }
    }

    func main() {
        var cfg Config

        if err := config.DecodeFile(&cfg, "server.conf"); err != nil {
            fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
            os.Exit(1)
        }
    }

## Syntax

A configuration file is a plain text file with a list of parameters and their
values. The value of a parameter can either be a literal, array, or a parameter
block. Typically, the filename should be suffixed with the `.conf` file
extension.

### Comments

Comments start with `#` and end with a newline. This can either be on a full
line, or inlined.

    # Full-line comment.
    temp 0.5 # Inline comment.

### String

A string is a sequence of bytes wrapped between a pair of `"`. As of now, string
literals are limited in their capability. 

    string  "this is a string literal"
    string2 "this is another \"string\" literal, with escapes"

### Number

Integers and floats are supported. Integers are decoded into the `int64` type,
and floats into the `float64` type.

    int   10
    float 10.25

### Bool

A bool is a `true` or `false` value.

    bool  true
    bool2 false

### Duration

Duration is a duration of time. This is a number literal suffixed with either
`s`, `m`, or `h`, for second, minute, or hour respectively. Duration is decoded
into the `time.Duration` type.

    seconds 10s
    minutes 10m
    hours   10h

The duration units can also be combined for more explicit values,

    hour_half 1h30m

### Size

Size is the amount of bytes. This is a number literal suffixed with the unit,
either `B`, `KB`, `MB`, `GB`, or `TB`. Size is decoded into the `int64` type.

    byte     1B
    kilobyte 1KB
    megabyte 1MB
    gigabyte 1GB
    terabyte 1TB

### Array

An array is a list of values, these can either be literals, or blocks wrapped
in a pair of `[ ]`. Arrays are decoded into a slice of the respective type.

    strings ["str", "str2", "str3"]
    numbers [1, 2, 3, 4]

    arrays [
        [1, 2, 3],
        [4, 5, 6],
        [7, 8, 9],
    ]

    blocks [{
        x 1
        y 2
        z 3
    }, {
        x 4
        y 5
        z 6
    }, {
        x 7
        y 8
        z 9
    ]]

### Block

A block is a list of parameters wrapped between a pair of `{ }`. Blocks are
decoded into a struct.

    block {
        param "value"

        block2 {
            param 10
        }
    }

### Label

A label can be used to distinguish between parameters of the same name. This
can be useful when you have similar configuration parameters that you want to
distinguish between. A labelled parameter is decoded into a map, where the
key of the map is a string, the label itself, and the value of the map is
the type for the parameter.

    auth ldap {
        addr "ldap://example.com"

        tls {
            cert "/var/lib/ssl/client.crt"
            key  "/var/lib/ssl/client.key"
        }
    }

    auth saml {
        addr "https://idp.example.com"

        tls {
            ca "/var/lib/ssl/ca.crt"
        }
    }

Labels aren't just limited to blocks, they can be applied to any other
parameter type,

    ports open ["8080", "8443"]

    ports close ["80", "443"]

# Config

* [Overview](#overview)
* [Options](#options)
  * [Error handling](#error-handling)
  * [Environment variables](#environment-variables)
  * [Includes](#includes)
* [Struct tags](#struct-tags)
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

        Log map[string]struct {
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

## Options

Options can be used to configure how a file is decoded. These are callbacks that
can be passed to either the `NewDecoder` function or the `DecodeFile` function.

### Error handling

A custom error handler can be configured via the `ErrorHandler` option. This
takes a `func(pos Pos, msg string)` callback, which is called when an error
occurs during parsing of a file. This is given the position at which the error
occurred, and the message. If no handler is configured, then the `Stderrh`
error handler is used by default.

    config.DecodeFile(&cfg, "file.conf", config.ErrorHandler(customHandler))

### Environment variables

Environment variables can be supported via the `Envvars` option. This will
expand any `${VARIABLE}` that is found in a string literal in the configuration
file into the respective environment variable.

    config.DecodeFile(&cfg, "file.conf", config.Envvars)

### Includes

Includes can be configured via the `Includes` option. This will support the
inclusion of configuration files via the `include` parameter.

    config.DecodeFile(&cfg, "file.conf", config.Includes)


This expects to be given either a string literal or an array of string literals
for the file(s) to include,

    include "database.conf"

    include [
        "database.conf",
        "smtp.conf",
    ]

## Struct tags

The decoding of each parameter can be configured via the `config` struct field
tag. The name of the tag specifies the parameter to map the field to, and the
subsequent comma separated list are additional options.

The `deprecated` option marks a field as deprecated. This will emit an error
to the error handler during decoding if the deprecated parameter is encountered.
For example, assume you have an `ssl` configuration block that you want to
deprecate, you would do the following,

    type TLSConfig struct {
        CA   string
        Cert string
        Key  string
    }

    type Config struct {
        TLS TLSConfig
        SSL TLSConfig `config:"ssl,deprecated"`
    }

to specify the parameter that should replace the `ssl` parameter you would
separate the name with a `:` in the option,

    type Config struct {
        TLS TLSConfig
        SSL TLSConfig `config:"ssl,deprecated:tls"`
    }

The `nogroup` option prevents the grouping of labelled parameters into a map.
This would be used in an instance where you want more explicit control over
how labelled parameters are decoded. For example, consider the following
configuration,

    store sftp {
        addr "sftp.example.com"

        auth {
            username "sftp"
            identity "/var/lib/ssh/id_rsa"
        }
    }

    store disk {
        path "/var/lib/files"
    }

this defines two `store` blocks that are labelled. Both blocks vary with the
parameters that they offer. We can decode the above into the below struct,

    type Config struct {
        Store struct {
            SFTP struct {
                Addr string

                Auth struct {
                    Username string
                    Identity string
                }
            }

            Disk struct {
                Path string
            }
        } `config:",nogroup"`
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
the type for the parameter. This is not the case if the `nogroup` parameter is
specified, in which case the label itself will be mapped to a field in a struct.

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

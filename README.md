# ps-http-sim

`ps-http-sim` stands in as a proxy in front of a normal MySQL database, and provides a PlanetScale HTTP API compatibility layer.

At the moment, this is not an official PlanetScale project, and is in early stage development and quality. Please do not use this in production environments.

This is intended to aid when adopting `@planetscale/database` in your application and wanting to run against a local database.

# Usage

```
$ ps-http-sim -help
Usage of ps-http-sim:
  -listen-addr string
        HTTP server address (default "127.0.0.1")
  -listen-port uint
        HTTP server port (default 8080)
  -mysql-addr string
        MySQL address (default "127.0.0.1")
  -mysql-dbname string
        MySQL database to connect to (default "mysql")
  -mysql-idle-timeout duration
        MySQL connection idle timeout (default 10s)
  -mysql-max-rows uint
        Max rows for a single query result set (default 1000)
  -mysql-no-pass
        Don't use password for MySQL connection
  -mysql-port uint
        MySQL port (default 3306)
```

There is an example Node configuration in the `example/` folder.

A sample `mysqld` docker container can be run by doing:

```
$ make run-mysql
```

If you'd prefer not to run a binary directly, a docker container can be built with:

```
$ make docker
```

I'm not good at docker, so if this is bad or any suggestions, I'm open to anything.

## Authentication

The authentication you configure in your database-js application, is passed along as-is to your local MySQL database. So you'd want to match up the authentication to match what you'd with a normal MySQL client.

If your database is configured to not use a password, the `-mysql-no-pass` flag must be set on `ps-http-sim`, but database-js must be configured still to send a password. The password may be anything except blank.

This is a bit of a "wart" in that the PlanetScale API would fail if not being presented with a password at all since it's impossible to have a PlanetScale database without a password.
